package main

import (
	"encoding/json"

	"github.com/fasthttp/router"
	"github.com/go-pg/pg/v10"
	"github.com/valyala/fasthttp"
)

type Server struct {
	Db            *pg.DB
	Router        *router.Router
	Hub           *Hub
	Channels      []*Channel
	Configuration Configuration
}

type Configuration struct {
	tableName   struct{} `pg:"configuration"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
}

func (server *Server) GetChannelByUuid(uuid string) *Channel {
	for _, channel := range server.Channels {
		if channel.Uuid == uuid {
			return channel
		}
	}
	return nil
}

func (s *Server) HttpGetConfiguration(ctx *fasthttp.RequestCtx) {
	json, err := json.Marshal(s.Configuration)
	if err != nil {
		HttpInternalServerError(ctx, err)
		return
	}

	ctx.Write(json)
}
