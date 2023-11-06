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
	"errors"
	"io"
	"strings"
	"text/template"

	"github.com/spf13/afero"
)

const (
	defaultExt       = ".html"
	defaultExtLen    = len(defaultExt)
	defaultSeparator = "#"
	defaultRootName  = "root"
)

var ErrViewNotFound = errors.New("view not found")

// a true trigger a reload
type ReloadRule = func(error) bool

func AlwaysReload(err error) bool {
	return true
}

func ReloadOnViewNotFound(err error) bool {
	return err == ErrViewNotFound
}

func NeverReload(err error) bool {
	return false
}

type LoadOption func(loadInfos) loadInfos

// option to use an alternate file system
func WithFs(fs afero.Fs) LoadOption {
	return func(li loadInfos) loadInfos {
		li.fs = fs
		return li
	}
}

// option to use an alternate extension to filter loaded file (default is ".html")
func WithFileExt(ext string) LoadOption {
	return func(li loadInfos) loadInfos {
		if ext != "" && ext[0] != '.' {
			ext = "." + ext
		}
		li.fileExt = ext
		li.fileExtLen = len(ext)
		return li
	}
}

// allow to load a template.FuncMap before parsing the go templates
func WithFuncs(customFuncs template.FuncMap) LoadOption {
	return func(li loadInfos) loadInfos {
		li.funcs = customFuncs
		return li
	}
}

// option to change the rule to reload on error (default is ReloadOnViewNotFound)
func WithReloadRule(rule ReloadRule) LoadOption {
	return func(li loadInfos) loadInfos {
		li.reloadRule = rule
		return li
	}
}

type PartRenderer struct {
	views      *viewManager
	reloadRule ReloadRule
	Separator  string
	RootName   string
}

// The componentsPath argument indicates a directory to walk in order to load all component templates
//
// The viewsPath argument indicates a  directory to walk in order to load all view templates (which can see components)
func MakePartRenderer(componentsPath string, viewsPath string, opts ...LoadOption) (PartRenderer, error) {
	infos := loadInfos{
		fs:             afero.NewOsFs(),
		componentsPath: componentsPath,
		viewsPath:      viewsPath,
		fileExt:        defaultExt,
		fileExtLen:     defaultExtLen,
		reloadRule:     ReloadOnViewNotFound,
	}

	for _, optionModifier := range opts {
		infos = optionModifier(infos)
	}

	infos, err := infos.init()
	if err != nil {
		return PartRenderer{}, err
	}

	views, err := infos.loadViews()
	if err != nil {
		return PartRenderer{}, err
	}

	vm := newViewManager(views, infos)
	return PartRenderer{views: vm, reloadRule: infos.reloadRule, Separator: defaultSeparator, RootName: defaultRootName}, nil
}

// Find a template and render it, global and partial rendering depend on PartRenderer.RootName and PartRenderer.Separator.
// Could try a reload on error depending on the ReloadRule option.
func (r PartRenderer) ExecuteTemplate(w io.Writer, viewName string, data any) error {
	partName := r.RootName
	if splitted := strings.Split(viewName, r.Separator); len(splitted) > 1 {
		viewName, partName = splitted[0], splitted[1]
	}

	err := r.innerExecuteTemplate(w, viewName, partName, data)
	if err != nil && r.reloadRule(err) {
		if err = r.views.reload(); err == nil {
			err = r.innerExecuteTemplate(w, viewName, partName, data)
		}
	}
	return err
}

func (r PartRenderer) innerExecuteTemplate(w io.Writer, viewName string, partName string, data any) error {
	view, err := r.views.get(viewName)
	if err != nil {
		return err
	}
	return view.ExecuteTemplate(w, partName, data)
}
