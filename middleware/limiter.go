package middleware

import (
	"fmt"
	"peer2http/app"

	"github.com/gin-gonic/gin"
)

func PlayLimiter() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if app.AppObj.PlayLimiter.Allow() {
			ctx.Next()
		} else {
			fmt.Println("限流了")
			ctx.AbortWithStatusJSON(429, gin.H{"msg": "too many request"})
		}
	}
}
