package util

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

var SQLArea = NewArea("SQL")

type SQLScript struct {
	Name string
	Code string
}
var init_scripts = []SQLScript{}
func SQLInitScript(Name string, Code string) {
	init_scripts = append( init_scripts, SQLScript{Name, Code})
}

func SQLGetSingle(Name, Query string, vars... any) ( info *sql.Row ) {
	return db.QueryRow(Query, vars...)
}

func SQLGet(Name, Query string, vars... any) ( info *sql.Rows, err error) {
	info, err = db.Query(Query, vars...)
	if (err != nil) {
		FLog(SQLArea, "Failed executing dynamic script [%s]: %v\n%s with %+v\n", Name, err, Query, vars)
	}
	return
}

func SQLDo(Name, Query string, vars... any) ( info sql.Result, err error) {
	info, err = db.Exec(Query, vars...)
	if (err != nil) {
		FLog(SQLArea, "Failed executing dynamic script [%s]: %v\n%s with %+v\n", Name, err, Query, vars)
	}
	return
}

type SQLFunc struct {
	Name string
	Func func(*sql.DB) error
}
var init_funcs = []SQLFunc{}

func SQLInitFunc(Name string, Func func(*sql.DB) error) {
	init_funcs = append( init_funcs, SQLFunc{Name, Func})
}

func InitSQL(dbfile string) error {
	var err error
	db, err = sql.Open("sqlite3", dbfile)
	if (err != nil) {
		FLog(SQLArea, "Failed openning %q with sqlite3 drivers\n", dbfile)
		return err
	}
	FLog(SQLArea, "Successefully openned %q with sqlite3 drivers\n", dbfile)

	for _, script :=range init_scripts {
		_, err = db.Exec(script.Code)

		if (err != nil) {
			FLog(SQLArea, "Failed executing script [%s]: %v\n%s\n", script.Name, err, script.Code)
			return err
		}
		FLog(SQLArea, "script [%s] executed successefully\n", script.Name)
	}

	for _, fnc :=range init_funcs {
		err = fnc.Func(db)

		if (err != nil) {
			FLog(SQLArea, "Failed executing func [%s]: %v\n", fnc.Name, err)
			return err
		}
		FLog(SQLArea, "func [%s] executed successefully\n", fnc.Name)
	}
	return nil
}

func StopSQL() {
	db.Close()
}
