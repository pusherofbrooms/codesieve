package app

import "testing"

func TestIsSecretPathBalancedMode(t *testing.T) {
	t.Setenv(secretPathPatternsEnvVar, "")
	t.Setenv(secretPathAllowPatternsEnvVar, "")
	t.Setenv(secretPathModeEnvVar, "")

	tests := []struct {
		path string
		want bool
	}{
		{path: ".env", want: true},
		{path: "config/.env.local", want: true},
		{path: "deploy/certs/server.pem", want: true},
		{path: "src/secrets.py", want: false},
		{path: "model/ClientSecretPolicy.java", want: false},
		{path: "config/client-secret-values.json", want: true},
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

func TestIsSecretPathStrictMode(t *testing.T) {
	t.Setenv(secretPathPatternsEnvVar, "")
	t.Setenv(secretPathAllowPatternsEnvVar, "")
	t.Setenv(secretPathModeEnvVar, "strict")

	if got := isSecretPath("src/secrets.py"); !got {
		t.Fatalf("strict mode expected src/secrets.py to be secret")
	}
	if got := isSecretPath("model/ClientSecretPolicy.java"); !got {
		t.Fatalf("strict mode expected ClientSecretPolicy.java to be secret")
	}
}

func TestIsSecretPathWithEnvPatterns(t *testing.T) {
	t.Setenv(secretPathPatternsEnvVar, "*.crt,config/private/*,foo.bar")
	t.Setenv(secretPathAllowPatternsEnvVar, "")
	t.Setenv(secretPathModeEnvVar, "")

	tests := []struct {
		path string
		want bool
	}{
		{path: "deploy/tls/server.CRT", want: true},
		{path: "config/private/token.txt", want: true},
		{path: "src/foo.bar", want: true},
		{path: "src/public/token.txt", want: false},
	}

	for _, tt := range tests {
		got := isSecretPath(tt.path)
		if got != tt.want {
			t.Fatalf("isSecretPath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestIsSecretPathAllowPatternsOverrideSoftHeuristic(t *testing.T) {
	t.Setenv(secretPathPatternsEnvVar, "")
	t.Setenv(secretPathAllowPatternsEnvVar, "config/*secret*.json")
	t.Setenv(secretPathModeEnvVar, "")

	if got := isSecretPath("config/client-secret-values.json"); got {
		t.Fatalf("allow pattern should suppress soft secret heuristic")
	}
	if got := isSecretPath("secrets/client-secret-values.pem"); !got {
		t.Fatalf("allow pattern must not suppress hard secret extension")
	}
}
