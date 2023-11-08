package main

import (
	"fmt"
	"net/http"
	. "mysrv/util"
	"mysrv/service"
)

func CreateHandler(w HttpWriter, r HttpReq, info map[string]any) (render bool, ret_r any) {
	ret_r = make(map[string]any)
	ret := ret_r.(map[string]any)
	ret["failed"] = false

	if (r.Method == "GET") {
		return true, ret
	}

	email := r.FormValue("email")
	username := r.FormValue("username")
	password := r.FormValue("password")
	if ( email == "" || username == "" || password == "" ) {
		ret["failed"] = true
		ret["failReason"] = "Missing account parameter"
		return true, ret
	}

	acc := NewAccount(email, username, password)
	if (acc == nil) {
		ret["failed"] = true
		ret["failReason"] = "Email already in use"
		ret["failEmail"] = true
		return true, ret
	}

	acc.SendCookie(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return false, ret
}

func LoginHandler(w HttpWriter, r HttpReq, info map[string]any) (render bool, ret_r any) {
	ret_r = make(map[string]any)
	ret := ret_r.(map[string]any)
	ret["failed"] = false

	if (r.Method == "GET") {
		return true, ret
	}

	email := r.FormValue("email")
	password := r.FormValue("password")
	if ( email == "" || password == "" ) {
		ret["failed"] = true
		ret["failReason"] = "Missing login parameter"
		return true, ret
	}

	acc, exists := GetAccount(email)
	if (!exists || acc.Hash != Hash(password)) {
		ret["failed"] = true
		ret["failReason"] = "Wrong password or email"
		fmt.Println(ret, Hash(password), acc, exists)
		return true, ret
	}

	acc.SendCookie(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return false, ret
}

var ( // templated pages
	index = LogicPage(
		"html/index.gohtml", nil,
		[]GOTMPlugin{GOTM_account, GOTM_log},
		func (w HttpWriter, r HttpReq, info map[string]any) (bool, any) {
			if (r.URL.Path != "/") { missing.ServeHTTP(w, r) }
			return r.URL.Path == "/", nil
		},
	)

	missing = TemplatePage(
		"html/missing.gohtml", nil,
		[]GOTMPlugin{GOTM_account, GOTM_urlInfo, GOTM_log},
	)
	users = TemplatePage(
		"html/users.gohtml", nil,
		[]GOTMPlugin{GOTM_account, GOTM_accounts, GOTM_log},
	)
	// must be logged in plugin
	chat = TemplatePage(
		"html/chat.gohtml", nil,
		[]GOTMPlugin{GOTM_account, GOTM_mustacc, GOTM_log},
	)
	//chat = LogicPage(
	//	"html/chat.gohtml", nil,
	//	[]GOTMPlugin{GOTM_account, GOTM_log},
	//	func (w HttpWriter, r HttpReq, info map[string]any) (bool, any) {
	//		if (!info["acc"].(map[string]any)["ok"].(bool)) {
	//			http.Redirect(w, r, "/login", http.StatusSeeOther)
	//			return false, nil
	//		}
	//		return true, nil
	//	},
	//)
)

func main() {
	InitSQL("sqlite3.db")

	http.Handle("/", index)
	http.Handle("/favicon.ico", StaticFile{"./files/dice.ico"})
	http.Handle("/users", users)
	http.Handle("/chat", chat)

	http.Handle("/register", LogicPage(
		"html/register.gohtml", nil,
		[]GOTMPlugin{GOTM_account, GOTM_log},
		CreateHandler,
	))
	http.Handle("/login", LogicPage(
		"html/login.gohtml", nil,
		[]GOTMPlugin{GOTM_account, GOTM_log},
		LoginHandler,
	))
	http.Handle("/files/", http.StripPrefix("/files", http.FileServer(http.Dir("./files/"))))
	http.Handle("/wschat", service.ChatServer)

	fmt.Println("running")
	http.ListenAndServe("0.0.0.0:8080", nil)
}
