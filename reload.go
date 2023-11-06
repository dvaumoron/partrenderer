/*
 *
 * Copyright 2023 partrenderer authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package partrenderer

import (
	"io/fs"
	"path/filepath"
	"text/template"

	"github.com/spf13/afero"
)

type viewManager struct {
	views        map[string]*template.Template
	reloadSender chan<- chan<- error
}

func newViewManager(views map[string]*template.Template, infos loadInfos) *viewManager {
	vm := &viewManager{views: views}
	reloadChan := make(chan chan<- error)
	go manageReload(reloadChan, infos, vm)
	vm.reloadSender = reloadChan
	return vm
}

func manageReload(reloadReceiver <-chan chan<- error, infos loadInfos, vm *viewManager) {
	var waitings []chan<- error
	loadingEnded := make(chan error)
	for {
		select {
		case responder := <-reloadReceiver:
			if len(waitings) == 0 {
				go reloadAndAlert(infos, vm, loadingEnded)
			}
			waitings = append(waitings, responder)
		case err := <-loadingEnded:
			for _, responder := range waitings {
				responder <- err
			}
			waitings = waitings[:0]
		}
	}
}

func reloadAndAlert(infos loadInfos, vm *viewManager, endSender chan<- error) {
	views, err := infos.loadViews()
	if err == nil {
		vm.views = views
	}
	endSender <- err
}

func (vm *viewManager) get(viewName string) (*template.Template, error) {
	view, ok := vm.views[viewName]
	if !ok {
		return nil, ErrViewNotFound
	}
	return view, nil
}

func (vm *viewManager) reload() error {
	ended := make(chan error)
	vm.reloadSender <- ended
	return <-ended
}

type loadInfos struct {
	fs             afero.Fs
	componentsPath string
	viewsPath      string
	fileExt        string
	fileExtLen     int
	funcs          template.FuncMap
	reloadRule     ReloadRule
}

func (options loadInfos) init() (loadInfos, error) {
	var err error
	if options.viewsPath, err = filepath.Abs(options.viewsPath); err != nil {
		return options, err
	}
	if last := len(options.viewsPath) - 1; options.viewsPath[last] != '/' {
		options.viewsPath += "/"
	}
	return options, nil
}

func (options loadInfos) loadViews() (map[string]*template.Template, error) {
	components := template.New("").Funcs(options.funcs)
	err := afero.Walk(options.fs, options.componentsPath, func(path string, fi fs.FileInfo, err error) error {
		if err == nil && !fi.IsDir() && path[len(path)-options.fileExtLen:] == options.fileExt {
			err = parseOne(options.fs, path, components)
		}
		return err
	})
	if err != nil {
		return nil, err
	}

	inSize := len(options.viewsPath)
	views := map[string]*template.Template{}
	err = afero.Walk(options.fs, options.viewsPath, func(path string, fi fs.FileInfo, err error) error {
		if end := len(path) - options.fileExtLen; err == nil && !fi.IsDir() && path[end:] == options.fileExt {
			t, _ := components.Clone() // here error is always nil
			err = parseOne(options.fs, path, t)
			views[path[inSize:end]] = t
		}
		return err
	})
	// not supposed to return data on error, but it's a private function
	return views, err
}

func parseOne(fs afero.Fs, path string, tmpl *template.Template) error {
	data, err := afero.ReadFile(fs, path)
	if err == nil {
		_, err = tmpl.New(path).Parse(string(data))
	}
	return err
}
