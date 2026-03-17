package app

import (
	"testing"

	"github.com/pusherofbrooms/codesieve/internal/parser"
)

func TestParseGoSymbols(t *testing.T) {
	src := []byte(`package sample

type User struct {}

func Authenticate(token string) error { return nil }

func (u *User) Login(name string) bool { return true }
`)
	syms, lang, err := parser.ParseSymbols("sample.go", src)
	if err != nil {
		t.Fatalf("ParseSymbols error: %v", err)
	}
	if lang != "go" {
		t.Fatalf("lang = %q", lang)
	}
	if len(syms) < 3 {
		t.Fatalf("expected at least 3 symbols, got %d", len(syms))
	}
	if syms[0].Name != "User" || syms[0].Kind != "struct" {
		t.Fatalf("unexpected first symbol: %+v", syms[0])
	}
	if syms[2].QualifiedName != "User.Login" || syms[2].Kind != "method" || syms[2].ParentID != "User" {
		t.Fatalf("unexpected method symbol: %+v", syms[2])
	}
}

func TestParsePythonSymbols(t *testing.T) {
	src := []byte(`class Auth:
    @decorator
    async def login(self, user):
        return True

def helper(name):
    return name
`)
	syms, lang, err := parser.ParseSymbols("auth.py", src)
	if err != nil {
		t.Fatalf("ParseSymbols error: %v", err)
	}
	if lang != "python" {
		t.Fatalf("lang = %q", lang)
	}
	if len(syms) != 3 {
		t.Fatalf("expected 3 symbols, got %d", len(syms))
	}
	if syms[1].QualifiedName != "Auth.login" || syms[1].Kind != "method" || syms[1].ParentID != "Auth" {
		t.Fatalf("unexpected symbol: %+v", syms[1])
	}
}

func TestParseTypeScriptSymbols(t *testing.T) {
	src := []byte(`export interface User {
  name: string
}

export class Client {
  login(token: string) {
    return token
  }
}

export const fetchUser = (id: string) => id
export const routes = lazy(() => createRoutes())
`)
	syms, lang, err := parser.ParseSymbols("client.ts", src)
	if err != nil {
		t.Fatalf("ParseSymbols error: %v", err)
	}
	if lang != "typescript" {
		t.Fatalf("lang = %q", lang)
	}
	if len(syms) < 5 {
		t.Fatalf("expected at least 5 symbols, got %d", len(syms))
	}
	foundMethod := false
	foundArrow := false
	foundWrappedArrow := false
	foundInterface := false
	for _, sym := range syms {
		switch {
		case sym.QualifiedName == "Client.login" && sym.Kind == "method" && sym.ParentID == "Client":
			foundMethod = true
		case sym.Name == "fetchUser" && sym.Kind == "function":
			foundArrow = true
		case sym.Name == "routes" && sym.Kind == "function":
			foundWrappedArrow = true
		case sym.Name == "User" && sym.Kind == "interface":
			foundInterface = true
		}
	}
	if !foundMethod || !foundArrow || !foundWrappedArrow || !foundInterface {
		t.Fatalf("missing expected symbols: %+v", syms)
	}
}

func TestParseRustSymbols(t *testing.T) {
	src := []byte(`use std::collections::HashMap;

struct AuthService {
    token: String,
}

impl AuthService {
    fn login(&self, user: &str) -> bool {
        !user.is_empty()
    }
}

fn build_index() -> HashMap<String, usize> {
    HashMap::new()
}
`)
	syms, lang, err := parser.ParseSymbols("auth.rs", src)
	if err != nil {
		t.Fatalf("ParseSymbols error: %v", err)
	}
	if lang != "rust" {
		t.Fatalf("lang = %q", lang)
	}

	found := map[string]bool{}
	for _, sym := range syms {
		found[sym.Kind+":"+sym.QualifiedName] = true
	}
	for _, key := range []string{
		"import:std::collections::HashMap",
		"struct:AuthService",
		"method:AuthService.login",
		"function:build_index",
	} {
		if !found[key] {
			t.Fatalf("missing expected symbol %q in %+v", key, syms)
		}
	}
}

func TestParseJavaSymbols(t *testing.T) {
	src := []byte(`package sample.auth;

import java.util.List;

public class AuthService {
  private final String authHeader = "X-Auth-Header";

  public AuthService() {}

  public boolean login(String user) {
    return true;
  }

  public boolean login(String user, int retries) {
    return retries > 0 && user != null;
  }

  static class Token {
    String value;
  }
}
`)
	syms, lang, err := parser.ParseSymbols("AuthService.java", src)
	if err != nil {
		t.Fatalf("ParseSymbols error: %v", err)
	}
	if lang != "java" {
		t.Fatalf("lang = %q", lang)
	}

	found := map[string]bool{}
	for _, sym := range syms {
		found[sym.Kind+":"+sym.QualifiedName] = true
	}
	for _, key := range []string{
		"package:sample.auth",
		"import:java.util.List",
		"class:AuthService",
		"constructor:AuthService.AuthService()",
		"method:AuthService.login(String)",
		"method:AuthService.login(String,int)",
		"field:AuthService.authHeader",
		"class:AuthService.Token",
		"field:AuthService.Token.value",
	} {
		if !found[key] {
			t.Fatalf("missing expected symbol %q in %+v", key, syms)
		}
	}
}

func TestParseBashSymbols(t *testing.T) {
	src := []byte(`#!/usr/bin/env bash

API_TOKEN="token"
project_name="dev"
export AUTH_HEADER
source ./lib/common.sh

login() {
  local user="$1"
  RETRY_COUNT=3
  . ./lib/inner.sh
  echo "$user"
}

function logout {
  echo "bye"
}
`)
	syms, lang, err := parser.ParseSymbols("script.sh", src)
	if err != nil {
		t.Fatalf("ParseSymbols error: %v", err)
	}
	if lang != "bash" {
		t.Fatalf("lang = %q", lang)
	}
	if len(syms) != 6 {
		t.Fatalf("expected 6 symbols, got %d (%+v)", len(syms), syms)
	}

	found := map[string]bool{}
	for _, sym := range syms {
		if sym.ParentID != "" && sym.ParentID != "script:script.sh" {
			t.Fatalf("unexpected parent for symbol %+v", sym)
		}
		found[sym.Kind+":"+sym.Name] = true
	}

	for _, key := range []string{"script:script:script.sh", "variable:API_TOKEN", "variable:AUTH_HEADER", "include:source:./lib/common.sh", "function:login", "function:logout"} {
		if !found[key] {
			t.Fatalf("missing expected symbol %q in %+v", key, syms)
		}
	}
	if found["include:source:./lib/inner.sh"] || found["variable:project_name"] || found["variable:RETRY_COUNT"] || found["variable:user"] {
		t.Fatalf("unexpected noisy symbols in %+v", syms)
	}
}

func TestParseBashSymbolsFromShebangWithoutExtension(t *testing.T) {
	src := []byte(`#!/usr/bin/env bash

deploy() {
  echo ok
}
`)
	syms, lang, err := parser.ParseSymbols("deploy", src)
	if err != nil {
		t.Fatalf("ParseSymbols error: %v", err)
	}
	if lang != "bash" {
		t.Fatalf("lang = %q", lang)
	}
	if len(syms) < 2 {
		t.Fatalf("expected at least script + function symbols, got %d (%+v)", len(syms), syms)
	}
}

func TestParseYAMLCloudFormationSymbols(t *testing.T) {
	src := []byte(`AWSTemplateFormatVersion: "2010-09-09"
Description: Sample template
Parameters:
  EnvName:
    Type: String
Resources:
  AppBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !Sub "${EnvName}-app"
Outputs:
  BucketName:
    Value:
      Ref: AppBucket
`)
	syms, lang, err := parser.ParseSymbols("template.yaml", src)
	if err != nil {
		t.Fatalf("ParseSymbols error: %v", err)
	}
	if lang != "yaml" {
		t.Fatalf("lang = %q", lang)
	}

	found := map[string]bool{}
	for _, sym := range syms {
		found[sym.Kind+":"+sym.QualifiedName] = true
	}
	for _, key := range []string{
		"template:template:template.yaml",
		"section:Resources",
		"parameter:Parameters.EnvName",
		"resource:Resources.AppBucket",
		"output:Outputs.BucketName",
		"reference:Resources.AppBucket.ref.EnvName",
		"reference:Outputs.BucketName.ref.AppBucket",
	} {
		if !found[key] {
			t.Fatalf("missing expected symbol %q in %+v", key, syms)
		}
	}
}

func TestParseYAMLGenericConfigSymbols(t *testing.T) {
	src := []byte(`service:
  name: codesieve
  features:
    search:
      enabled: true
`)
	syms, lang, err := parser.ParseSymbols("config.yml", src)
	if err != nil {
		t.Fatalf("ParseSymbols error: %v", err)
	}
	if lang != "yaml" {
		t.Fatalf("lang = %q", lang)
	}
	if len(syms) < 4 {
		t.Fatalf("expected generic yaml key symbols, got %d (%+v)", len(syms), syms)
	}
	if syms[0].Kind != "document" {
		t.Fatalf("expected document root for generic yaml, got %+v", syms[0])
	}
}

func TestParseJSONCloudFormationSymbols(t *testing.T) {
	src := []byte(`{
  "AWSTemplateFormatVersion": "2010-09-09",
  "Parameters": {
    "EnvName": {
      "Type": "String"
    }
  },
  "Resources": {
    "AppBucket": {
      "Type": "AWS::S3::Bucket",
      "Properties": {
        "BucketName": {
          "Fn::Sub": "${EnvName}-app"
        }
      }
    }
  },
  "Outputs": {
    "BucketName": {
      "Value": {
        "Ref": "AppBucket"
      }
    }
  }
}`)
	syms, lang, err := parser.ParseSymbols("template.json", src)
	if err != nil {
		t.Fatalf("ParseSymbols error: %v", err)
	}
	if lang != "json" {
		t.Fatalf("lang = %q", lang)
	}

	found := map[string]bool{}
	for _, sym := range syms {
		found[sym.Kind+":"+sym.QualifiedName] = true
	}
	for _, key := range []string{
		"template:template:template.json",
		"section:Resources",
		"parameter:Parameters.EnvName",
		"resource:Resources.AppBucket",
		"output:Outputs.BucketName",
		"reference:Resources.AppBucket.ref.EnvName",
		"reference:Outputs.BucketName.ref.AppBucket",
	} {
		if !found[key] {
			t.Fatalf("missing expected symbol %q in %+v", key, syms)
		}
	}
}

func TestParseJSONGenericConfigSymbols(t *testing.T) {
	src := []byte(`{
  "service": {
    "name": "codesieve",
    "features": {
      "search": {
        "enabled": true
      }
    }
  }
}`)
	syms, lang, err := parser.ParseSymbols("config.json", src)
	if err != nil {
		t.Fatalf("ParseSymbols error: %v", err)
	}
	if lang != "json" {
		t.Fatalf("lang = %q", lang)
	}
	if len(syms) < 4 {
		t.Fatalf("expected generic json key symbols, got %d (%+v)", len(syms), syms)
	}
	if syms[0].Kind != "document" {
		t.Fatalf("expected document root for generic json, got %+v", syms[0])
	}
}
