package main

import (
	"log"

	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

func (server *Server) SetupFastHTTPRouter() {
	server.Router = router.New()
	server.Router.GET("/ws", server.HttpHandleWebSocket)
	server.Router.GET("/users", server.HttpGetUsers)
	server.Router.POST("/users/login", server.HttpUserLogin)
	server.Router.POST("/users/register", server.HttpUserRegister)
	server.Router.POST("/users/profile", server.HttpUserProfile)
	server.Router.GET("/channels", server.HttpGetChannels)
	server.Router.GET("/channels/{uuid}/messages", server.HttpGetChannelMessages)
	server.Router.POST("/avatars", server.HttpPostAvatar)
	server.Router.GET("/avatars", server.HttpGetAvatars)
	server.Router.GET("/avatars/{uuid}", server.HttpGetAvatar)
	server.Router.DELETE("/avatars/{uuid}", server.HttpDeleteAvatar)
	server.Router.POST("/files", server.HttpPostFile)
	server.Router.GET("/files/{uuid}", server.HttpGetFile)
	server.Router.GET("/files/{uuid}/infos", server.HttpGetFileInfos)
	server.Router.GET("/files/{uuid}/download", server.HttpDownloadFile)
}

func (server *Server) HandleFastHTTP(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
	ctx.Response.Header.Set("Access-Control-Allow-Headers", "*")
	ctx.Response.Header.Set("Access-Control-Allow-Methods", "*")
	server.Router.Handler(ctx)
	log.Println(ctx.RemoteAddr(), "HTTP", ctx.Response.StatusCode(), string(ctx.Method()), string(ctx.Path()))
}

func HttpInternalServerError(ctx *fasthttp.RequestCtx, err error) {
	log.Print(err)
	ctx.Error("", fasthttp.StatusInternalServerError)
}
