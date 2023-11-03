package util

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

var sqlArea = NewArea("SQL")

type SQLScript struct {
	Name string
	Code string
}
var SQL_INIT_SCRIPTS = []SQLScript{}

type SQLFunc struct {
	Name string
	Func func(*sql.DB) error
}
var SQL_INIT_FUNCS = []SQLFunc{}

func InitSQL(dbfile string) error {
	var err error
	db, err = sql.Open("sqlite3", dbfile)
	if (err != nil) {
		FLog(sqlArea, "Failed openning %q with sqlite3 drivers\n", dbfile)
		return err
	}
	FLog(sqlArea, "Successefully openned %q with sqlite3 drivers\n", dbfile)

	for _, script :=range SQL_INIT_SCRIPTS {
		_, err = db.Exec(script.Code)

		if (err != nil) {
			FLog(sqlArea, "Failed executing script [%s]: %v\n%s\n", script.Name, err, script.Code)
			return err
		}
		FLog(sqlArea, "script [%s] executed successefully\n", script.Name)
	}

	for _, fnc :=range SQL_INIT_FUNCS {
		err = fnc.Func(db)

		if (err != nil) {
			FLog(sqlArea, "Failed executing func [%s]: %v\n", fnc.Name, err)
			return err
		}
		FLog(sqlArea, "func [%s] executed successefully\n", fnc.Name)
	}
	return nil
}

func StopSQL() {
	db.Close()
}
