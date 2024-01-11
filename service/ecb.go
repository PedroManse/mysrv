package service

import (
	. "mysrv/util"
	"strconv"
	"html/template"
)

var pastes = NewSyncMap[string, string]()
var ECBEndpoint = TemplatePage(
	"html/ecb/ecb.gohtml", nil,
	[]GOTMPlugin{GOTM_account},
)

var getRender = InlineComponent(`
{{ if eq .ok true }}
	<pre>{{.text}}</pre>
{{ else }}
	<h3>No such Paste "{{.id}}"</h3>
{{ end }}
`)
var postRender = InlineComponent(`
<h3>Paste "{{.id}}" created</h3>
<pre>{{.text}}</pre>
`)


func ECBHandler(w HttpWriter, r HttpReq) {
	if (r.Method == "GET") {
		id := r.FormValue("pastename")
		text, ok := pastes.Get(id)
		getRender.Render(w, map[string]any{
			"id": id,
			"ok": ok,
			"text": template.HTML(text),
		})
	} else {
		id := r.FormValue("pastename")
		text := r.FormValue("pastebody")
		if (id == "") {
			id = strconv.FormatInt(int64(Hash(text)&9999), 10)
		}
		pastes.Set(id, text)
		postRender.Render(w, map[string]any{
			"id": id,
			"text": text,
		})
	}
}

