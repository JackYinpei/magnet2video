package main

import (
	"fmt"
	"github.com/anacrolix/torrent"
	"log"
	"peer2http/util"
	"time"
)

//magnet:?xt=urn:btih:DD5B2337F90EE4D34012F0C270825B9EFF6A7960
func main() {
	// 创建一个 TorrentClient
	clientConfig := torrent.NewDefaultClientConfig()
	clientConfig.DataDir = "./"
	clientConfig.NoUpload = true
	clientConfig.DisableTrackers = true
	clientConfig.DisableIPv6 = true
	client, err := torrent.NewClient(clientConfig)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("创建client 成功")

	// 打开一个 torrent 文件

	//t, _ := client.AddMagnet("magnet:?xt=urn:btih:DD5B2337F90EE4D34012F0C270825B9EFF6A7960")

	t, err := client.AddTorrentFromFile("./DD5B2337F90EE4D34012F0C270825B9EFF6A7960.torrent")
	fmt.Println("Torrent name:", t.Name())
	fmt.Println("Total size:", t.Length())
	tracker := util.NewTracker("./tracker.txt")
	trackers := tracker.GetTrackerList()
	t.AddTrackers(trackers)
	fmt.Println("添加Tracker 成功")
	<-t.GotInfo()
	t.DownloadAll()

	signal := make(chan struct{})
	// 列出所有文件
	for _, f := range t.Files() {
		fmt.Println(f.Path())
		fmt.Println("开始下载咯")
		//f.Download()

		go func() {
			old := int64(0)
			for true {
				select {
				case <-signal:
					return
				default:
					porcess := f.Length()
					processing := f.BytesCompleted()
					speed := processing - old
					old = processing
					fmt.Printf("一共%d 这么大，下载完了 %d 这么多 速度是这么快 %d M per second%%\r", porcess/1024/1024, processing/1024/1024, speed/1024/1024)
					time.Sleep(time.Second)
				}
			}
		}()
		client.WaitAll()
		signal <- struct{}{}
	}
	fmt.Println("下载完了")
	// 关闭 TorrentClient
	client.Close()
}
