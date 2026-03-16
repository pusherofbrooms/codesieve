package fixture

type AuthService struct{}

func Login(user string) error {
	return nil
}

func (a *AuthService) Logout() error {
	return nil
}
