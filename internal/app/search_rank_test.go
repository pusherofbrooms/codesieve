package app

import "testing"

func TestRankSymbolBasicMatchTiers(t *testing.T) {
	item := storedSymbol{Name: "Login", QualifiedName: "Auth.Login", Kind: "function", FilePath: "src/auth.go", StartLine: 10}

	// exact
	scoreExact := rankSymbol(SearchSymbolOptions{Query: "Login"}, item)
	if scoreExact <= 0 {
		t.Fatalf("expected positive score for exact match, got %v", scoreExact)
	}

	// case-insensitive
	scoreInsensitive := rankSymbol(SearchSymbolOptions{Query: "login"}, item)
	if scoreInsensitive <= 0 {
		t.Fatalf("expected positive score for case-insensitive match, got %v", scoreInsensitive)
	}

	// case-sensitive mismatch
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

	sFn := rankSymbol(baseOpt, fn)
	sIface := rankSymbol(baseOpt, iface)
	sVendor := rankSymbol(baseOpt, vendorFn)

	if !(sFn > sIface) {
		t.Fatalf("expected function score > interface score, got %v <= %v", sFn, sIface)
	}
	if !(sFn > sVendor) {
		t.Fatalf("expected src/ function score > vendor function score, got %v <= %v", sFn, sVendor)
	}
}
