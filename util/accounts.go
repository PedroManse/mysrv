package util

import (
	"sync"
	"strconv"
	"net/http"
	"fmt"
	"time"
	"strings"
	"database/sql"
)

type Account struct {
	Email string
	Name string
	Hash HashResult
}

var (
	accLock = sync.Mutex{}
	Accounts = make(map[string]*Account)
)

func AccountsCopy() (cacc map[string]Account) {
	cacc = make(map[string]Account)
	accLock.Lock()
	for email, acc := range Accounts {
		cacc[email] = *acc
	}
	accLock.Unlock()
	return cacc
}

const AccountCookieName = "mysrv-cookie"

func (A *Account) MakeCookieStr() string {
	return fmt.Sprintf("%s:%v", A.Email, A.Hash)
}

func ReadAccountCookie(str string) (email string, value HashResult, ok bool) {
	steps := strings.SplitN(str, ":", 2)
	if len(steps) != 2 { // no ":" in string
		return "", 0 ,false
	}
	value64, ValidHashResult := strconv.ParseUint(steps[1], 10, HashBitLen)
	if (ValidHashResult != nil) {
		return "", 0 ,false
	}
	return steps[0], HashResult(value64), true
}

func (A *Account)SendCookie(w HttpWriter) {
	SendCookie := http.Cookie{
		Name: AccountCookieName,
		Value: A.MakeCookieStr(),
		HttpOnly: true,
		//Secure: true, // not serving https yet
		Path: "/",
		SameSite: http.SameSiteStrictMode,
		Expires: time.Now().Add(time.Hour * 24 * 10), // 10 days
	}
	http.SetCookie(w, &SendCookie)
}

func GetAccount(email string) (acc *Account, exists bool) {
	accLock.Lock()
	acc, exists = Accounts[email]
	accLock.Unlock()
	return
}

func NewAccount(email, name, password string) *Account {
	accLock.Lock()
	defer accLock.Unlock()
	_, used := Accounts[email]
	if (used) { return nil }

	hash := Hash(password)
	acc := &Account{email, name, hash}
	Accounts[email] = acc

	_, err := db.Exec(
		`INSERT INTO accounts (email, name, hash) VALUES (?, ?, ?);`,
		email, name, hash,
	)
	if (err != nil) {
		panic(err)
	}

	return acc
}

func loadAccounts(db *sql.DB) error {
	rows, err := db.Query("SELECT email, name, hash FROM accounts")
	if err != nil { return err }

	for rows.Next() {
		var email, name string
		var hash HashResult

		err = rows.Scan(&email, &name, &hash)
		Accounts[email] = &Account{email, name, hash}
		if err != nil { return err }
	}
	return nil
}

func init() {
	SQL_INIT_SCRIPTS = append(SQL_INIT_SCRIPTS, SQLScript{
		"accounts",
`CREATE TABLE IF NOT EXISTS accounts (
	email TEXT NOT NULL PRIMARY KEY,
	name TEXT NOT NULL,
	hash INT NOT NULL
);`})
	SQL_INIT_FUNCS = append(SQL_INIT_FUNCS, SQLFunc{
		"accounts",
		loadAccounts,
	})
}
