package ruby

import "testing"

func TestParseRubyExtractsModulesClassesMethodsAndConstants(t *testing.T) {
	src := []byte(`module Auth
  VERSION = "1.0"

  class Service
    def initialize(client)
      @client = client
    end

    def login(user)
      true
    end

    def self.build
      new(nil)
    end
  end
end

def helper(name)
  name
end
`)

	syms, err := Parse("auth.rb", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	found := map[string]bool{}
	for _, s := range syms {
		found[s.Kind+":"+s.QualifiedName] = true
	}

	for _, key := range []string{
		"module:Auth",
		"constant:Auth.VERSION",
		"class:Auth.Service",
		"constructor:Auth.Service.initialize",
		"method:Auth.Service.login",
		"method:Auth.Service.build",
		"function:helper",
	} {
		if !found[key] {
			t.Fatalf("missing expected symbol %q in %+v", key, syms)
		}
	}
}
