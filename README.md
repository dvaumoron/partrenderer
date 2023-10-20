# PartRenderer

Library to load several [go template](https://pkg.go.dev/text/template) decomposed in multiple file and render them globally or partially (useful with [htmx](https://htmx.org/)).

## Getting started

In order to use PartRenderer in your project Cornucopia (with the go langage already installed), you can use the command :

    go install github.com/dvaumoron/partrenderer@latest

Then you can import it :

```Go
import "github.com/dvaumoron/partrenderer"
```

And use it in two step :

```Go
// parse templates
renderer, err := partrenderer.MakePartRenderer(componentsPath, viewsPath, fileExt, customFuncs)
// and use them
err = renderer.ExecuteTemplate(writer, viewName, data)
```

The first call with :

- componentsPath indicate a directory to walk in order to load all component templates
- viewsPath indicate a directory to walk in order to load all view templates (which can see components)
- fileExt can be ".html" (filter readed files, a value without a starting dot have one added automatically)
- customFuncs is a [FuncMap](https://pkg.go.dev/text/template#FuncMap) to register your custom template functions

The second call has the same signature as [Template.ExecuteTemplate](https://pkg.go.dev/text/template#Template.ExecuteTemplate) where viewName has no extention ("hello/index" for an index.html file in hello folder) and can have a part selector (like in "hello/index#body", without this selector "root" is used).

With componentsPath/main.html like :

```html
{{define "root"}}
    <html>
        <head>
            <meta charset="utf-8"/>
            {{template "header" .}}
        </head>
        <body>
             {{template "body" .}}
        </body>
</html>
{{end}}
```

And viewsPath/hello/index.html like :

```html
{{define "header"}}
    <title>Hello World</title>
{{end}}
{{define "body"}}
    <h1 class="greetings">Hello World</h1>
{{end}}
```

See advanced examples of [componentsPath](https://github.com/dvaumoron/puzzletest/tree/main/templatedata/templates/components) and [viewsPath](https://github.com/dvaumoron/puzzletest/tree/main/templatedata/templates/views) templates.
