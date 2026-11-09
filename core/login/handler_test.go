package login

import "testing"

func TestCredentialsMatchChecksBothFixedLengthHashes(t *testing.T) {
	for _, example := range []struct {
		username string
		password string
		matches  bool
	}{{"administrator", "long-password", true}, {"administrator", "wrong-password", false}, {"wrong", "long-password", false}, {"", "", false}, {"administrator-extra", "long-password", false}, {"administrator", "long-password-extra", false}, {"administrator", "", false}} {
		if actual := credentialsMatch(example.username, example.password, "administrator", "long-password"); actual != example.matches {
			t.Errorf("credential match = %t, want %t", actual, example.matches)
		}
	}
}
