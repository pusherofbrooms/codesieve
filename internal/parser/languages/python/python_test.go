package python

import "testing"

func TestParsePythonDecoratedClassMethodAndTopLevelFunction(t *testing.T) {
	src := []byte(`class Auth:
    @decorator
    async def login(self, user):
        return True

def helper(name):
    return name
`)
	syms, err := Parse("auth.py", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	found := map[string]bool{}
	for _, s := range syms {
		found[s.Kind+":"+s.QualifiedName] = true
	}
	for _, key := range []string{"class:Auth", "method:Auth.login", "function:helper"} {
		if !found[key] {
			t.Fatalf("missing expected symbol %q in %+v", key, syms)
		}
	}
}
