package api

import (
	"fmt"
	"log"
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
		// here add magnet obj to app map
		err, files := app.AppObj.AddMagnet(service.Magnet)
		if err != nil {
			c.JSON(500, serializer.Response{
				Status: 50000,
				Msg:    "添加magnet失败",
				Error:  err.Error(),
			})
		} else {
			fmt.Println("下一步 添加这个magnet 到数据库")
			res := service.Create(uint(userid.(float64)))
			fileStruct := gin.H{
				"files": files,
			}
			res.Data = fileStruct
			log.Printf("%v files, Type %T, 第一个%T, \nres %v", files, files, files[0], res)
			c.JSON(200, res)
		}
	} else {
		c.JSON(200, ErrResponse(err))
	}
}

func DeleteMagnet(c *gin.Context) {
	magnetString := c.Param("magnet") + "?xt=" + c.Query("xt")
	userid, _ := c.Get("userid")
	service := service.MagnetServiceDelete{
		Magnet: magnetString,
		UserId: uint(userid.(float64)),
	}
	res := service.Delete()
	c.JSON(200, res)
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
		res := listService.ListMagnets(uint(userid.(float64)))
		c.JSON(200, res)
	} else {
		c.JSON(200, ErrResponse(err))
	}
}

func GetMagnetFile(c *gin.Context) {
	magnetString := c.Param("magnet") + "?xt=" + c.Query("xt")
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
	res := service.GetMagnetFiles()
	c.JSON(200, res)
}
