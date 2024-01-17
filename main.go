package main

import (
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
		return true, ret
	}

	acc.SendCookie(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return false, ret
}

var ( // system pages
	index = LogicPage(
		"html/sys/index.gohtml", nil,
		[]GOTMPlugin{GOTM_account},
		func (w HttpWriter, r HttpReq, info map[string]any) (bool, any) {
			if (r.URL.Path != "/") { missing.ServeHTTP(w, r) }
			return r.URL.Path == "/", nil
		},
	)
	register = LogicPage(
		"html/sys/register.gohtml", nil,
		[]GOTMPlugin{GOTM_account},
		CreateHandler,
	)
	login = LogicPage(
		"html/sys/login.gohtml", nil,
		[]GOTMPlugin{GOTM_account},
		LoginHandler,
	)

	missing = TemplatePage(
		"html/sys/missing.gohtml", nil,
		[]GOTMPlugin{GOTM_account, GOTM_urlInfo, GOTM_log},
	)
	users = TemplatePage(
		"html/sys/users.gohtml", nil,
		[]GOTMPlugin{GOTM_account, GOTM_mustacc, GOTM_accounts},
	)

	forms = TemplatePage(
		"html/forms/forms.gohtml", nil,
		[]GOTMPlugin{GOTM_account, GOTM_mustacc},
	)
)

func main() {
	InitSQL("sqlite3.db")
	InitAssoc()
	service.DebugSocial()

	// site-wide service
	http.Handle("/", index)
	http.Handle("/users", users)
	http.Handle("/login", login)
	http.Handle("/register", register)
	http.Handle("/favicon.ico", StaticFile("./files/dice.ico"))
	http.Handle("/files/", http.StripPrefix("/files", http.FileServer(http.Dir("./files/"))))

	// real-time WebSocket chat
	http.Handle("/chat", service.ChatEndpoint)
	http.Handle("/wschat", service.ChatServer)

	// ephemeral public info
	http.Handle("/ecb", service.ECBEndpoint)
	http.HandleFunc("/fsecb", service.ECBHandler)

	// non-ephemeral private info
	http.Handle("/pdb", service.PDBEndpoint)
	http.HandleFunc("/fspdb", service.PDBHandler)

	// plataform to create and host <form>s
	http.Handle("/forms", forms)
	// TODO: literally most parts

	// social media
	http.Handle("/social/all", service.AllEndpoint)
	http.Handle("/social/posts", service.PostPageEndpoint)
	http.Handle("/social/posts/create", service.CreatePostPageEndpoint)
	http.Handle("/social/posts/react", service.ReactToPostEndpoint)
	http.Handle("/social/comments/react", service.ReactToCommentEndpoint)
	http.Handle("/social/comments/create", service.CreateCommentEndpoint)
	http.Handle("/social/comp/reply-form", service.CompReplyFormEndpoint)
	http.Handle("/social/comp/reply-button", service.CompReplyButtonEndpoint)
	http.Handle("/social/community/create", service.CreateCommunityEndpoint)

	FLog(FLOG_INFO, "Running\n")
	panic(http.ListenAndServe("0.0.0.0:8080", nil))
}
