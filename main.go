package main

import (
	_ "net/http/pprof"
	"panel/core"
)

func main() {
	c := core.New()
	c.Route()
}
