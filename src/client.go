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
	case PACKET_TYPE_USER_LIST:
		var users []User
		err = client.Hub.Server.Db.Model(&users).Select()
		if err != nil {
			return err
		}
		client.SendPacket(Packet{
			Type: packet.Type,
			Data: users,
		})
	case PACKET_TYPE_ONLINE_USERS:
		client.Hub.Message <- ClientMessage{
			client,
			message,
		}
	case PACKET_TYPE_CHANNEL_LIST:
		var channels []Channel
		err = client.Hub.Server.Db.Model(&channels).Select()
		if err != nil {
			return err
		}
		client.SendPacket(Packet{
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
	case PACKET_TYPE_GET_MESSAGES:
		recvMsg := packet.Data.(map[string]interface{})
		var messages []Message
		query := client.Hub.Server.Db.Model(&messages).Where("channel_uuid = ?", recvMsg["channelUuid"])
		if recvMsg["fromMessageUuid"].(string) != "" {
			fromMessage := Message{
				Uuid: recvMsg["fromMessageUuid"].(string),
			}
			err := client.Hub.Server.Db.Model(&fromMessage).WherePK().Select()
			if err != nil {
				return err
			}
			query.Where("uuid != ? AND date <= ?", fromMessage.Uuid, fromMessage.Date)
		}
		err := query.Order("date DESC").Limit(int(recvMsg["count"].(float64))).Select()
		if err != nil {
			return err
		}
		client.SendPacket(Packet{
			Type: PACKET_TYPE_GET_MESSAGES,
			Data: messages,
		})
	default:
		log.Println("UNKNOWN PACKET TYPE:", packet.Type)
	}

	return nil
}
