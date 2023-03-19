package core

import "github.com/jmoiron/sqlx"

func QueryPath(Db *sqlx.DB) string {
	var data string
	_ = Db.Select(&data, `SELECT path FROM conf `)
	return data
}

func queryLogin(Db *sqlx.DB) *registerUsr {
	data := new(registerUsr)
	_ = Db.Get(&data, "SELECT username,password,salt FROM user WHERE id='1'")
	return data
}
