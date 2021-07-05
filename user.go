package main

type User struct {
	Uuid       string `json:"uuid"`
	Login      string `json:"login"`
	Password   string `json:"-"`
	Nickname   string `json:"nickname"`
	AvatarUuid string `json:"avatarUuid"`
	Bio        string `json:"bio"`
}

type UserToken struct {
	Token string `pg:",pk"`
	Uuid  string `pg:",nopk"`
}

type UserAvatar struct {
	Uuid     string
	UserUuid string
	Type     string
	Data     []byte
}

type UserFile struct {
	Uuid     string
	UserUuid string
	Type     string
	Data     []byte
}
