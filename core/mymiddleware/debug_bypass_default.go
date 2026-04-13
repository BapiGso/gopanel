//go:build !gopanel_debug

package mymiddleware

func debugBypassEnabled() bool {
	return false
}
