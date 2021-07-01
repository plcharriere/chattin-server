package main

import (
	"log"

	"github.com/fasthttp/websocket"
)

type Client struct {
	Conn *websocket.Conn
	Hub  *Hub
	User *User
}

func (client *Client) Goroutine() {
	defer func() {
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
		client.Hub.Message <- ClientMessage{
			client,
			message,
		}
	}
}
