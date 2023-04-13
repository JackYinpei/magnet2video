package middleware

import (
	"peer2http/app"

	"github.com/gin-gonic/gin"
)

func PlayLimiter() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		limiter := app.AppObj.PlayLimiter
		if limiter.Allow() {
			ctx.Next()
		} else {
			ctx.AbortWithStatusJSON(429, gin.H{"msg": "too many request"})
		}
	}
}
