package api

import (
	"peer2http/serializer"
	"peer2http/service"

	"github.com/gin-gonic/gin"
)

func AddMagnet(c *gin.Context) {
	service := service.MagnetService{}
	userid, ok := c.Get("userid")
	if !ok {
		c.JSON(500, serializer.Response{
			Status: 50001,
			Msg:    "中间件里明明有放userid",
		})
	}
	if err := c.ShouldBind(&service); err == nil {
		res := service.Create(uint(userid.(float64)))
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrResponse(err))
	}
}

func ListMagnets(c *gin.Context) {
	listService := service.ListMagnetsService{}
	userid, ok := c.Get("userid")
	if !ok {
		c.JSON(500, serializer.Response{
			Status: 50001,
			Msg:    "中间件里明明有放userid",
		})
	}
	if err := c.ShouldBind(&listService); err == nil {
		res := listService.Create(uint(userid.(float64)))
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrResponse(err))
	}
}
