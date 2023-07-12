package api

import (
	"encoding/json"
	"fmt"
	"peer2http/db"
	"peer2http/serializer"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

func Ping(c *gin.Context) {
	c.JSON(200, serializer.Response{
		Status: 0,
		Msg:    "Pong",
	})
}

func ErrResponse(err error) serializer.Response {
	// if ve, ok := err.(validator.ValidationErrors); ok {
	// 	for _, e := range ve {
	// 		// TODO conf 包没有写
	// 		field := conf.T(fmt.Sprintf("Field.%s".e.Field))
	// 		tag := conf.T(fmt.Sprintf("Tag.Valid.%s".e.Tag))
	// 		return serializer.Response{
	// 			Status: 40001,
	// 			Msg:    fmt.Sprintf("%s %s", &field, &tag),
	// 		}
	// 	}
	// }
	if _, ok := err.(*json.UnmarshalFieldError); ok {
		return serializer.Response{
			Status: 40001,
			Msg:    "Json 类型不匹配",
			Error:  err.Error(),
		}
	}
	return serializer.Response{
		Status: 40001,
		Msg:    "参数错误",
		Error:  err.Error(),
	}
}

func CurrentUser(c *gin.Context) *db.User {
	if user, exists := c.Get("user"); exists {
		fmt.Println("上下文中存在这个 user的map claim", user)
		// TODO DB中查询当前user
		if userid, ok := user.(jwt.MapClaims)["id"]; ok {
			fmt.Println("这是jwt类型断言之后", userid)
			userinDB, err := db.GetUser(userid)
			if err != nil {
				return nil
			}
			return &userinDB
		}
		fmt.Println("类型断言失败")
	}
	return nil
}
