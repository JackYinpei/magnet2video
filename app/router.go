package app

import (
	"peer2http/api"

	"github.com/gin-gonic/gin"
)

func (app *App) NewRouter() {
	app.Router = gin.Default()

	// 中间件
	// r.Use(middleware.Session(os.Getenv("SESSION_SECRET")))
	// app.Router.Use(middleware.Cors())
	// app.Router.Use(middleware.CurrentUser())
	v1 := app.Router.Group("/api/v1")
	{
		v1.POST("ping", api.Ping)

		v1.POST("user/register", api.UserRegister)
	}

}
