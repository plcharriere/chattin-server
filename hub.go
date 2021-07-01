package main

import (
	"encoding/json"
	"log"
)

type Hub struct {
	Server     *Server
	Clients    map[*Client]bool
	Register   chan *Client
	Unregister chan *Client
	Message    chan ClientMessage
	Broadcast  chan Packet
}

type ClientMessage struct {
	client  *Client
	message []byte
}

func NewHub(server *Server) *Hub {
	return &Hub{
		Server:     server,
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Clients:    make(map[*Client]bool),
		Message:    make(chan ClientMessage),
		Broadcast:  make(chan Packet),
	}
}

func (hub *Hub) Goroutine() {
	for {
		select {
		case client := <-hub.Register:
			hub.Clients[client] = true
		case client := <-hub.Unregister:
			if _, ok := hub.Clients[client]; ok {
				delete(hub.Clients, client)
			}
		case message := <-hub.Message:
			hub.ParseClientMessage(message.message, message.client)
		case packet := <-hub.Broadcast:
			for c := range hub.Clients {
				c.SendPacket(packet)
			}
		}
	}
}

func (hub *Hub) ParseClientMessage(message []byte, client *Client) error {
	var packet Packet
	err := json.Unmarshal(message, &packet)
	if err != nil {
		return err
	}

	switch packet.Type {
	case PACKET_TYPE_ONLINE_USERS:
		onlineUsers := []string{}
		for c := range hub.Clients {
			onlineUsers = append(onlineUsers, c.User.Uuid)
		}
		client.SendPacket(Packet{
			Type: packet.Type,
			Data: onlineUsers,
		})
	default:
		log.Println("UNKNOWN PACKET TYPE:", packet.Type)
	}

	return nil
}
