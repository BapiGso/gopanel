// core/security/restart.go
//go:build linux
// +build linux

package security

import (
	"fmt"
	"os"
	"syscall"
)

func restart() {
	executable, err := os.Executable()
	if err != nil {
		return
	}

	// 使用 syscall.Exec 替换当前进程
	err = syscall.Exec(executable, os.Args, os.Environ())
	if err != nil {
		fmt.Println("Error restarting process:", err)
		os.Exit(1)
	}
}
