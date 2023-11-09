package service

import (
	"net/http"
	"fmt"
	"mysrv/util"
	"html"
)

var pastes = map[string]string{}

func ECBHandler(w http.ResponseWriter, r *http.Request) {
	if (r.Method == "GET") {
		q := r.URL.Query()
		id := q.Get("pastename")
		text, ok := pastes[id]
		if (!q.Has("pastename") || !ok) {
			fmt.Fprintf(w, "<h3>No such Paste %q</h3>", id)
		} else {
			fmt.Fprintf(w, "<pre>%s</pre>", text)
		}
	} else {
		name := r.FormValue("pastename")
		body := html.EscapeString(r.FormValue("pastebody"))
		if (name == "") {
			name = fmt.Sprintf("%d", util.Hash(body)&9999)
		}
		pastes[name] = body
		fmt.Fprintf(w, "<h3>Paste %q Created</h3><pre>%s</pre>", name, body)
	}
}

