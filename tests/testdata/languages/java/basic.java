package sample;

public class AuthService {
  private static final String AUTH_HEADER = "X-Auth-Header";

  public boolean login(String user) {
    return user != null && !user.isEmpty();
  }

  public void logout() {
    // noop
  }
}
