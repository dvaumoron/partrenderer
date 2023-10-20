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
	"io"
	"io/fs"
	"path/filepath"
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

type loadOptions struct {
	fs         afero.Fs
	fileExt    string
	fileExtLen int
	funcs      template.FuncMap
}

type LoadOption func(loadOptions) loadOptions

func WithFs(fs afero.Fs) LoadOption {
	return func(lo loadOptions) loadOptions {
		lo.fs = fs
		return lo
	}
}

func WithFileExt(ext string) LoadOption {
	return func(lo loadOptions) loadOptions {
		if ext != "" && ext[0] != '.' {
			ext = "." + ext
		}
		lo.fileExt = ext
		lo.fileExtLen = len(ext)
		return lo
	}
}

func WithFuncs(customFuncs template.FuncMap) LoadOption {
	return func(lo loadOptions) loadOptions {
		lo.funcs = customFuncs
		return lo
	}
}

type PartRenderer struct {
	views     map[string]*template.Template
	Separator string
	RootName  string
}

func MakePartRenderer(componentsPath string, viewsPath string, opts ...LoadOption) (PartRenderer, error) {
	options := loadOptions{fs: afero.NewOsFs(), fileExt: defaultExt, fileExtLen: defaultExtLen}
	for _, optionModifier := range opts {
		options = optionModifier(options)
	}

	components, err := loadComponents(componentsPath, options)
	if err != nil {
		return PartRenderer{}, err
	}

	views, err := loadViews(viewsPath, components, options)
	if err != nil {
		return PartRenderer{}, err
	}
	return PartRenderer{views: views, Separator: defaultSeparator, RootName: defaultRootName}, nil
}

func (r PartRenderer) ExecuteTemplate(w io.Writer, viewName string, data any) error {
	partName := r.RootName
	if splitted := strings.Split(viewName, r.Separator); len(splitted) > 1 {
		viewName, partName = splitted[0], splitted[1]
	}
	return r.views[viewName].ExecuteTemplate(w, partName, data)
}

func loadComponents(componentsPath string, options loadOptions) (*template.Template, error) {
	components := template.New("").Funcs(options.funcs)
	err := afero.Walk(options.fs, componentsPath, func(path string, fi fs.FileInfo, err error) error {
		if err == nil && !fi.IsDir() && path[len(path)-options.fileExtLen:] == options.fileExt {
			err = parseOne(options.fs, path, components)
		}
		return err
	})
	// not supposed to return data on error, but it's a private function
	return components, err
}

func loadViews(viewsPath string, components *template.Template, options loadOptions) (map[string]*template.Template, error) {
	viewsPath, err := filepath.Abs(viewsPath)
	if err != nil {
		return nil, err
	}
	if last := len(viewsPath) - 1; viewsPath[last] != '/' {
		viewsPath += "/"
	}

	inSize := len(viewsPath)
	views := map[string]*template.Template{}
	err = afero.Walk(options.fs, viewsPath, func(path string, fi fs.FileInfo, err error) error {
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
