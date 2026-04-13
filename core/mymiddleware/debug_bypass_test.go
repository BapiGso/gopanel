package mymiddleware

import "testing"

func TestDebugBypassDisabledByDefault(t *testing.T) {
	if debugBypassEnabled() {
		t.Fatal("debugBypassEnabled() should be false without gopanel_debug build tag")
	}
}
