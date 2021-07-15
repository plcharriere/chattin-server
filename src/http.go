package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"strconv"

	"github.com/fasthttp/router"
	"github.com/go-pg/pg/v10"
	"github.com/google/uuid"
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
}

func (server *Server) HandleFastHTTP(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
	ctx.Response.Header.Set("Access-Control-Allow-Headers", "*")
	server.Router.Handler(ctx)
}

type CredentialsForm struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (s *Server) HttpLogin(ctx *fasthttp.RequestCtx) {
	var form CredentialsForm
	err := json.Unmarshal(ctx.Request.Body(), &form)
	if err != nil {
		log.Println(err)
		return
	}

	if len(form.Login) > 0 && len(form.Password) > 0 {
		hash := sha256.Sum256([]byte(form.Password))
		hashedPassword := hex.EncodeToString(hash[:])

		var uuid string
		_, err := s.Db.QueryOne(pg.Scan(&uuid), "SELECT uuid FROM users WHERE login = ? and password = ?", form.Login, hashedPassword)
		if err != nil {
			log.Print(err)
			ctx.WriteString("-1")
		} else {
			hash := sha256.Sum256([]byte(randomString(64)))
			token := hex.EncodeToString(hash[:])

			userToken := &UserToken{
				Token:    token,
				UserUuid: uuid,
			}
			_, err = s.Db.Model(userToken).Insert()
			panicIf(err)

			ctx.WriteString(token)
		}
	} else {
		ctx.WriteString("-1")
	}
}

func (s *Server) HttpRegister(ctx *fasthttp.RequestCtx) {
	var form CredentialsForm
	err := json.Unmarshal(ctx.Request.Body(), &form)
	if err != nil {
		log.Println(err)
		return
	}

	if len(form.Login) < 1 {
		ctx.Error("MISSING_LOGIN", fasthttp.StatusOK)
	} else if len(form.Password) < 1 {
		ctx.Error("MISSING_PASSWORD", fasthttp.StatusOK)
	} else {
		var exists bool
		_, err := s.Db.QueryOne(pg.Scan(&exists), "SELECT EXISTS(SELECT 1 FROM users WHERE login = ?)", form.Login)
		panicIf(err)

		if exists {
			ctx.Error("LOGIN_ALREADY_TAKEN", fasthttp.StatusOK)
		} else {
			hash := sha256.Sum256([]byte(form.Password))
			hashedPassword := hex.EncodeToString(hash[:])
			user := User{
				Uuid:     uuid.New().String(),
				Login:    form.Login,
				Password: hashedPassword,
			}
			_, err := s.Db.Model(&user).Insert()
			panicIf(err)

			s.Hub.Broadcast <- Packet{
				Type: PACKET_TYPE_ADD_USERS,
				Data: []User{user},
			}

			hash = sha256.Sum256([]byte(randomString(64)))
			token := hex.EncodeToString(hash[:])

			userToken := &UserToken{
				Token:    token,
				UserUuid: user.Uuid,
			}
			_, err = s.Db.Model(userToken).Insert()
			panicIf(err)

			ctx.WriteString(token)
		}
	}
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

	_, err = s.Db.Model(user).WherePK().Update()
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

func (s *Server) HttpGetAvatars(ctx *fasthttp.RequestCtx) {
	token := string(ctx.Request.Header.Peek("token"))

	userUuid, err := s.GetUserUuidByToken(token)
	if err != nil {
		log.Print(err)
		return
	}

	var avatars []UserAvatar
	err = s.Db.Model(&avatars).Where("user_uuid = ?", userUuid).Select()
	if err != nil {
		log.Print(err)
		return
	}

	var uuids []string
	for _, avatar := range avatars {
		uuids = append(uuids, avatar.Uuid)
	}

	json, err := json.Marshal(uuids)
	if err != nil {
		log.Print(err)
		return
	}

	ctx.Write(json)
}

func (s *Server) HttpPostAvatar(ctx *fasthttp.RequestCtx) {
	token := string(ctx.FormValue("token"))

	user, err := s.GetUserByToken(token)
	if err != nil {
		ctx.WriteString("-1")
		return
	}

	fileHeader, err := ctx.FormFile("file")
	if err == nil {
		validTypes := map[string]bool{
			"image/png":  true,
			"image/jpeg": true,
			"image/gif":  true,
			"image/webp": true,
		}

		fileType := fileHeader.Header.Get("Content-Type")
		if _, ok := validTypes[fileType]; !ok {
			log.Print("bad type")
			return
		}

		file, err := fileHeader.Open()
		if err != nil {
			log.Println(err)
			return
		}

		var buf bytes.Buffer
		io.Copy(&buf, file)

		file.Close()

		userAvatar := &UserAvatar{
			Uuid:     uuid.New().String(),
			UserUuid: user.Uuid,
			Type:     fileType,
			Data:     buf.Bytes(),
		}
		_, err = s.Db.Model(userAvatar).Insert()
		if err != nil {
			log.Print(err)
			return
		}

		user.AvatarUuid = userAvatar.Uuid
	} else {
		user.AvatarUuid = ""
	}

	_, err = s.Db.Model(user).WherePK().Column("avatar_uuid").Update()
	if err != nil {
		log.Print(err)
		return
	}

	go func() {
		s.Hub.Broadcast <- Packet{
			Type: PACKET_TYPE_UPDATE_USERS,
			Data: []User{*user},
		}
	}()

	ctx.WriteString("1")
}

func (s *Server) HttpGetAvatar(ctx *fasthttp.RequestCtx) {
	uuid := ctx.UserValue("uuid")
	if uuid == nil {
		return
	}

	userAvatar := &UserAvatar{
		Uuid: uuid.(string),
	}
	err := s.Db.Model(userAvatar).WherePK().Select()
	if err != nil {
		log.Print(err)
		return
	}

	ctx.Response.Header.Set("Content-Type", userAvatar.Type)
	ctx.Write(userAvatar.Data)
}
