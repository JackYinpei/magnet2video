package app

import (
	"bufio"
	"fmt"
	"github.com/anacrolix/torrent"
	"io"
	"os"
	"path"
	"peer2http/util"
	"strings"
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
	// this filename is download path with join torrent file name such as C:\goproj\peer2HttpDemo\torrents/
	// DD5B2337F90EE4D34012F0C270825B9EFF6A7960.torrent

	filename = path.Base(filename)
	hash := strings.Trim(filename, path.Ext(filename))
	// wait something I don't know
	<-t.GotInfo()
	t.DownloadAll()
	fmt.Println("准备torrent 对象完成")
	for _, file := range t.Files() {
		fmt.Println(file.Path(), "DEBUG 列出所有文件")
	}
	fmt.Println(hash, "这是添加torrent 对象到app的map 时的hash值， 要是和后面的一样，那就是这里添加的问题，可能是指针的问题")
	a.torrents[hash] = t
}

func (a *App) GetFiles(hash string) []string {
	files := make([]string, 0)
	for key, value := range a.torrents {
		fmt.Println(key, "key", "下面是这个magnet里面的文件名", "进来的hash", hash, key == hash, "等于吗")
		if key != hash {
			continue
		}
		for _, f := range value.Files() {
			fmt.Println(f.Path(), " hash 里的一个文件名")
			files = append(files, f.Path())
		}
	}
	return files
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
			fmt.Println(f.Path(), "这里是在a.torrents 里找到对应的hash 后再在hash对应的torrent 对象里找符合入参filepath 的torrent.file 对象", filePath)
			if f.Path() == filePath {
				return f, true
			}
		}
	}
	return nil, false
}

func (a *App) DownloadFile(hash, filePath string) {
	fmt.Println("get hash in downloadFile", hash, "and filePath", filePath)
	f, ok := a.getDownloadFileObj(hash, filePath)
	if !ok {
		fmt.Println("没找到这个hash 对应的这个文件名")
	}

	r := f.NewReader()
	defer r.Close()
	cwd, _ := os.Getwd()
	filePath = path.Join(cwd, "haojiahuo.mkv")
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		fmt.Println("创建文件失败， 因为", err)
	}
	outFile := bufio.NewWriter(file)
	n, err := io.Copy(outFile, r)
	fmt.Println("等待下载")
	//f.Download()
	if err != nil {
		fmt.Println("io copy 的时候出现错误 因为", err)
	}
	fmt.Println("一共这么大", n/1024/1024)
	//f.Download()
	//a.Wg.Add(1)
	//a.client.WaitAll()
	//a.Wg.Done()
}
