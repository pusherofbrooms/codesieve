package php

import "testing"

func TestParsePHPExtractsNamespaceTypesAndMembers(t *testing.T) {
	src := []byte(`<?php

namespace App\Auth;

use Psr\Log\LoggerInterface;
use Symfony\Component\HttpFoundation\{Request, Response as HttpResponse};

const APP_VERSION = '1.0.0';

class AuthService {
    private string $token;
    public const HEADER = 'X-Auth';

    public function __construct(string $token) {
        $this->token = $token;
    }

    public function login(Request $request): bool {
        return true;
    }
}

enum State: string {
    case Ready = 'ready';
}

function helper(): void {}
`)

	syms, err := Parse("AuthService.php", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	found := map[string]bool{}
	for _, s := range syms {
		found[s.Kind+":"+s.QualifiedName] = true
	}

	for _, key := range []string{
		"namespace:App.Auth",
		"import:Psr.Log.LoggerInterface",
		"import:Symfony.Component.HttpFoundation.Request",
		"import:Symfony.Component.HttpFoundation.Response",
		"constant:App.Auth.APP_VERSION",
		"class:App.Auth.AuthService",
		"property:App.Auth.AuthService.token",
		"constant:App.Auth.AuthService.HEADER",
		"constructor:App.Auth.AuthService.__construct",
		"method:App.Auth.AuthService.login",
		"enum:App.Auth.State",
		"constant:App.Auth.State.Ready",
		"function:App.Auth.helper",
	} {
		if !found[key] {
			t.Fatalf("missing expected symbol %q in %+v", key, syms)
		}
	}
}
