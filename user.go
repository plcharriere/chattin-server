package main

type User struct {
	Uuid     string `json:"uuid"`
	Login    string `json:"login"`
	Password string `json:"-"`
	Nickname string `json:"nickname"`
}

type UserToken struct {
	Token string `pg:",pk"`
	Uuid  string `pg:",nopk"`
}
