package app

import (
	"os"
	"path"
	"path/filepath"
	"strings"
)

var secretBasenames = map[string]struct{}{
	".env":        {},
	".npmrc":      {},
	".pypirc":     {},
	".netrc":      {},
	".htpasswd":   {},
	"id_rsa":      {},
	"id_ed25519":  {},
	"id_dsa":      {},
	"id_ecdsa":    {},
	"credentials": {},
}

var secretExtensions = map[string]struct{}{
	".pem":      {},
	".key":      {},
	".p12":      {},
	".pfx":      {},
	".jks":      {},
	".keystore": {},
	".token":    {},
	".secrets":  {},
}

var docExtensions = map[string]struct{}{
	".md":       {},
	".markdown": {},
	".mdx":      {},
	".rst":      {},
	".txt":      {},
	".adoc":     {},
	".asciidoc": {},
	".html":     {},
	".htm":      {},
	".ipynb":    {},
}

const secretPathPatternsEnvVar = "CODESIEVE_SECRET_PATH_PATTERNS"

func isSecretPath(relPath string) bool {
	rel := strings.ToLower(filepath.ToSlash(relPath))
	base := strings.ToLower(filepath.Base(rel))
	ext := strings.ToLower(filepath.Ext(base))

	if base == ".env" || strings.HasPrefix(base, ".env.") {
		return true
	}
	if _, ok := secretBasenames[base]; ok {
		return true
	}
	if strings.HasPrefix(base, "id_rsa.") || strings.HasPrefix(base, "id_ed25519.") {
		return true
	}
	if strings.HasPrefix(base, "service-account") && strings.HasSuffix(base, ".json") {
		return true
	}
	if base == "credentials.json" {
		return true
	}
	if _, ok := secretExtensions[ext]; ok {
		return true
	}
	if strings.Contains(base, "secret") {
		if _, safeDoc := docExtensions[ext]; !safeDoc {
			return true
		}
	}
	for _, p := range extraSecretPatterns() {
		if p == "" {
			continue
		}
		if matched, _ := path.Match(p, base); matched {
			return true
		}
		if matched, _ := path.Match(p, rel); matched {
			return true
		}
	}
	return false
}

func extraSecretPatterns() []string {
	raw := strings.TrimSpace(os.Getenv(secretPathPatternsEnvVar))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	patterns := make([]string, 0, len(parts))
	for _, part := range parts {
		p := strings.ToLower(strings.TrimSpace(part))
		if p == "" {
			continue
		}
		patterns = append(patterns, p)
	}
	return patterns
}
