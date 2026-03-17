using System;

namespace Acme.Auth;

public class AuthService {
    private readonly string authHeader = "X-Auth-Header";
    public string Token { get; init; }

    public AuthService(string token) {
        Token = token;
    }

    public bool Login(string user) {
        return !string.IsNullOrWhiteSpace(user);
    }
}
