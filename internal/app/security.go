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

var sourceCodeExtensions = map[string]struct{}{
	".go":    {},
	".py":    {},
	".java":  {},
	".js":    {},
	".jsx":   {},
	".ts":    {},
	".tsx":   {},
	".rb":    {},
	".php":   {},
	".cs":    {},
	".c":     {},
	".h":     {},
	".cpp":   {},
	".hpp":   {},
	".cc":    {},
	".m":     {},
	".mm":    {},
	".rs":    {},
	".kt":    {},
	".kts":   {},
	".swift": {},
	".scala": {},
	".sh":    {},
	".bash":  {},
	".zsh":   {},
}

const (
	secretPathPatternsEnvVar      = "CODESIEVE_SECRET_PATH_PATTERNS"
	secretPathAllowPatternsEnvVar = "CODESIEVE_SECRET_PATH_ALLOW_PATTERNS"
	secretPathModeEnvVar          = "CODESIEVE_SECRET_PATH_MODE"
)

func isSecretPath(relPath string) bool {
	rel := strings.ToLower(filepath.ToSlash(relPath))
	base := strings.ToLower(filepath.Base(rel))
	ext := strings.ToLower(filepath.Ext(base))

	if isHardSecretPath(rel, base, ext) {
		return true
	}
	if matchesAnyPattern(extraSecretPatterns(), base, rel) {
		return true
	}
	if matchesAnyPattern(extraSecretAllowPatterns(), base, rel) {
		return false
	}
	if !strings.Contains(base, "secret") {
		return false
	}

	if _, safeDoc := docExtensions[ext]; safeDoc {
		return false
	}
	if secretPathMode() == "strict" {
		return true
	}
	if _, isSource := sourceCodeExtensions[ext]; isSource {
		return false
	}
	return true
}

func isHardSecretPath(rel, base, ext string) bool {
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
	return false
}

func matchesAnyPattern(patterns []string, base, rel string) bool {
	for _, p := range patterns {
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
	return patternsFromEnv(secretPathPatternsEnvVar)
}

func extraSecretAllowPatterns() []string {
	return patternsFromEnv(secretPathAllowPatternsEnvVar)
}

func patternsFromEnv(envVar string) []string {
	raw := strings.TrimSpace(os.Getenv(envVar))
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

func secretPathMode() string {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv(secretPathModeEnvVar)))
	if mode == "strict" {
		return "strict"
	}
	return "balanced"
}
