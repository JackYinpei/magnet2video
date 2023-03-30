package api

import (
	"peer2http/serializer"
	"peer2http/service"

	"github.com/gin-gonic/gin"
)

func HotMagnets(c *gin.Context) {
	// var service service.MagnetHotService
	// service.Get()
	c.JSON(404, nil)
}

func MyHotMangets(c *gin.Context) {
	myHot := service.MyHotMangetsService{
		Limit:        20,
		Start:        0,
		MagnetString: make([]string, 0),
	}
	userid, ok := c.Get("userid")
	if !ok {
		c.JSON(500, serializer.Response{
			Status: 50001,
			Msg:    "中间件里明明有放userid",
		})
	}
	c.JSON(200, myHot.GetMyLove(uint(userid.(float64))))
}
