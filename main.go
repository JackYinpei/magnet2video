package main

import (
	app2 "peer2http/app"
	"peer2http/db"
)

// magnet:?xt=urn:btih:DD5B2337F90EE4D34012F0C270825B9EFF6A7960
// magnet:?xt=urn:btih:08ada5a7a6183aae1e09d831df6748d566095a10
func main() {
	// download file by given torrent file
	app, _ := app2.New("../torrents")
	// 链接数据库
	db.Databases()
	app.NewRouter()
	// app.Router.GET("/magnet", func(context *gin.Context) {
	// 	magnet := context.Query("name")
	// 	fmt.Println(magnet)
	// 	app.AddMagnet(magnet)
	// 	files := app.GetFiles(util.GetHash(magnet))
	// 	context.String(http.StatusOK, "现在里面有这些文件%v", files)
	// })
	// app.Router.GET("/video", func(context *gin.Context) {
	// 	magnet := context.Query("name")
	// 	fmt.Println(magnet)
	// 	fileName := context.Query("file")
	// 	app.ContentServer(context.Writer, context.Request, util.GetHash(magnet), fileName)
	// })
	app.Router.Run(":8080")
	// 关闭 TorrentClient
	app.Close()
}
