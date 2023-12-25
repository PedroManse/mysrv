package service

import (
	"mysrv/util"
	"errors"
	"net/http"
	"database/sql"
	"io"
	"encoding/json"
)

type HttpWriter = http.ResponseWriter
type HttpReq = *http.Request

var pdbSQLScript = `
CREATE TABLE IF NOT EXISTS pdb_info (
	email TEXT NOT NULL PRIMARY KEY,
	rowcount INT NOT NULL DEFAULT 1,
	colcount INT NOT NULL DEFAULT 1,
	FOREIGN KEY(email) REFERENCES accounts(email)
);

CREATE TABLE IF NOT EXISTS pdb_data (
	email TEXT NOT NULL,
	row INT NOT NULL,
	col INT NOT NULL,
	data TEXT,
	FOREIGN KEY(email) REFERENCES pdb_info(email),
	UNIQUE(email, row, col)
);
`

func init() {
	util.SQLInitScript( "pdb", pdbSQLScript )
}

func pdbSetSize(email, row, col string) {
	_, err := util.SQLDo("pdb set size", `
UPDATE pdb_info
SET rowcount=?, colcount=?
WHERE email=?;
`, row, col, email)
	if (err != nil) {panic(err)}
}

func pdbSet(email, row, col, data string) {
	_, err := util.SQLDo("pdb set data", `
INSERT INTO pdb_data (email, row, col, data)
VALUES (?, ?, ?, ?)
ON CONFLICT DO UPDATE SET data=?;
`, email, row, col, data, data)
	if (err != nil) {panic(err)}
}

func pdbSetBatch(email string, pdbSets []PDBSlot) {
	// TODO batch sql command
	for set := range pdbSets {
		pdbSet(email,  set.row, set.col, set.data)
	}
}

func pdbRead(email string) [][]string {
	sqlrow := util.SQLGetSingle("pdb get col/row count", "SELECT rowcount, colcount FROM pdb_info WHERE (email=?);", email)

	var rowcount, colcount int
	err := sqlrow.Scan(&rowcount, &colcount)

	if (err != nil) {
		if errors.Is(err, sql.ErrNoRows) {
			rowcount = 1
			colcount = 1
			util.FLog(1, "creating pdb entry for %s\n", email)
			_, err = util.SQLDo("create pdb info entry", "INSERT INTO pdb_info (email) VALUES (?)", email)
			if (err != nil) {panic(err)}
		} else {
			panic(err)
		}
	}

	table := make([][]string, rowcount)
	for i := range table {
		table[i] = make([]string, colcount)
	}

	sqlinfo, err := util.SQLGet("pdb get data", `
SELECT row, col, data FROM pdb_data
WHERE email=? AND row<? AND col<?
LIMIT ?;`, email, rowcount, colcount, rowcount*colcount)
	if (err != nil) {panic(err)}

	var data string
	var row, col int

	for sqlinfo.Next() {
		err = sqlinfo.Scan(&row, &col, &data)
		if (err != nil) {panic(err)}
		table[row][col] = data
	}

	return table
}

func pdbCopy(w HttpWriter, r HttpReq, info map[string]any) (render bool, ret_r any) {
	accinfo := info["acc"].(map[string]any)

	if (!accinfo["ok"].(bool)) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return false, nil
	}

	email := accinfo["email"].(string)
	return true, pdbRead(email)
}
var GOTM_pdbcopy = util.GOTMPlugin{"pdb", pdbCopy}

type PDBSlot struct {
	Row string `json:"row"`
	Col string `json:"col"`
	Data string `json:"data"`
}

type PDBSize struct {
	Row string `json:"row"`
	Col string `json:"col"`
}

func PDBHandler(w HttpWriter, r HttpReq) {
	_, accinfo := util.GOTM_account.Plug(w, r, make(map[string]any))
	ok, _ := util.GOTM_mustacc.Plug(w, r, map[string]any{"acc":accinfo})
	if (!ok) {return}
	email := accinfo.(map[string]any)["email"].(string)

	switch (r.Method) {
	case "POST":
		read, e := io.ReadAll(r.Body)
		if (e != nil) { panic ( e ) }
		var setBatch = r.URL.Query().Get("batch")=="true"
		if (setBatch) {
			var pdbupdate []PDBSlot
			e = json.Unmarshal(read, &pdbupdate)
			if (e != nil) { panic ( e ) }
			pdbSetBatch(email, pdbupdate)
		} else {
			var pdbupdate PDBSlot
			e = json.Unmarshal(read, &pdbupdate)
			if (e != nil) { panic ( e ) }
			pdbSet(email, pdbupdate.Row, pdbupdate.Col, pdbupdate.Data)
		}
	case "PATCH":
		read, e := io.ReadAll(r.Body)
		if (e != nil) { panic ( e ) }
		var pdbupdate PDBSize
		e = json.Unmarshal(read, &pdbupdate)
		if (e != nil) { panic ( e ) }
		//pdbSetSize(email, pdbupdate.Row, pdbupdate.Col)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
