package service

import (
	. "mysrv/util"
	"errors"
	"net/http"
	"database/sql"
	"io"
	"encoding/json"
)

var PDBEndpoint = TemplatePage(
	"html/pdb/pdb.gohtml", nil,
	[]GOTMPlugin{GOTM_account, GOTM_mustacc, GOTM_pdbcopy},
)

var pdbSQLScript = `
CREATE TABLE IF NOT EXISTS pdb_info (
	accid INTEGER NOT NULL PRIMARY KEY,
	rowcount INT DEFAULT 1,
	colcount INT DEFAULT 1,
	FOREIGN KEY(accid) REFERENCES accounts(id)
);

CREATE TABLE IF NOT EXISTS pdb_data (
	accid INTEGER NOT NULL,
	row INT NOT NULL,
	col INT NOT NULL,
	data TEXT NOT NULL,
	FOREIGN KEY(accid) REFERENCES pdb_info(accid),
	UNIQUE(accid, row, col),
	CHECK(row >= 0),
	CHECK(col >= 0)
);
`

func init() {
	SQLInitScript( "pdb", pdbSQLScript )
}

func pdbRemoveRow(id int64, row string) (err error) {
	_, err = SQLDo("pdb remove row", `
	DELETE FROM pdb_data WHERE accid=? AND row=?;
	UPDATE OR REPLACE pdb_data SET row=row-1 WHERE (accid=? AND row>?);
	UPDATE pdb_info SET rowcount=max(rowcount-1, 1) WHERE accid=?;
	`, id, row, id, row, id)
	return
}

func pdbRemoveCol(id int64, col string) (err error) {
	_, err = SQLDo("pdb remove col", `
	DELETE FROM pdb_data WHERE accid=? AND col=?;
	UPDATE OR REPLACE pdb_data SET col=col-1 WHERE (accid=? AND col>?);
	UPDATE pdb_info SET colcount=max(colcount-1, 1) WHERE accid=?;
	`, id, col, id, col, id)
	return
}

func pdbSetSize(id int64, row, col string) {
	_, err := SQLDo("pdb set size", `
UPDATE pdb_info
SET rowcount=?, colcount=?
WHERE accid=?;
`, row, col, id)
	if (err != nil) {panic(err)}
}

func pdbSet(id int64, row, col, data string) {
	_, err := SQLDo("pdb set data", `
INSERT INTO pdb_data (accid, row, col, data)
VALUES (?, ?, ?, ?)
ON CONFLICT DO UPDATE SET data=?;
`, id, row, col, data, data)
	if (err != nil) {panic(err)}
}

func pdbSetBatch(id int64, pdbSets []PDBSlot) {
// INSERT INTO pdb_data (id, row, col, data) VALUES
// (?, ?, ?, ?) ON CONFLICT DO UPDATE SET data=? * len(pdbSets)
// ;
	// TODO batch sql command (somehow set the ON CONFLICT for each insert)
	for _, set := range pdbSets {
		pdbSet(id,  set.Row, set.Col, set.Data)
	}
}

func pdbRead(id int64) [][]string {
	sqlrow := SQLGetSingle("pdb get col/row count", "SELECT rowcount, colcount FROM pdb_info WHERE (accid=?);", id)

	var rowcount, colcount int
	err := sqlrow.Scan(&rowcount, &colcount)

	if (err != nil) {
		if errors.Is(err, sql.ErrNoRows) {
			rowcount = 1
			colcount = 1
			FLog(SQLArea, "creating pdb entry for [%d]\n", id)
			_, err = SQLDo("create pdb info entry", "INSERT INTO pdb_info (accid) VALUES (?)", id)
			if (err != nil) {panic(err)}
		} else {
			panic(err)
		}
	}

	table := make([][]string, rowcount)
	for i := range table {
		table[i] = make([]string, colcount)
	}

	sqlinfo, err := SQLGet("pdb get data", `
SELECT row, col, data FROM pdb_data
WHERE accid=? AND row<? AND col<?
LIMIT ?;`, id, rowcount, colcount, rowcount*colcount)
	if (err != nil) {panic(err)}
	defer sqlinfo.Close()

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

	id := accinfo["id"].(int64)
	return true, pdbRead(id)
}
var GOTM_pdbcopy = GOTMPlugin{"pdb", pdbCopy}

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
	_, accinfo := GOTM_account.Plug(w, r, make(map[string]any))
	ok, _ := GOTM_mustacc.Plug(w, r, map[string]any{"acc":accinfo})
	if (!ok) {return}
	id := accinfo.(map[string]any)["id"].(int64)

	read, e := io.ReadAll(r.Body)
	if (e != nil) { panic ( e ) }
	switch (r.Method) {
	case "POST":
		var setBatch = r.URL.Query().Get("batch")=="true"
		if (setBatch) {
			var pdbupdate []PDBSlot
			e = json.Unmarshal(read, &pdbupdate)
			if (e != nil) { panic ( e ) }
			pdbSetBatch(id, pdbupdate)
		} else {
			var pdbupdate PDBSlot
			e = json.Unmarshal(read, &pdbupdate)
			if (e != nil) { panic ( e ) }
			pdbSet(id, pdbupdate.Row, pdbupdate.Col, pdbupdate.Data)
		}
	case "PATCH":
		var pdbupdate PDBSize
		e = json.Unmarshal(read, &pdbupdate)
		if (e != nil) { panic ( e ) }
		pdbSetSize(id, pdbupdate.Row, pdbupdate.Col)
	case "DELETE":
		var pdbupdate PDBSize
		e = json.Unmarshal(read, &pdbupdate)
		if (e != nil) { panic ( e ) }
		var e error

		if (pdbupdate.Col != "" && pdbupdate.Row == "") {
			e = pdbRemoveCol(id,pdbupdate.Col)
		} else if (pdbupdate.Row != "" && pdbupdate.Col == "") {
			e = pdbRemoveRow(id, pdbupdate.Row)
		} else {
			FLog(FLOG_ERROR, "User [%d] tried to remove row and column or neither\n", id)
			w.WriteHeader(http.StatusBadRequest)
			panic("User tried to remove row and column or neither")
		}
		if (e != nil) {
			panic(e)
		}

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

