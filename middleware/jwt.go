package middleware

import (
	"fmt"
	"log"
	"net/http"
	"peer2http/serializer"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

const (
	signingKey = "haojiahuo"
)

func JwtVerify() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing Authorization header"})
		}

		log.Println(authHeader, "这是authHeader")

		tokenString := authHeader[7:]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				// 这里有返回是因为这是在一个匿名函数里
				return nil, jwt.ErrSignatureInvalid
			}
			fmt.Println("这实在验证token 是有效的且合法的")
			return []byte(signingKey), nil
		})

		if err != nil {
			fmt.Println(err, "验证token失败，然后应该直接就返回了，不应该再继续请求其他方法了的", authHeader, tokenString)
			c.AbortWithStatusJSON(http.StatusUnauthorized, serializer.Response{
				Status: 40001,
				Msg:    "invalid token",
			})
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			fmt.Println(claims, "jwt 的 payload 信息")
			userid := claims["id"]
			fmt.Printf("%t userid 类型", userid)
			c.Set("user", claims) // 将jwt的payload信息存储到gin的上下文中
			c.Set("userid", userid)
			c.Next()
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, serializer.Response{
				Status: 40001,
				Msg:    "invalid token",
			})
		}
	}
}
