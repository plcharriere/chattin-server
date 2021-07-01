package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
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
			hub.SendUserState(client.User, true)
		case client := <-hub.Unregister:
			if _, ok := hub.Clients[client]; ok {
				delete(hub.Clients, client)
			}
			hub.SendUserState(client.User, false)
		case message := <-hub.Message:
			hub.ParseClientMessage(message.message, message.client)
		case packet := <-hub.Broadcast:
			for c := range hub.Clients {
				c.Conn.WriteJSON(packet)
			}
		}
	}
}

func (hub *Hub) SendUserState(user *User, online bool) {
	uuids := []string{user.Uuid}
	packetType := PACKET_TYPE_ONLINE_USERS
	if !online {
		packetType = PACKET_TYPE_OFFLINE_USERS
	}
	for c := range hub.Clients {
		c.Conn.WriteJSON(Packet{
			Type: packetType,
			Data: uuids,
		})
	}
}

func (hub *Hub) ParseClientMessage(message []byte, client *Client) error {
	var packet Packet
	err := json.Unmarshal(message, &packet)
	if err != nil {
		return err
	}

	switch packet.Type {
	case PACKET_TYPE_USER_LIST:
		var users []User
		err = hub.Server.Db.Model(&users).Select()
		panicIf(err)
		client.Conn.WriteJSON(Packet{
			Type: packet.Type,
			Data: users,
		})
	case PACKET_TYPE_ONLINE_USERS:
		onlineUsers := []string{}
		for c := range hub.Clients {
			onlineUsers = append(onlineUsers, c.User.Uuid)
		}
		client.Conn.WriteJSON(Packet{
			Type: packet.Type,
			Data: onlineUsers,
		})
	case PACKET_TYPE_CHANNEL_LIST:
		var channels []Channel
		err = hub.Server.Db.Model(&channels).Select()
		panicIf(err)
		client.Conn.WriteJSON(Packet{
			Type: packet.Type,
			Data: channels,
		})
	case PACKET_TYPE_MESSAGE:
		recvMsg := packet.Data.(map[string]interface{})
		msg := &Message{
			uuid.New().String(),
			recvMsg["channelUuid"].(string),
			client.User.Uuid,
			time.Now(),
			0,
			recvMsg["content"].(string),
		}

		_, err := hub.Server.Db.Model(msg).Insert()
		panicIf(err)

		res := Packet{
			Type: packet.Type,
			Data: msg,
		}
		for c := range hub.Clients {
			c.Conn.WriteJSON(res)
		}
	default:
		log.Println("UNKNOWN PACKET TYPE:", packet.Type)
	}

	return nil
}
