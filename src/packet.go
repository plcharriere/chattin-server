package main

import "encoding/json"

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
	PACKET_TYPE_ADD_USERS     PacketType = 4
	PACKET_TYPE_REMOVE_USERS  PacketType = 5
	PACKET_TYPE_UPDATE_USERS  PacketType = 6
	PACKET_TYPE_MESSAGE       PacketType = 7
)

func ParsePacketJson(packetJson []byte) (*Packet, error) {
	packet := &Packet{}
	err := json.Unmarshal(packetJson, packet)
	if err != nil {
		return nil, err
	}
	return packet, nil
}
