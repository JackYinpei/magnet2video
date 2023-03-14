package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	app2 "peer2http/app"
	"peer2http/util"
)

//magnet:?xt=urn:btih:DD5B2337F90EE4D34012F0C270825B9EFF6A7960
func main() {
	// download file by magnet file
	//// new app obj
	//app, err := app2.New("C:\\goproj\\peer2HttpDemo\\torrents")
	//if err != nil {
	//	panic("")
	//}
	//// add magnet to app
	//app.AddMagnet("magnet:?xt=urn:btih:DD5B2337F90EE4D34012F0C270825B9EFF6A7960")
	//// get torrent files inside by hash or magnet
	//files := app.GetFiles("magnet:?xt=urn:btih:DD5B2337F90EE4D34012F0C270825B9EFF6A7960")
	//app.DownloadFile("ubuntu-20.04.5-live-server-amd64.iso", files[0])

	// download file by given torrent file
	app, _ := app2.New("C:\\goproj\\peer2HttpDemo\\torrents")
	app.Router.GET("/magnet", func(context *gin.Context) {
		magnet := context.Query("name")
		fmt.Println(magnet)
		app.AddMagnet(magnet)
		files := app.GetFiles(util.GetHash(magnet))
		context.String(http.StatusOK, "现在里面有这些文件%v", files)
	})
	app.Router.GET("/video", func(context *gin.Context) {
		magnet := context.Query("name")
		fmt.Println(magnet)
		fileName := context.Query("file")
		app.ContentServer(context.Writer, context.Request, util.GetHash(magnet), fileName)
	})
	app.Router.Run(":8080")
	//app.GetTorrent("C:/goproj/peer2HttpDemo/torrents/ubuntu-20.04.5-live-server-amd64.iso.torrent")
	//files := app.GetFiles("ubuntu-20.04.5-live-server-amd64.iso")
	//app.DownloadFile("ubuntu-20.04.5-live-server-amd64.iso", files[0], "C:/goproj/peer2HttpDemo/download/haojiahuo.iso")
	//app.ReadFromHead("ubuntu-20.04.5-live-server-amd64.iso", files[0])

	// 关闭 TorrentClient
	app.Close()
}
