package auth

import "crypto/subtle"

type User struct {
	Username string
	Password string
}

type Authenticator struct {
	userMap map[string][][]byte
}

func NewAuthenticator(users []User) *Authenticator {
	if len(users) == 0 {
		return nil
	}
	au := &Authenticator{
		userMap: make(map[string][][]byte),
	}
	for _, user := range users {
		au.userMap[user.Username] = append(au.userMap[user.Username], []byte(user.Password))
	}
	return au
}

func (au *Authenticator) Verify(username string, password string) bool {
	passwordList, ok := au.userMap[username]
	if !ok {
		return false
	}
	for _, candidate := range passwordList {
		if subtle.ConstantTimeCompare(candidate, []byte(password)) == 1 {
			return true
		}
	}
	return false
}
