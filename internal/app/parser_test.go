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
	if syms[2].QualifiedName != "User.Login" || syms[2].Kind != "method" || syms[2].ParentID != "User" {
		t.Fatalf("unexpected method symbol: %+v", syms[2])
	}
}

func TestParsePythonSymbols(t *testing.T) {
	src := []byte(`class Auth:
    @decorator
    async def login(self, user):
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
	if syms[1].QualifiedName != "Auth.login" || syms[1].Kind != "method" || syms[1].ParentID != "Auth" {
		t.Fatalf("unexpected symbol: %+v", syms[1])
	}
}

func TestParseTypeScriptSymbols(t *testing.T) {
	src := []byte(`export interface User {
  name: string
}

export class Client {
  login(token: string) {
    return token
  }
}

export const fetchUser = (id: string) => id
export const routes = lazy(() => createRoutes())
`)
	syms, lang, err := ParseSymbols("client.ts", src)
	if err != nil {
		t.Fatalf("ParseSymbols error: %v", err)
	}
	if lang != "typescript" {
		t.Fatalf("lang = %q", lang)
	}
	if len(syms) < 5 {
		t.Fatalf("expected at least 5 symbols, got %d", len(syms))
	}
	foundMethod := false
	foundArrow := false
	foundWrappedArrow := false
	foundInterface := false
	for _, sym := range syms {
		switch {
		case sym.QualifiedName == "Client.login" && sym.Kind == "method" && sym.ParentID == "Client":
			foundMethod = true
		case sym.Name == "fetchUser" && sym.Kind == "function":
			foundArrow = true
		case sym.Name == "routes" && sym.Kind == "function":
			foundWrappedArrow = true
		case sym.Name == "User" && sym.Kind == "interface":
			foundInterface = true
		}
	}
	if !foundMethod || !foundArrow || !foundWrappedArrow || !foundInterface {
		t.Fatalf("missing expected symbols: %+v", syms)
	}
}
