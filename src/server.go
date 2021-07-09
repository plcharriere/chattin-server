package main

import (
	"github.com/fasthttp/router"
	"github.com/go-pg/pg/v10"
)

type Server struct {
	Db       *pg.DB
	Router   *router.Router
	Hub      *Hub
	Channels []*Channel
}

func (server *Server) GetChannelByUuid(uuid string) *Channel {
	for _, channel := range server.Channels {
		if channel.Uuid == uuid {
			return channel
		}
	}
	return nil
}

func (server *Server) GetUserUuidByToken(token string) (string, error) {
	userToken := &UserToken{
		Token: token,
	}
	err := server.Db.Model(userToken).WherePK().Select()
	if err != nil {
		return "", err
	}

	return userToken.UserUuid, nil
}

func (server *Server) GetUserByToken(token string) (*User, error) {
	userToken := &UserToken{
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
