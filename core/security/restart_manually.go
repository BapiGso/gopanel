//go:build !linux

package security

import "fmt"

func restart() error {
	return fmt.Errorf("Your operating system need manually reboot")
}
