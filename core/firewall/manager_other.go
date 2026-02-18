//go:build !linux && !windows && !darwin

package firewall

import "fmt"

// NewManager returns an error on unsupported platforms.
func NewManager() (Manager, error) {
	return nil, fmt.Errorf("firewall management not supported on this platform")
}
