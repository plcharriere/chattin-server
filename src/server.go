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
