package util

import (
	"html/template"
	"net/http"
	"io"
)

type HttpWriter = http.ResponseWriter
type HttpReq = *http.Request
type Plugin = func(w HttpWriter, r HttpReq, info map[string]any) (render bool, addinfo any)
// TODO: add "terminator" flag to plugin
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
	"html/index.gohtml", nil, {GOTM_acc},
)
*/

// Content Server Creators
// Static(File|Component)
// (Dynamic|Templated(Logiced)?)(Plugged)?(Page|Component)

// NOTE if the Content Server has a render engine, and info map
// (map[string]any) can be provided (or may be nil). That info map will be
// passed to the render engine as the root of information

// Plugged defines if there should be an []GOTMPlugin to run before rendering.
// NOTE GOTMPlugin can read/write to requests, read/write to the info map, stop
// execution of futher plugins and disabled rendering for a request

// Templated attaches a Template file to the Content Server
// NOTE /templates/*.gohtml will be parsed before the template file
// Logiced attaches a user defined function to the end of the PluginList
// NOTE Logiced may be used even if Plugged is no specified

// File, Component, Page don't modify the content server.
// However, recomended usage for
// - Page is to ACT upon some information from the client (and possibly render it).
// - Component is to render information from the server.
// - File is to to serve as is.
// TL;DR
// - Page: side-effects and render engine
// - Component: render engine
// - File: info as if

type ContentServer = http.Handler

func StaticFile(filename string) ContentServer {
	return staticServer{filename}
}

type staticServer struct {
	Filename string
}

type TemplatedComponent struct {
	Template *template.Template
}

type TemplatedPage struct {
	Template *template.Template
	Info map[string]any
	Plugins []GOTMPlugin
}

// TODO DynamicPage
// LogicedPage without template
type LogicedPage struct {
	Template *template.Template
	Info map[string]any
	Plugins []GOTMPlugin
	Fn Plugin
}

func (s staticServer) ServeHTTP (w HttpWriter, r HttpReq) {
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

func LogicPage(
	filename string,
	info map[string]any,
	plugins []GOTMPlugin,
	fn Plugin,
) (TemplatedPage) {
	if info == nil {
		info = make(map[string]any)
	}

	tmpl := template.Must(
		template.Must(
			template.ParseFiles(filename),
		).ParseGlob("templates/*.gohtml"),
	)

	return TemplatedPage{
		tmpl, info, append(plugins, GOTMPlugin{"logic", fn}),
	}
}

func Component (
	filename string,
) TemplatedComponent {
	tmpl := template.Must(
		template.Must(
			template.ParseFiles(filename),
		).ParseGlob("templates/*.gohtml"),
	)
	return TemplatedComponent{tmpl}
}

func (Tc TemplatedComponent) Render(w io.Writer, einfo any) {
	Tc.Template.Execute(w, einfo)
}

func (Tc TemplatedComponent) RenderString(einfo any) (s string) {
	panic("NOT IMPLEMENTED")
	return ""
}

