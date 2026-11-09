//go:build !linux

package security

import "fmt"

func restart() error {
	return fmt.Errorf("your operating system needs a manual reboot")
}
