package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/fasthttp/router"
	"github.com/fasthttp/websocket"
	"github.com/go-pg/pg/v10"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

func (server *Server) SetupFastHTTPRouter() {
	server.Router = router.New()
	server.Router.GET("/ws", server.HttpHandleWebSocket)
	server.Router.POST("/register", server.HttpUserRegister)
	server.Router.POST("/login", server.HttpUserLogin)
	server.Router.POST("/user/profile", server.HttpUserProfile)
	server.Router.POST("/avatars", server.HttpUserPostAvatar)
	server.Router.GET("/avatars/{uuid}", server.HttpUserGetAvatar)
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

func (s *Server) HttpUserLogin(ctx *fasthttp.RequestCtx) {
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

func (s *Server) HttpUserRegister(ctx *fasthttp.RequestCtx) {
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

	userToken := &UserToken{
		Token: form.Token,
	}
	err = s.Db.Model(userToken).WherePK().Select()
	if err != nil {
		ctx.WriteString("-1")
		return
	}

	user := User{
		Uuid:     userToken.UserUuid,
		Nickname: form.Nickname,
		Bio:      form.Bio,
	}
	_, err = s.Db.Model(&user).WherePK().Column("nickname", "bio").Update()
	panicIf(err)

	user = User{
		Uuid: userToken.UserUuid,
	}
	err = s.Db.Model(&user).WherePK().Select()
	panicIf(err)

	go func() {
		s.Hub.Broadcast <- Packet{
			Type: PACKET_TYPE_UPDATE_USERS,
			Data: []User{user},
		}
	}()

	ctx.WriteString("1")
}

func (s *Server) HttpUserPostAvatar(ctx *fasthttp.RequestCtx) {
	token := string(ctx.FormValue("token"))
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		log.Print(err)
		return
	}

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

	userToken := &UserToken{
		Token: token,
	}
	err = s.Db.Model(userToken).WherePK().Select()
	if err != nil {
		ctx.WriteString("-1")
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
		UserUuid: userToken.UserUuid,
		Type:     fileType,
		Data:     buf.Bytes(),
	}
	_, err = s.Db.Model(userAvatar).Insert()
	if err != nil {
		log.Print(err)
		return
	}

	user := User{
		Uuid:       userToken.UserUuid,
		AvatarUuid: userAvatar.Uuid,
	}
	_, err = s.Db.Model(&user).WherePK().Column("avatar_uuid").Update()
	if err != nil {
		log.Print(err)
		return
	}

	user = User{
		Uuid: userToken.UserUuid,
	}
	err = s.Db.Model(&user).WherePK().Select()
	panicIf(err)

	go func() {
		s.Hub.Broadcast <- Packet{
			Type: PACKET_TYPE_UPDATE_USERS,
			Data: []User{user},
		}
	}()

	ctx.WriteString("1")
}

func (s *Server) HttpUserGetAvatar(ctx *fasthttp.RequestCtx) {
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

var upgrader = websocket.FastHTTPUpgrader{
	CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
		return true
	},
}

func (s *Server) HttpHandleWebSocket(ctx *fasthttp.RequestCtx) {
	err := upgrader.Upgrade(ctx, func(conn *websocket.Conn) {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			conn.Close()
			return
		}

		var packet Packet
		err = json.Unmarshal(msg, &packet)
		if err != nil {
			log.Println(err)
			conn.Close()
			return
		}

		if packet.Type != PACKET_TYPE_AUTH {
			conn.Close()
			return
		}

		token := fmt.Sprintf("%s", packet.Data)

		log.Println("AUTH:", token)

		userToken := &UserToken{
			Token: token,
		}
		err = s.Db.Model(userToken).WherePK().Select()
		if err != nil {
			log.Println("WRONG TOKEN:", token)
			conn.WriteJSON(Packet{
				Type: PACKET_TYPE_AUTH,
				Data: false,
			})
			conn.Close()
			return
		}

		user := &User{
			Uuid: userToken.UserUuid,
		}
		err = s.Db.Model(user).WherePK().ExcludeColumn("password").Select()
		panicIf(err)

		log.Println("GOOD TOKEN:", token, "IS", user.Uuid, user.Login)

		conn.WriteJSON(Packet{
			Type: PACKET_TYPE_AUTH,
			Data: user.Uuid,
		})

		client := &Client{
			Conn: conn,
			Hub:  s.Hub,
			User: user,
		}
		s.Hub.Register <- client

		s.Hub.Broadcast <- Packet{
			Type: PACKET_TYPE_ONLINE_USERS,
			Data: []string{user.Uuid},
		}

		client.Goroutine()
	})

	if err != nil {
		if _, ok := err.(websocket.HandshakeError); ok {
			log.Println(err)
		}
		return
	}
}
