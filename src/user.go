package main

import (
	"encoding/json"

	"github.com/go-pg/pg/v10"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

type User struct {
	Uuid        string `json:"uuid"`
	Login       string `json:"login"`
	Password    string `json:"-"`
	Online      bool   `json:"online"`
	ChannelUuid string `json:"-"`
	Nickname    string `json:"nickname"`
	AvatarUuid  string `json:"avatarUuid"`
	Bio         string `json:"bio"`
}

func (s *Server) HttpGetUsers(ctx *fasthttp.RequestCtx) {
	token := string(ctx.Request.Header.Peek("token"))

	_, err := s.GetUserUuidByToken(token)
	if err != nil {
		if err == pg.ErrNoRows {
			ctx.Error("", fasthttp.StatusUnauthorized)
		} else {
			HttpInternalServerError(ctx, err)
		}
		return
	}

	var users []User
	err = s.Db.Model(&users).Select()
	if err != nil {
		HttpInternalServerError(ctx, err)
		return
	}

	json, err := json.Marshal(users)
	if err != nil {
		HttpInternalServerError(ctx, err)
		return
	}

	ctx.Write(json)
}

func (s *Server) HttpUserLogin(ctx *fasthttp.RequestCtx) {
	login := string(ctx.FormValue("login"))
	password := string(ctx.FormValue("password"))

	if len(login) == 0 || len(password) == 0 {
		ctx.Error("", fasthttp.StatusBadRequest)
		return
	}

	var uuid string
	_, err := s.Db.QueryOne(pg.Scan(&uuid), "SELECT uuid FROM users WHERE login ILIKE ? and password = ?", login, hashPassword(password))
	if err != nil {
		if err == pg.ErrNoRows {
			ctx.Error("", fasthttp.StatusUnauthorized)
		} else {
			HttpInternalServerError(ctx, err)
		}
		return
	}

	token := &Token{
		Token:    randomHash(),
		UserUuid: uuid,
	}
	_, err = s.Db.Model(token).Insert()
	if err != nil {
		HttpInternalServerError(ctx, err)
		return
	}

	ctx.WriteString(token.Token)
}

func (s *Server) HttpUserRegister(ctx *fasthttp.RequestCtx) {
	login := string(ctx.FormValue("login"))
	password := string(ctx.FormValue("password"))

	if len(login) == 0 || len(password) == 0 {
		ctx.Error("", fasthttp.StatusBadRequest)
		return
	}

	var exists bool

	_, err := s.Db.QueryOne(pg.Scan(&exists), "SELECT EXISTS(SELECT 1 FROM users WHERE login = ?)", login)
	if err != nil {
		HttpInternalServerError(ctx, err)
		return
	}

	if exists {
		ctx.Error("", fasthttp.StatusConflict)
		return
	}

	user := User{
		Uuid:     uuid.New().String(),
		Login:    login,
		Password: hashPassword(password),
	}

	_, err = s.Db.Model(&user).Insert()
	if err != nil {
		HttpInternalServerError(ctx, err)
		return
	}

	go func() {
		s.Hub.Broadcast <- Packet{
			Type: PACKET_TYPE_ADD_USERS,
			Data: []User{user},
		}
	}()

	token := &Token{
		Token:    randomHash(),
		UserUuid: user.Uuid,
	}
	_, err = s.Db.Model(token).Insert()
	if err != nil {
		HttpInternalServerError(ctx, err)
		return
	}

	ctx.WriteString(token.Token)
}

func (s *Server) HttpUserProfile(ctx *fasthttp.RequestCtx) {
	token := string(ctx.Request.Header.Peek("token"))

	user, err := s.GetUserByToken(token)
	if err != nil {
		if err == pg.ErrNoRows {
			ctx.Error("", fasthttp.StatusUnauthorized)
		} else {
			HttpInternalServerError(ctx, err)
		}
		return
	}

	user.Nickname = string(ctx.FormValue("nickname"))
	user.Bio = string(ctx.FormValue("bio"))

	_, err = s.Db.Model(user).WherePK().Column("nickname", "bio").Update()
	if err != nil {
		HttpInternalServerError(ctx, err)
		return
	}

	go func() {
		s.Hub.Broadcast <- Packet{
			Type: PACKET_TYPE_UPDATE_USERS,
			Data: []User{*user},
		}
	}()
}
