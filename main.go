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
		[]GOTMPlugin{GOTM_account},
		func (w HttpWriter, r HttpReq, info map[string]any) (bool, any) {
			if (r.URL.Path != "/") { missing.ServeHTTP(w, r) }
			return r.URL.Path == "/", nil
		},
	)
	register = LogicPage(
		"html/register.gohtml", nil,
		[]GOTMPlugin{GOTM_account},
		CreateHandler,
	)
	login = LogicPage(
		"html/login.gohtml", nil,
		[]GOTMPlugin{GOTM_account},
		LoginHandler,
	)

	missing = TemplatePage(
		"html/missing.gohtml", nil,
		[]GOTMPlugin{GOTM_account, GOTM_urlInfo, GOTM_log},
	)
	users = TemplatePage(
		"html/users.gohtml", nil,
		[]GOTMPlugin{GOTM_account, GOTM_accounts},
	)
	// must be logged in plugin
	chat = TemplatePage(
		"html/chat.gohtml", nil,
		[]GOTMPlugin{GOTM_account, GOTM_mustacc},
	)
	ecb = TemplatePage(
		"html/ecb.gohtml", nil,
		[]GOTMPlugin{GOTM_account, GOTM_mustacc},
	)
	pdb = TemplatePage(
		"html/pdb.gohtml", nil,
		[]GOTMPlugin{GOTM_account, GOTM_mustacc, service.GOTM_pdbcopy},
	)
	forms = TemplatePage(
		"html/forms.gohtml", nil,
		[]GOTMPlugin{GOTM_account, GOTM_mustacc},
	)
)

func main() {
	InitSQL("sqlite3.db")
	InitAssoc()
	service.DebugSocial()

	// site-wide service
	http.Handle("/", index)
	http.Handle("/login", login)
	http.Handle("/register", register)
	http.Handle("/favicon.ico", StaticFile{"./files/dice.ico"})
	http.Handle("/files/", http.StripPrefix("/files", http.FileServer(http.Dir("./files/"))))

	// front-ends
	http.Handle("/users", users)
	http.Handle("/chat", chat)
	http.Handle("/ecb", ecb)
	http.Handle("/pdb", pdb)
	http.Handle("/forms", forms)

	// back-ends
	http.Handle("/wschat", service.ChatServer)
	http.HandleFunc("/fsecb", service.ECBHandler)
	http.HandleFunc("/fspdb", service.PDBHandler)

	// /social
	http.Handle("/social/all", service.AllEndpoint)

	FLog(FLOG_INFO, "Running\n")
	panic(http.ListenAndServe("0.0.0.0:8080", nil))
}
