package typescript

import "testing"

func TestParseTypeScriptTSXByExtension(t *testing.T) {
	src := []byte(`interface Props { name: string }
export const Component = ({name}: Props) => <div>{name}</div>
`)
	syms, err := Parse("component.tsx", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	found := map[string]bool{}
	for _, s := range syms {
		found[s.Kind+":"+s.QualifiedName] = true
	}
	for _, key := range []string{"interface:Props", "function:Component"} {
		if !found[key] {
			t.Fatalf("missing expected symbol %q in %+v", key, syms)
		}
	}
}
