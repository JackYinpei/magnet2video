package api

import (
	"peer2http/serializer"
	"peer2http/service"

	"github.com/gin-contrib/sessions"
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

func UserLogin(c *gin.Context) {
	var service service.UserLoginService
	if err := c.ShouldBind(&service); err != nil {
		if user, err := service.Login(); err != nil {
			c.JSON(200, err)
		} else {
			s := sessions.Default(c)
			s.Clear()
			s.Set("userid", user.ID)
			s.Save()
			res := serializer.BuildUserResponse(user)
			c.JSON(200, res)
		}
	} else {
		c.JSON(200, ErrResponse(err))
	}
}

func UserMe(c *gin.Context) {
	user := CurrentUser(c)
	res := serializer.BuildUserResponse(*user)
	c.JSON(200, res)
}

func UserLogout(c *gin.Context) {
	s := sessions.Default(c)
	s.Clear()
	s.Save()
	c.JSON(200, serializer.Response{
		Status: 0,
		Msg:    "登出成功",
	})
}
