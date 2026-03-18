package zig

import "testing"

func TestParseZigExtractsTypesMembersFunctionsAndTests(t *testing.T) {
	src := []byte(`const std = @import("std");

pub const Mode = enum {
    debug,
    release,
};

pub const Client = struct {
    token: []const u8,

    pub fn init(token: []const u8) Client {
        return .{ .token = token };
    }

    pub fn login(self: *Client, user: []const u8) bool {
        return user.len > 0;
    }
};

pub fn top(name: []const u8) void {}

test "login" {
    _ = Client.init("x");
}
`)

	syms, err := Parse("basic.zig", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	found := map[string]bool{}
	for _, s := range syms {
		found[s.Kind+":"+s.QualifiedName] = true
	}

	for _, key := range []string{
		"constant:std",
		"enum:Mode",
		"variant:Mode.debug",
		"struct:Client",
		"field:Client.token",
		"method:Client.init",
		"method:Client.login",
		"function:top",
		"test:test.login",
	} {
		if !found[key] {
			t.Fatalf("missing expected symbol %q in %+v", key, syms)
		}
	}
}
