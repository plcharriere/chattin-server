package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"

	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

type Avatar struct {
	Uuid     string
	UserUuid string
	Type     string
	Data     []byte
}

func (s *Server) HttpGetAvatars(ctx *fasthttp.RequestCtx) {
	token := string(ctx.Request.Header.Peek("token"))

	userUuid, err := s.GetUserUuidByToken(token)
	if err != nil {
		log.Print(err)
		return
	}

	var avatars []Avatar
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

		userAvatar := &Avatar{
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
		avatarUuid := string(ctx.FormValue("uuid"))
		user.AvatarUuid = avatarUuid
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

	userAvatar := &Avatar{
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

func (s *Server) HttpDeleteAvatar(ctx *fasthttp.RequestCtx) {
	token := string(ctx.Request.Header.Peek("token"))

	user, err := s.GetUserByToken(token)
	if err != nil {
		ctx.WriteString("-1")
		return
	}

	uuid := ctx.UserValue("uuid")
	if uuid == nil {
		return
	}

	userAvatar := &Avatar{
		Uuid: uuid.(string),
	}
	_, err = s.Db.Model(userAvatar).WherePK().Where("user_uuid = ?", user.Uuid).Delete()
	if err != nil {
		log.Print(err)
		return
	}

	if user.AvatarUuid == userAvatar.Uuid {
		user.AvatarUuid = ""
		_, err = s.Db.Model(user).WherePK().Column("avatar_uuid").Update()
		if err != nil {
			log.Print(err)
		}

		go func() {
			s.Hub.Broadcast <- Packet{
				Type: PACKET_TYPE_UPDATE_USERS,
				Data: []User{*user},
			}
		}()
	}

	ctx.WriteString("1")
}
