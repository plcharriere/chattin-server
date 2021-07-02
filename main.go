package main

import (
	"log"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

func main() {
	log.Println("Welcome to IM Server")

	server := &Server{}

	log.Println("Connecting to postgresql...")
	server.Db = pg.Connect(&pg.Options{
		User:     "root",
		Password: "kYkPg7TtSFeDqwXU",
		Database: "im-server",
	})
	defer server.Db.Close()
	var n int
	_, err := server.Db.QueryOne(pg.Scan(&n), "SELECT 1")
	panicIf(err)

	log.Println("Postgresql connection successful")

	log.Print("Loading channels...")
	err = server.Db.Model(&server.Channels).Select()
	panicIf(err)

	log.Printf("Loaded %d channel(s)", len(server.Channels))

	err = createSchema(server.Db)
	if err == nil {
		log.Println("Created postgres schema")
	}

	server.Hub = NewHub(server)
	go server.Hub.Goroutine()

	fasthttp.ListenAndServe(":2727", server.HandleFastHTTP)
}

func createSchema(db *pg.DB) error {
	models := []interface{}{
		(*User)(nil),
		(*UserToken)(nil),
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
