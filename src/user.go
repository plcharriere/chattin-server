package main

type User struct {
	Uuid        string `json:"uuid"`
	Login       string `json:"login"`
	Password    string `json:"-"`
	Online      bool   `json:"online"`
	ChannelUuid string `json:"-"`
	Nickname    string `json:"nickname"`
	AvatarUuid  string `json:"avatarUuid"`
	Bio         string `json:"bio"`
}
