package util

import (
	"html/template"
	"net/http"
)

type HttpWriter = http.ResponseWriter
type HttpReq = *http.Request
type GOTMPlugin struct {
	Name string
	Plug func(w HttpWriter, r HttpReq, info map[string]any)any
}

/* example

func GOTM_example(w HttpWriter, r HttpReq, info map[string]any) any {
	return 4
}
var GOTM_acc GOTMPlugin = {"acc", GOTM_example}

index := TemplatePage(
	"html/index.gohtml",
	map[string]any{"server name":"my server!"},
	{{"acc", GOTM_acc}, {"view", GOTM_view_counter}},
)

*/

type TemplatedPage struct {
	Template *template.Template
	Info map[string]any
	Plugins []GOTMPlugin
}

func TemplatePage(filename string, info map[string]any, plugins []GOTMPlugin) TemplatedPage {
	if info == nil {
		info = make(map[string]any)
	}
	if info == nil {
		info = make(map[string]any)
	}
	tmpl, e := template.ParseFiles(filename)
	if (e != nil) {panic(e)}
	return TemplatedPage{
		tmpl, info, plugins,
	}
}

func (s TemplatedPage) ServeHTTP (w HttpWriter, r HttpReq) {
	for _, plug := range s.Plugins {
		s.Info[plug.Name] = plug.Plug(w, r, s.Info)
	}

	s.Template.Execute(w, s.Info)
}

