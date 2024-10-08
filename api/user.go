package api

import (
	"fmt"
	"net/http"
	"peer2http/serializer"
	"peer2http/service"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const (
	signingKey = "haojiahuo"
)

func UserRegister(c *gin.Context) {
	var service service.UserRegisterService
	if err := c.ShouldBind(&service); err == nil {
		name, _ := c.Get("username")
		password, _ := c.Get("password")
		password_confirm, _ := c.Get("password_confirm")
		fmt.Println(name, password, password_confirm, "好家伙")
		fmt.Println(service, "绑定之后的service 是入参")
		if user, err := service.Register(); err != nil {
			c.JSON(200, err)
		} else {
			res := serializer.BuildUserResponse(user)
			c.JSON(200, res)
		}
	} else {
		fmt.Println(err, "should bind failed")
		// TODO 这里会panic
		c.JSON(200, ErrResponse(err))
	}
}

func UserLogin(c *gin.Context) {
	var service service.UserLoginService
	if err := c.ShouldBind(&service); err == nil {
		if user, err := service.Login(); err != nil {
			c.JSON(200, err)
		} else {
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"id":       user.ID,
				"username": user.Username,
				"exp":      time.Now().Add(time.Hour * 24 * 365).Unix(), // 设置token过期时间为24小时
			})

			tokenString, err := token.SignedString([]byte(signingKey))
			if err != nil {
				fmt.Println(err, "签名失败")
				c.JSON(501, serializer.Response{
					Status: 100001,
					Msg:    "内部错误",
				})
			}
			c.JSON(http.StatusOK, serializer.Response{
				Status: 200000,
				Msg:    "登录ok 返回token",
				Data: gin.H{
					"Authorization": "Bearer:" + tokenString,
					"username":      user.Username,
				},
				Error: "",
			})
			// return token in response header
			// c.Header("Authorization", "Bearer "+tokenString)
			// c.JSON(200, gin.H{
			// 	"haojiahuo": "haojiahuo",
			// })
			// redirect to index page
			// c.Redirect(http.StatusMovedPermanently, "/")
			// c.JSON(200, gin.H{
			// 	"token": tokenString,
			// })
		}
	} else {
		fmt.Println("用户账号密码错误，登录失败")
		c.JSON(200, ErrResponse(err))
	}
}

func UserMe(c *gin.Context) {
	user := CurrentUser(c)
	if user == nil {
		fmt.Println("没有这个user in db")
		c.JSON(200, serializer.Response{
			Status: 400001,
			Msg:    "你这个token 不对吧",
		})
	} else {
		res := serializer.BuildUserResponse(*user)
		fmt.Printf("%v 奇了怪 这应该有信息的呀\n", res)
		c.JSON(200, res)
	}
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
