// core/security/restart.go
//go:build !linux
// +build !linux

package security

import "fmt"

func restart() func() {
	return func() {
		fmt.Println(123)
	}
}
