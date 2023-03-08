package app

import (
	"fmt"
	"github.com/anacrolix/torrent"
	"peer2http/util"
	"sync"
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
	// download file here
	downloadTo string
	Wg         *sync.WaitGroup
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
	singleFile := ""
	for key, value := range a.torrents {
		fmt.Println(key, "key", "下面是这个magnet里面的文件名")
		if key != hash {
			continue
		}
		for _, f := range value.Files() {
			fmt.Println(f.Path(), "")
			singleFile = f.Path()
			break
		}
	}
	return []string{singleFile}
}

func (a *App) Close() {
	a.client.Close()
}

func (a *App) getDownloadFileObj(hash, filePath string) (*torrent.File, bool) {
	for key, value := range a.torrents {
		// download file by hash and file path
		if key != hash {
			continue
		}
		for _, f := range value.Files() {
			if f.Path() == filePath {
				return f, true
			}
		}
	}
	return nil, false
}

func (a *App) DownloadFile(hash, filePath string) {
	f, ok := a.getDownloadFileObj(hash, filePath)
	if !ok {
		fmt.Println("没找到这个hash 对应的这个文件名")
	}
	r := f.NewReader()
	r.SetReadahead()
	//f.Download()
	//a.Wg.Add(1)
	//a.client.WaitAll()
	//a.Wg.Done()
}
