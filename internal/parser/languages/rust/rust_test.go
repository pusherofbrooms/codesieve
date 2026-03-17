package rust

import "testing"

func TestParseRustExtractsModulesTraitsImplsAndEnums(t *testing.T) {
	src := []byte(`use std::fmt::Debug;

mod auth {
    pub trait Login {
        fn login(&self, user: &str) -> bool;
    }

    pub enum State {
        Ready,
        Failed,
    }

    pub struct Service;

    impl Login for Service {
        fn login(&self, user: &str) -> bool {
            !user.is_empty()
        }
    }
}
`)
	syms, err := Parse("auth.rs", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	found := map[string]bool{}
	for _, s := range syms {
		found[s.Kind+":"+s.QualifiedName] = true
	}
	for _, key := range []string{
		"import:std::fmt::Debug",
		"module:auth",
		"trait:auth.Login",
		"method:auth.Login.login",
		"enum:auth.State",
		"variant:auth.State.Ready",
		"struct:auth.Service",
		"method:auth.Service.login",
	} {
		if !found[key] {
			t.Fatalf("missing expected symbol %q in %+v", key, syms)
		}
	}
}
