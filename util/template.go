package util

import (
	"html/template"
	"net/http"
)

type HttpWriter = http.ResponseWriter
type HttpReq = *http.Request
type Plugin = func(w HttpWriter, r HttpReq, info map[string]any) (render bool, addinfo any)
// add "terminator" flag to plugin
// GOTM_mustacc is added as a guard, to relieve the
// : programmer of the duty to check if the user is logged in
// : plugins after GOTM_mustacc shoudln't be executed
type GOTMPlugin struct {
	Name string
	Plug Plugin
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

type StaticFile struct {
	Filename string
}

type TemplatedPage struct {
	Template *template.Template
	Info map[string]any
	Plugins []GOTMPlugin
}

type LogicedPage struct {
	Template *template.Template
	Info map[string]any
	Plugins []GOTMPlugin
	Fn Plugin
}

func (s StaticFile) ServeHTTP (w HttpWriter, r HttpReq) {
	http.ServeFile(w, r, s.Filename)
}

func TemplatePage(filename string, info map[string]any, plugins []GOTMPlugin) TemplatedPage {
	if info == nil {
		info = make(map[string]any)
	}

	tmpl := template.Must(
		template.Must(
			template.ParseFiles(filename),
		).ParseGlob("templates/*.gohtml"),
	)

	return TemplatedPage{
		tmpl, info, plugins,
	}
}

func (s TemplatedPage) ServeHTTP (w HttpWriter, r HttpReq) {
	var render = true
	var prender bool
	for _, plug := range s.Plugins {
		prender, s.Info[plug.Name] = plug.Plug(w, r, s.Info)
		render = render&&prender
	}
	if (render) {
		e := s.Template.Execute(w, s.Info)
		if (e != nil) {
			panic(e)
		}
	}
}

func (s LogicedPage) ServeHTTP (w HttpWriter, r HttpReq) {
	var render = true
	var prender bool
	for _, plug := range s.Plugins {
		prender, s.Info[plug.Name] = plug.Plug(w, r, s.Info)
		render = render&&prender
	}
	prender, s.Info["logic"] = s.Fn(w, r, s.Info)
	render = render&&prender

	if (render) {
		e := s.Template.Execute(w, s.Info)
		if (e != nil) {
			panic(e)
		}
	}
}

func LogicPage(
	filename string,
	info map[string]any,
	plugins []GOTMPlugin,
	fn Plugin,
) (LogicedPage) {
	if info == nil {
		info = make(map[string]any)
	}

	tmpl := template.Must(
		template.Must(
			template.ParseFiles(filename),
		).ParseGlob("templates/*.gohtml"),
	)

	return LogicedPage{
		tmpl, info, plugins, fn,
	}
}

