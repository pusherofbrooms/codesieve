package json

import "testing"

func TestParseJSONCloudFormationTemplate(t *testing.T) {
	src := []byte(`{
  "AWSTemplateFormatVersion": "2010-09-09",
  "Resources": {
    "AppBucket": {"Type": "AWS::S3::Bucket"}
  }
}`)
	syms, err := Parse("template.json", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	found := map[string]bool{}
	for _, s := range syms {
		found[s.Kind+":"+s.QualifiedName] = true
	}
	for _, key := range []string{"template:template:template.json", "section:Resources", "resource:Resources.AppBucket"} {
		if !found[key] {
			t.Fatalf("missing expected symbol %q in %+v", key, syms)
		}
	}
}

func TestParseJSONGenericConfigEmitsKeyHierarchy(t *testing.T) {
	src := []byte(`{
  "service": {
    "features": {
      "search": {
        "enabled": true
      }
    }
  }
}`)
	syms, err := Parse("config.json", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	found := map[string]bool{}
	for _, s := range syms {
		found[s.Kind+":"+s.QualifiedName] = true
	}
	for _, key := range []string{"document:document:config.json", "key:service", "key:service.features.search.enabled"} {
		if !found[key] {
			t.Fatalf("missing expected symbol %q in %+v", key, syms)
		}
	}
}
