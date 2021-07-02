package main

import "github.com/go-pg/pg/v10"

type Server struct {
	Db       *pg.DB
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
