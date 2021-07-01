package main

type Packet struct {
	Type PacketType  `json:"type"`
	Data interface{} `json:"data"`
}

type PacketType int

const (
	PACKET_TYPE_AUTH          PacketType = 0
	PACKET_TYPE_USER_LIST     PacketType = 1
	PACKET_TYPE_ONLINE_USERS  PacketType = 2
	PACKET_TYPE_OFFLINE_USERS PacketType = 3
	PACKET_TYPE_CHANNEL_LIST  PacketType = 4
	PACKET_TYPE_MESSAGE       PacketType = 5
)
