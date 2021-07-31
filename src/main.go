package main

import (
	"log"
	"os"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"gopkg.in/ini.v1"
)

func main() {
	log.Println("Welcome to IM Server")

	log.Print("Parsing configuration...")

	cfg, err := ini.Load("config.ini")
	if err != nil {
		log.Print(err)
	}

	postgresAddress := os.Getenv("POSTGRES_ADDRESS")
	if len(postgresAddress) == 0 && cfg != nil {
		postgresAddress = cfg.Section("postgres").Key("address").String()
	}
	postgresUser := os.Getenv("POSTGRES_USER")
	if len(postgresUser) == 0 && cfg != nil {
		postgresUser = cfg.Section("postgres").Key("user").String()
	}
	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	if len(postgresPassword) == 0 && cfg != nil {
		postgresPassword = cfg.Section("postgres").Key("password").String()
	}
	postgresDatabase := os.Getenv("POSTGRES_DATABASE")
	if len(postgresDatabase) == 0 && cfg != nil {
		postgresDatabase = cfg.Section("postgres").Key("database").String()
	}

	httpAddress := os.Getenv("ADDRESS")
	if len(httpAddress) == 0 && cfg != nil {
		httpAddress = cfg.Section("http").Key("address").String()
	}
	certFilePath := os.Getenv("SSL_CERT")
	if len(certFilePath) == 0 && cfg != nil {
		certFilePath = cfg.Section("ssl").Key("cert").String()
	}
	keyFilePath := os.Getenv("SSL_KEY")
	if len(keyFilePath) == 0 && cfg != nil {
		keyFilePath = cfg.Section("ssl").Key("key").String()
	}

	server := &Server{}

	log.Println("Connecting to postgresql...")
	server.Db = pg.Connect(&pg.Options{
		Addr:     postgresAddress,
		User:     postgresUser,
		Password: postgresPassword,
		Database: postgresDatabase,
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

	log.Print("Loading server configuration...")
	err = server.Db.Model(&server.Configuration).Select()
	panicIf(err)

	log.Print("Loading channels...")
	err = server.Db.Model(&server.Channels).Select()
	panicIf(err)

	log.Printf("Loaded %d channel(s)", len(server.Channels))

	server.SetupFastHTTPRouter()

	server.Hub = NewHub(server)
	go server.Hub.Goroutine()

	fasthttpServer := &fasthttp.Server{
		Handler:            server.HandleFastHTTP,
		Name:               server.Configuration.Name,
		MaxRequestBodySize: 10 * 1024 * 1024 * 1024, // 10 MB
	}

	if len(certFilePath) > 0 && len(keyFilePath) > 0 {
		log.Print("Launching HTTPS server on ", httpAddress)
		err = fasthttpServer.ListenAndServeTLS(httpAddress, certFilePath, keyFilePath)
	} else {
		log.Print("Launching HTTP server on ", httpAddress)
		err = fasthttpServer.ListenAndServe(httpAddress)
	}

	panicIf(err)
}

func createSchema(db *pg.DB) error {
	models := []interface{}{
		(*Configuration)(nil),
		(*User)(nil),
		(*Token)(nil),
		(*Avatar)(nil),
		(*Channel)(nil),
		(*Message)(nil),
		(*File)(nil),
	}

	for _, model := range models {
		err := db.Model(model).CreateTable(&orm.CreateTableOptions{})
		if err != nil {
			return err
		}
	}

	db.Model(&Configuration{
		Name:        "Chattin",
		Description: "",
	}).Insert()

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
