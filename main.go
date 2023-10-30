package main

import (
	_ "github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
	_ "net/http/pprof"
	"panel/core"
)

func main() {
	c := core.New()
	c.BindFlag()
	c.First()
	c.Route()
	c.Run()
}
