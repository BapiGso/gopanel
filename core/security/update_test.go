package security

import "testing"

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   string
		wantOK bool
	}{
		{
			name:   "with v prefix",
			input:  "v1.2.3",
			want:   "1.2.3",
			wantOK: true,
		},
		{
			name:   "without v prefix",
			input:  "1.2.3",
			want:   "1.2.3",
			wantOK: true,
		},
		{
			name:   "with whitespace",
			input:  " v1.2.3 ",
			want:   "1.2.3",
			wantOK: true,
		},
		{
			name:   "minor tag",
			input:  "v2.8",
			want:   "2.8.0",
			wantOK: true,
		},
		{
			name:   "major tag",
			input:  "v2",
			want:   "2.0.0",
			wantOK: true,
		},
		{
			name:   "development build",
			input:  "(devel)",
			wantOK: false,
		},
		{
			name:   "invalid version",
			input:  "main",
			wantOK: false,
		},
		{
			name:   "empty version",
			input:  "",
			wantOK: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := parseVersion(test.input)
			if ok != test.wantOK {
				t.Fatalf("parseVersion(%q) ok = %v, want %v", test.input, ok, test.wantOK)
			}
			if !test.wantOK {
				return
			}
			if got.String() != test.want {
				t.Fatalf("parseVersion(%q) = %s, want %s", test.input, got, test.want)
			}
		})
	}
}

func TestCurrentVersionUsesInjectedBuildVersion(t *testing.T) {
	oldBuildVersion := buildVersion
	t.Cleanup(func() {
		buildVersion = oldBuildVersion
	})

	buildVersion = "v2.3.4"
	got, ok := currentVersion()
	if !ok {
		t.Fatal("currentVersion() ok = false, want true")
	}
	if got.String() != "2.3.4" {
		t.Fatalf("currentVersion() = %s, want 2.3.4", got)
	}
}
