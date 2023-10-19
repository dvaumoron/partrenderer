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
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const (
	defaultSeparator = "#"
	defaultRootName  = "root"
)

type PartRenderer struct {
	views     map[string]*template.Template
	Separator string
	RootName  string
}

func MakePartRenderer(componentsPath string, viewsPath string, fileExt string, funcs template.FuncMap) (PartRenderer, error) {
	fileExt = cleanExt(fileExt)

	components, err := loadComponents(componentsPath, fileExt, funcs)
	if err != nil {
		return PartRenderer{}, err
	}

	views, err := loadViews(viewsPath, fileExt, components)
	if err != nil {
		return PartRenderer{}, err
	}
	return PartRenderer{views: views, Separator: defaultSeparator, RootName: defaultRootName}, nil
}

func (r PartRenderer) ExecuteTemplate(w io.Writer, viewName string, data any) error {
	if splitted := strings.Split(viewName, r.Separator); len(splitted) > 1 {
		return r.views[splitted[0]].ExecuteTemplate(w, splitted[1], data)
	} else {
		return r.views[viewName].ExecuteTemplate(w, r.RootName, data)
	}
}

func loadComponents(componentsPath string, fileExt string, funcs template.FuncMap) (*template.Template, error) {
	var filepaths []string
	fileExtLen := len(fileExt)
	err := filepath.WalkDir(componentsPath, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if path[len(path)-fileExtLen:] == fileExt {
				filepaths = append(filepaths, path)
			}
		}
		return err
	})

	if err != nil {
		return nil, err
	}
	return template.New("").Funcs(funcs).ParseFiles(filepaths...)
}

func loadViews(viewsPath string, fileExt string, components *template.Template) (map[string]*template.Template, error) {
	viewsPath, err := filepath.Abs(viewsPath)
	if err != nil {
		return nil, err
	}
	if last := len(viewsPath) - 1; viewsPath[last] != '/' {
		viewsPath += "/"
	}

	inSize := len(viewsPath)
	fileExtLen := len(fileExt)
	views := map[string]*template.Template{}
	err = filepath.WalkDir(viewsPath, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if end := len(path) - fileExtLen; path[end:] == fileExt {
				var t *template.Template
				if t, err = components.Clone(); err == nil {
					var data []byte
					if data, err = os.ReadFile(path); err == nil {
						_, err = t.Parse(string(data))
						views[path[inSize:end]] = t
					}
				}
			}
		}
		return err
	})
	// not supposed to return data on error, but it's a private function
	return views, err
}

// ensure start with a dot
func cleanExt(ext string) string {
	if ext != "" && ext[0] != '.' {
		ext = "." + ext
	}
	return ext
}
