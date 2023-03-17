package api

import (
	"peer2http/middleware"

	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {

	Router := gin.Default()

	// 中间件
	// r.Use(middleware.Session(os.Getenv("SESSION_SECRET")))
	// app.Router.Use(middleware.Cors())
	// app.Router.Use(middleware.CurrentUser())
	v1 := Router.Group("/api/v1")
	{
		v1.POST("ping", Ping)

		v1.POST("user/register", UserRegister)

		v1.POST("user/login", UserLogin)
		authed := v1.Group("/")
		authed.Use(middleware.JwtVerify())
		authed.GET("me", UserMe)
		authed.POST("magnet", AddMagnet)
		// authed.POST("magnet", func(ctx *gin.Context) {

		// 	// app.AddMagnet(magnet)
		// 	// files := app.GetFiles(util.GetHash(magnet))
		// 	// api.AddMagnet(ctx)
		// })
		authed.GET("magnets", ListMagnets)
	}
	return Router
}
