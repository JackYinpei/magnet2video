package service

import (
	"peer2http/app"
	"peer2http/util"

	"github.com/gin-gonic/gin"
)

type MagnetSpecFileService struct {
	MagnetName string `json:"magnet" form:"magnet"`
	FileName   string `json:"file" form:"file"`
}

func (m *MagnetSpecFileService) GetFile(c *gin.Context) {
	app := app.AppObj
	app.ContentServer(c.Writer, c.Request, util.GetHash(m.MagnetName), m.FileName)
}
