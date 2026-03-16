package app

import "testing"

func TestIsSecretPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: ".env", want: true},
		{path: "config/.env.local", want: true},
		{path: "deploy/certs/server.pem", want: true},
		{path: "src/secrets.py", want: true},
		{path: "src/Service.SECRETS", want: true},
		{path: "docs/secrets-handling.md", want: false},
		{path: "README.md", want: false},
		{path: "src/auth.py", want: false},
	}

	for _, tt := range tests {
		got := isSecretPath(tt.path)
		if got != tt.want {
			t.Fatalf("isSecretPath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}
