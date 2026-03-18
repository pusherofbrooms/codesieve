const std = @import("std");

pub const Client = struct {
    token: []const u8,

    pub fn login(self: *Client, user: []const u8) bool {
        return user.len > 0 and self.token.len > 0;
    }
};

pub fn buildClient(token: []const u8) Client {
    return .{ .token = token };
}
