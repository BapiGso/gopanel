package login

import (
	"testing"

	"gopanel/core/passwords"
)

func TestCredentialsVerifyAgainstStoredHash(t *testing.T) {
	stored, err := passwords.HashPassword("long-password")
	if err != nil {
		t.Fatal(err)
	}
	for _, example := range []struct {
		name     string
		username string
		password string
		matches  bool
	}{
		{"ok", "administrator", "long-password", true},
		{"wrong password", "administrator", "wrong-password", false},
		{"wrong username", "wrong", "long-password", false},
		{"empty both", "", "", false},
		{"extra username", "administrator-extra", "long-password", false},
		{"extra password", "administrator", "long-password-extra", false},
	} {
		actual := example.username == "administrator" && passwords.VerifyPassword(stored, example.password)
		if actual != example.matches {
			t.Errorf("%s: credential match = %t, want %t", example.name, actual, example.matches)
		}
	}
}

func TestCredentialsLegacyPlaintextFallback(t *testing.T) {
	if !passwords.VerifyPassword("long-password", "long-password") {
		t.Fatal("legacy plaintext credential should still match")
	}
	if passwords.VerifyPassword("long-password", "wrong-password") {
		t.Fatal("legacy plaintext credential matched a wrong password")
	}
}
