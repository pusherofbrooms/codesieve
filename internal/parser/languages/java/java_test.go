package java

import "testing"

func TestParseJavaMethodOverloadsProduceDistinctQualifiedNames(t *testing.T) {
	src := []byte(`class A {
  void f(String a) {}
  void f(String a, int b) {}
}`)
	syms, err := Parse("A.java", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	found := map[string]bool{}
	for _, s := range syms {
		found[s.Kind+":"+s.QualifiedName] = true
	}
	if !found["method:A.f(String)"] || !found["method:A.f(String,int)"] {
		t.Fatalf("expected overload-qualified symbols, got %+v", syms)
	}
}

func TestParseJavaExtractsPackageAndImport(t *testing.T) {
	src := []byte(`package x.y;
import java.util.Map;
class A {}`)
	syms, err := Parse("A.java", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	found := map[string]bool{}
	for _, s := range syms {
		found[s.Kind+":"+s.QualifiedName] = true
	}
	if !found["package:x.y"] {
		t.Fatalf("missing package symbol in %+v", syms)
	}
	if !found["import:java.util.Map"] {
		t.Fatalf("missing import symbol in %+v", syms)
	}
}
