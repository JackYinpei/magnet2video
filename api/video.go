package api

import (
	"peer2http/cache"
	"peer2http/serializer"
	"peer2http/service"

	"github.com/gin-gonic/gin"
)

func GetVideo(c *gin.Context) {
	// userid, _ := c.Get("userid")
	// TODO 感觉这里的请求方式不太规范 没有走service层 直接在API层就请求了app
	var mf service.MagnetSpecFileService
	if err := c.ShouldBind(&mf); err != nil {
		c.JSON(200, err)
	} else {
		userid, ok := c.Get("userid")
		if !ok {
			c.JSON(500, serializer.Response{
				Status: 50001,
				Msg:    "中间件里明明有放userid",
			})
		}
		// when user request this magnet obj add this user's specified magnet view one
		cache.UserMagnetViewCountPlusOne(uint(userid.(float64)), mf.MagnetName)
		mf.GetFile(c)
	}

}
