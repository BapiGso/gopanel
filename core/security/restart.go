//go:build linux

package security

import (
	"os"
	"syscall"
)

func restart() error {

	executable, err := os.Executable()
	if err != nil {
		return err
	}

	// 使用 syscall.Exec 替换当前进程
	err = syscall.Exec(executable, os.Args, os.Environ())
	if err != nil {
		return err
	}

	return nil
}
