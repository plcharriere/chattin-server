package main

import (
	"encoding/json"
	"strconv"

	"github.com/go-pg/pg/v10"
	"github.com/valyala/fasthttp"
)

type Channel struct {
	Uuid         string `json:"uuid"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Nsfw         bool   `json:"nsfw"`
	SaveMessages bool   `json:"saveMessages"`
}

func (s *Server) HttpGetChannels(ctx *fasthttp.RequestCtx) {
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

	var channels []Channel
	err = s.Db.Model(&channels).Select()
	if err != nil {
		HttpInternalServerError(ctx, err)
		return
	}

	json, err := json.Marshal(channels)
	if err != nil {
		HttpInternalServerError(ctx, err)
		return
	}

	ctx.Write(json)
}

func (s *Server) HttpGetChannelMessages(ctx *fasthttp.RequestCtx) {
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

	channelUuid := ctx.UserValue("uuid")
	if channelUuid == nil {
		ctx.Error("", fasthttp.StatusBadRequest)
		return
	}

	fromMessageUuid := string(ctx.FormValue("from"))

	count := string(ctx.FormValue("count"))
	if len(count) == 0 {
		ctx.Error("", fasthttp.StatusBadRequest)
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
			HttpInternalServerError(ctx, err)
			return
		}

		query.Where("uuid != ? AND date <= ?", fromMessage.Uuid, fromMessage.Date)
	}

	countInt, err := strconv.Atoi(count)
	if err != nil {
		HttpInternalServerError(ctx, err)
		return
	}

	err = query.Order("date DESC").Limit(countInt).Select()
	if err != nil {
		HttpInternalServerError(ctx, err)
		return
	}

	json, err := json.Marshal(messages)
	if err != nil {
		HttpInternalServerError(ctx, err)
		return
	}

	ctx.Write(json)
}
