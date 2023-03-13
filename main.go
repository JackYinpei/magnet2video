package main

import (
	"fmt"
	app2 "peer2http/app"
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
	app.GetTorrent("C:/goproj/peer2HttpDemo/torrents/ubuntu-20.04.5-live-server-amd64.iso.torrent")
	files := app.GetFiles("ubuntu-20.04.5-live-server-amd64.iso")
	//app.DownloadFile("ubuntu-20.04.5-live-server-amd64.iso", files[0], "C:/goproj/peer2HttpDemo/download/haojiahuo.iso")
	app.ReadFromHead("ubuntu-20.04.5-live-server-amd64.iso", files[0])

	fmt.Println("下载完了")
	// 关闭 TorrentClient
	app.Close()
}
