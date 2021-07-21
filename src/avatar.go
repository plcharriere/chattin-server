package main

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/go-pg/pg/v10"
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
		if err == pg.ErrNoRows {
			ctx.Error("", fasthttp.StatusUnauthorized)
		} else {
			HttpInternalServerError(ctx, err)
		}
	}

	var avatars []Avatar
	err = s.Db.Model(&avatars).Where("user_uuid = ?", userUuid).Select()
	if err != nil {
		HttpInternalServerError(ctx, err)
		return
	}

	if len(avatars) == 0 {
		ctx.Error("", fasthttp.StatusNoContent)
		return
	}

	var avatarUuids []string
	for _, avatar := range avatars {
		avatarUuids = append(avatarUuids, avatar.Uuid)
	}

	json, err := json.Marshal(avatarUuids)
	if err != nil {
		HttpInternalServerError(ctx, err)
		return
	}

	ctx.Write(json)
}

func (s *Server) HttpPostAvatar(ctx *fasthttp.RequestCtx) {
	token := string(ctx.FormValue("token"))

	user, err := s.GetUserByToken(token)
	if err != nil {
		if err == pg.ErrNoRows {
			ctx.Error("", fasthttp.StatusUnauthorized)
		} else {
			HttpInternalServerError(ctx, err)
		}
		return
	}

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		avatarUuid := string(ctx.FormValue("uuid"))
		user.AvatarUuid = avatarUuid
	} else {
		imageTypes := map[string]bool{
			"image/png":  true,
			"image/jpeg": true,
			"image/gif":  true,
			"image/webp": true,
		}

		fileType := fileHeader.Header.Get("Content-Type")
		if _, ok := imageTypes[fileType]; !ok {
			ctx.Error("", fasthttp.StatusNotAcceptable)
			return
		}

		file, err := fileHeader.Open()
		if err != nil {
			HttpInternalServerError(ctx, err)
			return
		}

		var buf bytes.Buffer
		io.Copy(&buf, file)

		file.Close()

		avatar := &Avatar{
			Uuid:     uuid.New().String(),
			UserUuid: user.Uuid,
			Type:     fileType,
			Data:     buf.Bytes(),
		}

		_, err = s.Db.Model(avatar).Insert()
		if err != nil {
			HttpInternalServerError(ctx, err)
			return
		}

		user.AvatarUuid = avatar.Uuid
	}

	_, err = s.Db.Model(user).WherePK().Column("avatar_uuid").Update()
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

func (s *Server) HttpGetAvatar(ctx *fasthttp.RequestCtx) {
	avatarUuid := ctx.UserValue("uuid")
	if avatarUuid == nil {
		ctx.Error("", fasthttp.StatusBadRequest)
		return
	}

	avatar := &Avatar{
		Uuid: avatarUuid.(string),
	}
	err := s.Db.Model(avatar).WherePK().Select()
	if err != nil {
		if err == pg.ErrNoRows {
			ctx.Error("", fasthttp.StatusNotFound)
		} else {
			HttpInternalServerError(ctx, err)
		}
		return
	}

	ctx.Success(avatar.Type, avatar.Data)
}

func (s *Server) HttpDeleteAvatar(ctx *fasthttp.RequestCtx) {
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

	avatarUuid := ctx.UserValue("uuid")
	if avatarUuid == nil {
		ctx.Error("", fasthttp.StatusBadRequest)
		return
	}

	avatar := &Avatar{
		Uuid: avatarUuid.(string),
	}
	r, err := s.Db.Model(avatar).WherePK().Where("user_uuid = ?", user.Uuid).Delete()
	if err != nil {
		HttpInternalServerError(ctx, err)
		return
	}
	if r.RowsAffected() == 0 {
		ctx.Error("", fasthttp.StatusNotModified)
		return
	}

	if user.AvatarUuid == avatar.Uuid {
		user.AvatarUuid = ""
		_, err = s.Db.Model(user).WherePK().Column("avatar_uuid").Update()
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
}
