package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/google/uuid"
)

type Client struct {
	Conn    *websocket.Conn
	SendMux sync.Mutex
	Hub     *Hub
	User    *User
}

func (client *Client) Goroutine() {
	defer func() {
		client.Hub.Broadcast <- Packet{
			Type: PACKET_TYPE_OFFLINE_USERS,
			Data: []string{client.User.Uuid},
		}
		client.Hub.Unregister <- client
		client.Conn.Close()

		client.User.Online = false
		client.Hub.Server.Db.Model(client.User).WherePK().Column("online").Update()
	}()
	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		err = client.ParseMessage(message)
		if err != nil {
			log.Println("client.ParseMessage:", err)
		}
	}
}

func (client *Client) SendPacket(packet Packet) {
	client.SendMux.Lock()
	defer client.SendMux.Unlock()
	client.Conn.WriteJSON(packet)
}

func (client *Client) ParseMessage(message []byte) error {
	var packet Packet
	err := json.Unmarshal(message, &packet)
	if err != nil {
		return err
	}

	switch packet.Type {
	case PACKET_TYPE_ONLINE_USERS:
		client.Hub.Message <- ClientMessage{
			client,
			message,
		}
	case PACKET_TYPE_MESSAGE:
		recvMsg := packet.Data.(map[string]interface{})
		msg := &Message{
			uuid.New().String(),
			recvMsg["channelUuid"].(string),
			client.User.Uuid,
			time.Now(),
			time.Time{},
			recvMsg["content"].(string),
		}

		channel := client.Hub.Server.GetChannelByUuid(recvMsg["channelUuid"].(string))
		if channel != nil {
			if channel.SaveMessages {
				_, err := client.Hub.Server.Db.Model(msg).Insert()
				if err != nil {
					return err
				}
			}

			client.Hub.Broadcast <- Packet{
				Type: packet.Type,
				Data: msg,
			}
		}
	case PACKET_TYPE_SET_CHANNEL_UUID:
		channelUuid := packet.Data.(string)
		client.User.ChannelUuid = channelUuid
		client.Hub.Server.Db.Model(client.User).WherePK().Column("channel_uuid").Update()
	case PACKET_TYPE_TYPING:
		channelUuid := packet.Data.(string)
		client.Hub.Broadcast <- Packet{
			Type: packet.Type,
			Data: []string{channelUuid, client.User.Uuid},
		}
	case PACKET_TYPE_DELETE_MESSAGE:
		messageUuid := packet.Data.(string)

		var message Message
		r, err := client.Hub.Server.Db.Model(&message).Where("uuid = ?", messageUuid).Where("user_uuid = ?", client.User.Uuid).Delete()
		if err != nil {
			return err
		}

		if r.RowsAffected() > 0 {
			client.Hub.Broadcast <- Packet{
				Type: packet.Type,
				Data: messageUuid,
			}
		}
	case PACKET_TYPE_EDIT_MESSAGE:
		recvMsg := packet.Data.(map[string]interface{})
		messageUuid := recvMsg["messageUuid"].(string)
		content := recvMsg["content"].(string)

		message := &Message{
			Content: content,
			Edited:  time.Now(),
		}
		r, err := client.Hub.Server.Db.Model(message).Column("content", "edited").Where("uuid = ?", messageUuid).Where("user_uuid = ?", client.User.Uuid).Update()
		if err != nil {
			return err
		}

		if r.RowsAffected() > 0 {
			client.Hub.Broadcast <- Packet{
				Type: packet.Type,
				Data: PacketEditMessage{
					messageUuid,
					content,
					message.Edited,
				},
			}
		}
	default:
		log.Println("UNKNOWN PACKET TYPE:", packet.Type)
	}

	if packet.Type != PACKET_TYPE_TYPING {
		log.Println(client.Conn.RemoteAddr(), "WS", packet.Type)
	}

	return nil
}
