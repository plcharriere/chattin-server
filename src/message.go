package main

import "time"

type Message struct {
	Uuid        string    `json:"uuid"`
	ChannelUuid string    `json:"channelUuid"`
	UserUuid    string    `json:"userUuid"`
	Date        time.Time `json:"date"`
	Edited      time.Time `json:"edited"`
	Content     string    `json:"content"`
}
