package main

import (
	"fmt"
	"net/http"
	. "mysrv/util"
)

var ( // templated pages
	index = TemplatePage(
		"html/index.gohtml",
		map[string]any{},
		[]GOTMPlugin{GOTM_Log},
	)
)

func main() {
	http.Handle("/", index)
	fmt.Println("running")
	http.ListenAndServe(":8080", nil)
}
