package main

type Token struct {
	Token    string `pg:",pk"`
	UserUuid string `pg:",nopk"`
}

func (server *Server) GetUserUuidByToken(token string) (string, error) {
	userToken := &Token{
		Token: token,
	}

	err := server.Db.Model(userToken).WherePK().Select()
	if err != nil {
		return "", err
	}

	return userToken.UserUuid, nil
}

func (server *Server) GetUserByToken(token string) (*User, error) {
	userToken := &Token{
		Token: token,
	}
	err := server.Db.Model(userToken).WherePK().Select()
	if err != nil {
		return nil, err
	}

	user := &User{
		Uuid: userToken.UserUuid,
	}
	err = server.Db.Model(user).WherePK().ExcludeColumn("password").Select()
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (server *Server) IsTokenValid(token string) (bool, error) {
	userToken := &Token{
		Token: token,
	}
	err := server.Db.Model(userToken).WherePK().Select()
	if err != nil {
		return false, err
	}
	return true, nil
}
