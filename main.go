package main

import (
	"fmt"
	app2 "peer2http/app"
)

//magnet:?xt=urn:btih:DD5B2337F90EE4D34012F0C270825B9EFF6A7960
func main() {
	//// 创建一个 TorrentClient
	//client := util.NewClient()
	//
	//// new magnet 2 torrent file getter
	//// download torrent file to this path
	//downloader := util.NewDownload("C:\\goproj\\peer2HttpDemo\\torrents")
	//downloader.SetMagnet("magnet:?xt=urn:btih:DD5B2337F90EE4D34012F0C270825B9EFF6A7960")
	//fileName := downloader.GetTorrent()
	//if fileName == "" {
	//	fmt.Println("通过magnet 获取torrent 文件失败")
	//}
	//
	//// Open torrent file
	//t, err := client.AddTorrentFromFile(fileName)
	//if err != nil {
	//	fmt.Println("解析torrent 文件失败")
	//}
	//// some torrent info
	//fmt.Println("Torrent name:", t.Name())
	//fmt.Println("Total size:", t.Length())
	//// Add torrent client tracker
	//tracker := util.NewTracker("./tracker.txt")
	//trackers := tracker.GetTrackerList()
	//t.AddTrackers(trackers)
	//fmt.Println("添加Tracker 成功")
	//<-t.GotInfo()
	//t.DownloadAll()

	// new app obj
	app, err := app2.New("C:\\goproj\\peer2HttpDemo\\torrents")
	if err != nil {
		panic("")
	}
	// add magnet to app
	app.AddMagnet("magnet:?xt=urn:btih:DD5B2337F90EE4D34012F0C270825B9EFF6A7960")
	// get torrent files inside by hash or magnet
	app.GetFiles("magnet:?xt=urn:btih:DD5B2337F90EE4D34012F0C270825B9EFF6A7960")
	//signal := make(chan struct{})
	//// 列出所有文件
	//for _, f := range t.Files() {
	//	fmt.Println(f.Path())
	//	fmt.Println("开始下载咯")
	//	//f.Download()
	//
	//	go func() {
	//		old := int64(0)
	//		for true {
	//			select {
	//			case <-signal:
	//				return
	//			default:
	//				porcess := f.Length()
	//				processing := f.BytesCompleted()
	//				speed := processing - old
	//				old = processing
	//				fmt.Printf("一共%d 这么大，下载完了 %d 这么多 速度是这么快 %d M per second%%\r", porcess/1024/1024, processing/1024/1024, speed/1024/1024)
	//				time.Sleep(time.Second)
	//			}
	//		}
	//	}()
	//	client.WaitAll()
	//	signal <- struct{}{}
	//}
	fmt.Println("下载完了")
	// 关闭 TorrentClient
	app.Close()
}
