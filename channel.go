package main

type Channel struct {
	Uuid         string `json:"uuid"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Nsfw         bool   `json:"nsfw"`
	SaveMessages bool   `json:"save_messages"`
}
