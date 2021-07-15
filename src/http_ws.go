package main

import (
	"fmt"
	"log"

	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

var upgrader = websocket.FastHTTPUpgrader{
	CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
		return true
	},
}

func (s *Server) HttpHandleWebSocket(ctx *fasthttp.RequestCtx) {
	err := upgrader.Upgrade(ctx, func(conn *websocket.Conn) {
		defer conn.Close()

		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}

		packet, err := ParsePacketJson(msg)
		if err != nil || packet.Type != PACKET_TYPE_AUTH {
			return
		}

		token := fmt.Sprintf("%s", packet.Data)
		user, err := s.GetUserByToken(token)
		if err != nil {
			log.Print(err)
			conn.WriteJSON(Packet{
				Type: PACKET_TYPE_AUTH,
				Data: false,
			})
			return
		}

		user.Online = true
		_, err = s.Db.Model(user).WherePK().Column("online").Update()
		if err != nil {
			log.Print(err)
		}

		client := &Client{
			Conn: conn,
			Hub:  s.Hub,
			User: user,
		}
		s.Hub.Register <- client

		packetAuth := PacketAuth{
			user.Uuid,
			user.ChannelUuid,
		}
		conn.WriteJSON(Packet{
			Type: PACKET_TYPE_AUTH,
			Data: packetAuth,
		})

		log.Println("{"+user.Uuid+"}", user.Login, "logged in")

		client.Goroutine()
	})

	if err != nil {
		if _, ok := err.(websocket.HandshakeError); ok {
			log.Println(err)
		}
		return
	}
}
