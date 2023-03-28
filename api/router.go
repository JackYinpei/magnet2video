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
		// use jwt middleware
		authed.Use(middleware.JwtVerify())
		// who am i
		authed.GET("me", UserMe)
		// add magnet to this user
		authed.POST("magnet", AddMagnet)
		// TODO 可能要加一个中间件 来判断user 是不是own this magnet
		// list all magnets which this user owns
		authed.GET("magnets", ListMagnets)
		authed.DELETE("magnet/:magnet", DeleteMagnet)
		// get magnet files 前面的路由已经存在了，怪不得这里进不去
		// 这里使用的时候不需要加？magnet=*** 直接加magnet string
		// 操！！！ 我传进来的magnet string 被裁掉了
		authed.GET("mf/:magnet", GetMagnetFile)
		authed.GET("video", GetVideo)
		// authed.POST("magnet", func(ctx *gin.Context) {

		// 	// app.AddMagnet(magnet)
		// 	// files := app.GetFiles(util.GetHash(magnet))
		// 	// api.AddMagnet(ctx)
		// })
	}
	return Router
}
