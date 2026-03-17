package app

import "testing"

func TestRankSymbolBasicMatchTiers(t *testing.T) {
	item := storedSymbol{Name: "Login", QualifiedName: "Auth.Login", Kind: "function", FilePath: "src/auth.go", StartLine: 10}

	scoreExactQualified := rankSymbol(SearchSymbolOptions{Query: "Auth.Login"}, item)
	scoreExactName := rankSymbol(SearchSymbolOptions{Query: "Login"}, item)
	scorePrefix := rankSymbol(SearchSymbolOptions{Query: "Log"}, item)
	if !(scoreExactQualified > scoreExactName && scoreExactName > scorePrefix) {
		t.Fatalf("unexpected tiering: qualified=%v name=%v prefix=%v", scoreExactQualified, scoreExactName, scorePrefix)
	}

	scoreInsensitive := rankSymbol(SearchSymbolOptions{Query: "login"}, item)
	if scoreInsensitive <= 0 {
		t.Fatalf("expected positive score for case-insensitive match, got %v", scoreInsensitive)
	}

	scoreSensitive := rankSymbol(SearchSymbolOptions{Query: "login", CaseSensitive: true}, item)
	if scoreSensitive != 0 {
		t.Fatalf("expected zero score for case-sensitive mismatch, got %v", scoreSensitive)
	}
}

func TestRankSymbolKindAndPathHeuristics(t *testing.T) {
	baseOpt := SearchSymbolOptions{Query: "Login"}
	fn := storedSymbol{Name: "Login", Kind: "function", FilePath: "src/auth.go", StartLine: 10}
	iface := storedSymbol{Name: "Login", Kind: "interface", FilePath: "src/auth.go", StartLine: 10}
	vendorFn := storedSymbol{Name: "Login", Kind: "function", FilePath: "vendor/pkg/auth.go", StartLine: 10}
	testFn := storedSymbol{Name: "Login", Kind: "function", FilePath: "src/test/java/com/acme/AuthTest.java", StartLine: 10}
	generatedFn := storedSymbol{Name: "Login", Kind: "function", FilePath: "src/generated/auth.pb.go", StartLine: 10}

	sFn := rankSymbol(baseOpt, fn)
	sIface := rankSymbol(baseOpt, iface)
	sVendor := rankSymbol(baseOpt, vendorFn)
	sTest := rankSymbol(baseOpt, testFn)
	sGenerated := rankSymbol(baseOpt, generatedFn)

	if !(sFn > sIface) {
		t.Fatalf("expected function score > interface score, got %v <= %v", sFn, sIface)
	}
	if !(sFn > sVendor) {
		t.Fatalf("expected src/ function score > vendor function score, got %v <= %v", sFn, sVendor)
	}
	if !(sFn > sTest) {
		t.Fatalf("expected production path score > test path score, got %v <= %v", sFn, sTest)
	}
	if !(sFn > sGenerated) {
		t.Fatalf("expected production path score > generated path score, got %v <= %v", sFn, sGenerated)
	}
}

func TestRankSymbolQualifiedOverloadQueryPrefersQualifiedName(t *testing.T) {
	opt := SearchSymbolOptions{Query: "AuthService.login(String)"}
	bare := storedSymbol{Name: "login", QualifiedName: "AuthService.login", Kind: "method", FilePath: "src/main/java/com/acme/AuthService.java", StartLine: 20}
	overload := storedSymbol{Name: "login", QualifiedName: "AuthService.login(String)", Kind: "method", FilePath: "src/main/java/com/acme/AuthService.java", StartLine: 30}

	sBare := rankSymbol(opt, bare)
	sOverload := rankSymbol(opt, overload)
	if !(sOverload > sBare) {
		t.Fatalf("expected overload-specific qname score > bare qname score, got %v <= %v", sOverload, sBare)
	}
}
