package main

import "time"

type Message struct {
	Uuid        string    `json:"uuid"`
	ChannelUuid string    `json:"channelUuid"`
	UserUuid    string    `json:"userUuid"`
	Date        time.Time `json:"date"`
	Edited      int       `json:"edited"`
	Content     string    `json:"content"`
}

type MessageInput struct {
	ChannelUuid string `json:"channelUuid"`
	Content     string `json:"content"`
}
