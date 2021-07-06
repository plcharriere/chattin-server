package main

import (
	"log"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"gopkg.in/ini.v1"
)

func main() {
	log.Println("Welcome to IM Server")

	log.Println("Parsing configuration file...")
	cfg, err := ini.Load("config.ini")
	panicIf(err)

	server := &Server{}

	log.Println("Connecting to postgresql...")
	server.Db = pg.Connect(&pg.Options{
		Addr:     cfg.Section("postgres").Key("address").String(),
		User:     cfg.Section("postgres").Key("user").String(),
		Password: cfg.Section("postgres").Key("password").String(),
		Database: cfg.Section("postgres").Key("database").String(),
	})
	defer server.Db.Close()
	var n int
	_, err = server.Db.QueryOne(pg.Scan(&n), "SELECT 1")
	panicIf(err)

	log.Println("Postgresql connection successful")

	err = createSchema(server.Db)
	if err == nil {
		log.Println("Created postgres schema")
	}

	log.Print("Loading channels...")
	err = server.Db.Model(&server.Channels).Select()
	panicIf(err)

	log.Printf("Loaded %d channel(s)", len(server.Channels))

	server.SetupFastHTTPRouter()

	server.Hub = NewHub(server)
	go server.Hub.Goroutine()

	fasthttpServer := &fasthttp.Server{
		Handler:            server.HandleFastHTTP,
		Name:               "Instant Messenger",
		MaxRequestBodySize: 10 * 1024 * 1024 * 1024, // 10 MB
	}

	httpAddress := cfg.Section("http").Key("address").String()
	log.Print("Launching HTTP server on ", httpAddress)

	err = fasthttpServer.ListenAndServe(httpAddress)
	panicIf(err)
}

func createSchema(db *pg.DB) error {
	models := []interface{}{
		(*User)(nil),
		(*UserToken)(nil),
		(*UserAvatar)(nil),
		(*UserFile)(nil),
		(*Channel)(nil),
		(*Message)(nil),
	}

	for _, model := range models {
		err := db.Model(model).CreateTable(&orm.CreateTableOptions{})
		if err != nil {
			return err
		}
	}

	db.Model(&Channel{
		Uuid:         uuid.New().String(),
		Name:         "general",
		Description:  "General channel",
		Nsfw:         false,
		SaveMessages: true,
	}).Insert()
	db.Model(&Channel{
		Uuid:         uuid.New().String(),
		Name:         "dev",
		Description:  "Development channel",
		Nsfw:         false,
		SaveMessages: true,
	}).Insert()
	db.Model(&Channel{
		Uuid:         uuid.New().String(),
		Name:         "tmp",
		Description:  "Messages sent in this channel won't be saved",
		Nsfw:         false,
		SaveMessages: false,
	}).Insert()

	return nil
}
