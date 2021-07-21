package main

import (
	"encoding/json"
	"log"
	"strconv"

	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

func (server *Server) SetupFastHTTPRouter() {
	server.Router = router.New()
	server.Router.GET("/ws", server.HttpHandleWebSocket)
	server.Router.POST("/login", server.HttpLogin)
	server.Router.POST("/register", server.HttpRegister)
	server.Router.GET("/users", server.HttpGetUsers)
	server.Router.GET("/channels", server.HttpGetChannels)
	server.Router.GET("/channels/{uuid}/messages", server.HttpGetChannelMessages)
	server.Router.POST("/users/profile", server.HttpUserProfile)
	server.Router.POST("/avatars", server.HttpPostAvatar)
	server.Router.GET("/avatars", server.HttpGetAvatars)
	server.Router.GET("/avatars/{uuid}", server.HttpGetAvatar)
	server.Router.DELETE("/avatars/{uuid}", server.HttpDeleteAvatar)
}

func (server *Server) HandleFastHTTP(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
	ctx.Response.Header.Set("Access-Control-Allow-Headers", "*")
	ctx.Response.Header.Set("Access-Control-Allow-Methods", "*")
	server.Router.Handler(ctx)
}

func HttpInternalServerError(ctx *fasthttp.RequestCtx, err error) {
	log.Print(err)
	ctx.Error("", fasthttp.StatusInternalServerError)
}

type ProfileForm struct {
	Token    string `json:"token"`
	Nickname string `json:"nickname"`
	Bio      string `json:"bio"`
}

func (s *Server) HttpUserProfile(ctx *fasthttp.RequestCtx) {
	var form ProfileForm
	err := json.Unmarshal(ctx.Request.Body(), &form)
	if err != nil {
		log.Println(err)
		return
	}

	user, err := s.GetUserByToken(form.Token)
	if err != nil {
		ctx.WriteString("-1")
		return
	}

	user.Nickname = form.Nickname
	user.Bio = form.Bio

	_, err = s.Db.Model(user).WherePK().Column("nickname", "bio").Update()
	panicIf(err)

	go func() {
		s.Hub.Broadcast <- Packet{
			Type: PACKET_TYPE_UPDATE_USERS,
			Data: []User{*user},
		}
	}()

	ctx.WriteString("1")
}

func (s *Server) HttpGetUsers(ctx *fasthttp.RequestCtx) {
	token := string(ctx.Request.Header.Peek("token"))
	valid, err := s.IsTokenValid(token)
	if err != nil || !valid {
		return
	}

	var users []User
	err = s.Db.Model(&users).Select()
	if err != nil {
		log.Print(err)
		return
	}

	json, err := json.Marshal(users)
	if err != nil {
		log.Print(err)
		return
	}

	ctx.Write(json)
}

func (s *Server) HttpGetChannels(ctx *fasthttp.RequestCtx) {
	token := string(ctx.Request.Header.Peek("token"))
	valid, err := s.IsTokenValid(token)
	if err != nil || !valid {
		return
	}

	var channels []Channel
	err = s.Db.Model(&channels).Select()
	if err != nil {
		log.Print(err)
		return
	}

	json, err := json.Marshal(channels)
	if err != nil {
		log.Print(err)
		return
	}

	ctx.Write(json)
}

func (s *Server) HttpGetChannelMessages(ctx *fasthttp.RequestCtx) {
	token := string(ctx.Request.Header.Peek("token"))
	valid, err := s.IsTokenValid(token)
	if err != nil || !valid {
		return
	}

	channelUuid := ctx.UserValue("uuid")
	if channelUuid == nil {
		return
	}

	fromMessageUuid := string(ctx.FormValue("from"))

	count := string(ctx.FormValue("count"))
	if len(count) == 0 {
		return
	}

	var messages []Message

	query := s.Db.Model(&messages).Where("channel_uuid = ?", channelUuid)

	if len(fromMessageUuid) > 0 {
		fromMessage := Message{
			Uuid: fromMessageUuid,
		}
		err := s.Db.Model(&fromMessage).WherePK().Select()
		if err != nil {
			log.Print(err)
			return
		}
		query.Where("uuid != ? AND date <= ?", fromMessage.Uuid, fromMessage.Date)
	}

	countInt, err := strconv.Atoi(count)
	if err != nil {
		log.Print(err)
		return
	}

	err = query.Order("date DESC").Limit(countInt).Select()
	if err != nil {
		log.Print(err)
		return
	}

	json, err := json.Marshal(messages)
	if err != nil {
		log.Print(err)
		return
	}

	ctx.Write(json)
}
