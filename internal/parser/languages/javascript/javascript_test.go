package javascript

import "testing"

func TestParseJavaScriptExtractsClassMethodsAndFunctionLikeVars(t *testing.T) {
	src := []byte(`class Client {
  login(token) {
    return token
  }
}

const fetchUser = (id) => id
const routes = lazy(() => createRoutes())
`)
	syms, err := Parse("client.js", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	found := map[string]bool{}
	for _, s := range syms {
		found[s.Kind+":"+s.QualifiedName] = true
	}
	for _, key := range []string{"class:Client", "method:Client.login", "function:fetchUser", "function:routes"} {
		if !found[key] {
			t.Fatalf("missing expected symbol %q in %+v", key, syms)
		}
	}
}
