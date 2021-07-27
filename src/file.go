package main

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/go-pg/pg/v10"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

type File struct {
	Uuid     string
	UserUuid string
	Name     string
	Type     string
	Size     int64
	Data     []byte
}

func (s *Server) HttpPostFile(ctx *fasthttp.RequestCtx) {
	token := string(ctx.Request.Header.Peek("token"))

	userUuid, err := s.GetUserUuidByToken(token)
	if err != nil {
		if err == pg.ErrNoRows {
			ctx.Error("", fasthttp.StatusUnauthorized)
		} else {
			HttpInternalServerError(ctx, err)
		}
		return
	}

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		ctx.Error("", fasthttp.StatusBadRequest)
		return
	}

	fileType := fileHeader.Header.Get("Content-Type")

	fileM, err := fileHeader.Open()
	if err != nil {
		HttpInternalServerError(ctx, err)
		return
	}

	var buf bytes.Buffer
	io.Copy(&buf, fileM)

	fileM.Close()

	file := &File{
		Uuid:     uuid.New().String(),
		UserUuid: userUuid,
		Name:     fileHeader.Filename,
		Type:     fileType,
		Size:     fileHeader.Size,
		Data:     buf.Bytes(),
	}

	_, err = s.Db.Model(file).Insert()
	if err != nil {
		HttpInternalServerError(ctx, err)
		return
	}

	ctx.WriteString(file.Uuid)
}

func (s *Server) HttpGetFile(ctx *fasthttp.RequestCtx) {
	fileUuid := ctx.UserValue("uuid")
	if fileUuid == nil {
		ctx.Error("", fasthttp.StatusBadRequest)
		return
	}

	file := &File{
		Uuid: fileUuid.(string),
	}
	err := s.Db.Model(file).WherePK().Select()
	if err != nil {
		if err == pg.ErrNoRows {
			ctx.Error("", fasthttp.StatusNotFound)
		} else {
			HttpInternalServerError(ctx, err)
		}
		return
	}

	ctx.Success(file.Type, file.Data)
}

func (s *Server) HttpGetFileInfos(ctx *fasthttp.RequestCtx) {
	fileUuid := ctx.UserValue("uuid")
	if fileUuid == nil {
		ctx.Error("", fasthttp.StatusBadRequest)
		return
	}

	file := &File{
		Uuid: fileUuid.(string),
	}
	err := s.Db.Model(file).WherePK().Column("name", "type", "size").Select()
	if err != nil {
		if err == pg.ErrNoRows {
			ctx.Error("", fasthttp.StatusNotFound)
		} else {
			HttpInternalServerError(ctx, err)
		}
		return
	}

	json, err := json.Marshal(map[string]interface{}{
		"name": file.Name, "type": file.Type, "size": file.Size,
	})
	if err != nil {
		HttpInternalServerError(ctx, err)
		return
	}

	ctx.Write(json)
}

func (s *Server) HttpDownloadFile(ctx *fasthttp.RequestCtx) {
	fileUuid := ctx.UserValue("uuid")
	if fileUuid == nil {
		ctx.Error("", fasthttp.StatusBadRequest)
		return
	}

	file := &File{
		Uuid: fileUuid.(string),
	}
	err := s.Db.Model(file).WherePK().Select()
	if err != nil {
		if err == pg.ErrNoRows {
			ctx.Error("", fasthttp.StatusNotFound)
		} else {
			HttpInternalServerError(ctx, err)
		}
		return
	}

	ctx.Response.Header.Add("Content-Disposition", "attachment; filename=\""+file.Name+"\"")

	ctx.Success(file.Type, file.Data)
}
