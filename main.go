package main

import (
	_ "github.com/jmoiron/sqlx"
	"gopanel/core"
	_ "modernc.org/sqlite"
	_ "net/http/pprof"
)

func main() {
	c := core.New()
	c.BindFlag()
	c.First()
	c.Route()
	c.Run()
}
