package api

import (
	"peer2http/app"
	"peer2http/util"

	"github.com/gin-gonic/gin"
)

func GetVideo(c *gin.Context) {
	magnetString := c.Param("magnet") + "?xt=" + c.Query("xt")
	fileName := c.Query("file")
	// userid, _ := c.Get("userid")
	app := app.AppObj
	app.ContentServer(c.Writer, c.Request, util.GetHash(magnetString), fileName)

}
