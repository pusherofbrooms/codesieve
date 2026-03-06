package app

import "testing"

func TestParseGoSymbols(t *testing.T) {
	src := []byte(`package sample

type User struct {}

func Authenticate(token string) error { return nil }

func (u *User) Login(name string) bool { return true }
`)
	syms, lang, err := ParseSymbols("sample.go", src)
	if err != nil {
		t.Fatalf("ParseSymbols error: %v", err)
	}
	if lang != "go" {
		t.Fatalf("lang = %q", lang)
	}
	if len(syms) < 3 {
		t.Fatalf("expected at least 3 symbols, got %d", len(syms))
	}
	if syms[0].Name != "User" || syms[0].Kind != "struct" {
		t.Fatalf("unexpected first symbol: %+v", syms[0])
	}
	if syms[2].QualifiedName != "User.Login" || syms[2].Kind != "method" {
		t.Fatalf("unexpected method symbol: %+v", syms[2])
	}
}

func TestParsePythonSymbols(t *testing.T) {
	src := []byte(`class Auth:
    def login(self, user):
        return True

def helper(name):
    return name
`)
	syms, lang, err := ParseSymbols("auth.py", src)
	if err != nil {
		t.Fatalf("ParseSymbols error: %v", err)
	}
	if lang != "python" {
		t.Fatalf("lang = %q", lang)
	}
	if len(syms) != 3 {
		t.Fatalf("expected 3 symbols, got %d", len(syms))
	}
	if syms[1].QualifiedName != "Auth.login" || syms[1].Kind != "method" {
		t.Fatalf("unexpected symbol: %+v", syms[1])
	}
}
