package app

// import (
// 	"peer2http/api"
// 	"peer2http/middleware"
// 	"peer2http/util"

// 	"github.com/gin-gonic/gin"
// )

// type GinHandleFuncWrap func(ctx *gin.Context, app *App) func(ctx *gin.Context)

// func (app *App) NewRouter() {
// 	app.Router = gin.Default()

// 	// 中间件
// 	// r.Use(middleware.Session(os.Getenv("SESSION_SECRET")))
// 	// app.Router.Use(middleware.Cors())
// 	// app.Router.Use(middleware.CurrentUser())
// 	v1 := app.Router.Group("/api/v1")
// 	{
// 		v1.POST("ping", api.Ping)

// 		v1.POST("user/register", api.UserRegister)

// 		v1.POST("user/login", api.UserLogin)
// 		authed := v1.Group("/")
// 		authed.Use(middleware.JwtVerify())
// 		authed.GET("me", api.UserMe)
// 		// authed.POST("magnet", api.AddMagnet)
// 		authed.POST("magnet", func(ctx *gin.Context) {

// 			app.AddMagnet(magnet)
// 			files := app.GetFiles(util.GetHash(magnet))
// 			api.AddMagnet(ctx)
// 		})
// 		authed.GET("magnets", api.ListMagnets)
// 	}

// }
