package csharp

import "testing"

func TestParseCSharpExtractsNamespacesTypesAndMembers(t *testing.T) {
	src := []byte(`using System;
using System.Collections.Generic;

namespace Acme.Auth;

public class AuthService {
    private readonly string authHeader = "X-Auth-Header";
    public string Token { get; init; }
    public event EventHandler? LoggedIn;

    public AuthService(string token) {
        Token = token;
    }

    public bool Login(string user, int retries) {
        return retries > 0 && !string.IsNullOrWhiteSpace(user);
    }
}

public enum AuthState {
    Ready,
    Failed,
}
`)

	syms, err := Parse("AuthService.cs", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	found := map[string]bool{}
	for _, s := range syms {
		found[s.Kind+":"+s.QualifiedName] = true
	}

	for _, key := range []string{
		"import:System",
		"import:System.Collections.Generic",
		"namespace:Acme.Auth",
		"class:AuthService",
		"field:AuthService.authHeader",
		"property:AuthService.Token",
		"event:AuthService.LoggedIn",
		"constructor:AuthService.AuthService(string)",
		"method:AuthService.Login(string,int)",
		"enum:AuthState",
		"constant:AuthState.Ready",
	} {
		if !found[key] {
			t.Fatalf("missing expected symbol %q in %+v", key, syms)
		}
	}
}
