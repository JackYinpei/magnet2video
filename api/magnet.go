package api

import (
	"fmt"
	"peer2http/app"
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
		err := app.AppObj.AddMagnet(service.Magnet)
		if err != nil {
			c.JSON(500, serializer.Response{
				Status: 50000,
				Msg:    "添加magnet失败",
				Error:  err.Error(),
			})
		} else {
			res := service.Create(uint(userid.(float64)))
			c.JSON(200, res)
		}
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

func GetMagnetFile(c *gin.Context) {
	magnetString := c.Param("magnet")
	userid, ok := c.Get("userid")
	fmt.Println(userid, magnetString, "dadadadadaddadadada")
	if !ok {
		c.JSON(500, serializer.Response{
			Status: 50001,
			Msg:    "中间件里明明有放userid",
		})
	}
	service := service.MagnetListService{
		Magnet: magnetString,
		UserID: uint(userid.(float64)),
	}
	fmt.Println(service, "service info in GetMagnetFile function")
	res := service.GetMagnetService()
	c.JSON(200, res)
}
