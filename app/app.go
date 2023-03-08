package app

import (
	"fmt"
	"github.com/anacrolix/torrent"
	"peer2http/util"
)

type App struct {
	client *torrent.Client
	// key as torrent obj magnet hash value
	torrents map[string]*torrent.Torrent
	// key as torrent obj file name
	files map[string]*torrent.File
	// torrent file getter via magnet
	torrentGetter util.Magnet2TorrentGetter
	// torrent filenames which download via magnet
	fileNames []string
}

func New(path string) (*App, error) {
	client := util.NewClient()
	getter := util.NewDownload(path)
	return &App{
		client:        client,
		torrents:      make(map[string]*torrent.Torrent),
		files:         nil,
		torrentGetter: getter,
	}, nil
}

func (a *App) AddMagnet(magnet string) {
	a.torrentGetter.SetMagnet(magnet)
	fmt.Println("Get magnet torrent file done")
	// get torrent file by magnet
	filename := a.torrentGetter.GetTorrent()
	if filename == "" {
		fmt.Println("获取这个magnet 对应的torrent file失败")
	}
	// Get torrent by magnet should insensible
	a.GetTorrent(filename)
}

func (a *App) GetTorrent(filename string) {
	// 将通过magnet 下载的torrent 文件 获取到torrent 对象
	t, err := a.client.AddTorrentFromFile(filename)
	fmt.Println("Add torrent obj to app")
	if err != nil {
		fmt.Println("解析torrent 文件失败 因为：", err)
	}
	// Add tracker list to torrent obj
	tracker := util.NewTracker("./tracker.txt")
	trackers := tracker.GetTrackerList()
	t.AddTrackers(trackers)
	// get torrent file hash which is just downloaded by magnet and add torrent obj to app
	hash := util.GetHash(filename)
	// wait something I don't know
	<-t.GotInfo()
	t.DownloadAll()
	a.torrents[hash] = t
}

func (a *App) GetFiles(hash string) []string {
	for key, value := range a.torrents {
		fmt.Println(key, "key", "下面是这个magnet里面的文件名")
		for _, f := range value.Files() {
			fmt.Println(f.Path(), "")
		}
	}
	return nil
}

func (a *App) Close() {
	a.client.Close()
}
