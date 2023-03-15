package api

import (
	"peer2http/serializer"
	"peer2http/service"

	"github.com/gin-gonic/gin"
)

func UserRegister(c *gin.Context) {
	var service service.UserRegisterService
	if err := c.ShouldBind(&service); err != nil {
		if user, err := service.Register(); err != nil {
			c.JSON(200, err)
		} else {
			res := serializer.BuildUserResponse(user)
			c.JSON(200, res)
		}
	} else {
		c.JSON(200, ErrResponse(err))
	}
}
