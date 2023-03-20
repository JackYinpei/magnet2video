package app

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"peer2http/util"
	"strings"
	"sync"
	"time"

	"github.com/anacrolix/torrent"
)

var AppObj *App

type App struct {
	client *torrent.Client
	// key as torrent obj magnet hash value
	torrents map[string]*torrent.Torrent
	// key as torrent obj file name
	files map[string]*file
	// torrent file getter via magnet
	torrentGetter util.Magnet2TorrentGetter
	// torrent filenames which download via magnet
	fileNames []string
	// download file here
	downloadTo string
	Wg         *sync.WaitGroup
}

type file struct {
	file   map[string]*torrent.File
	reader torrent.Reader
}

func New(path string) (*App, error) {
	client := util.NewClient()
	getter := util.NewDownload(path)
	AppObj = &App{
		client:        client,
		torrents:      make(map[string]*torrent.Torrent),
		files:         make(map[string]*file, 0),
		torrentGetter: getter,
	}
	return AppObj, nil
}

func (a *App) AddMagnet(magnet string) error {
	fmt.Println(magnet, "magnet is going to add into client")
	a.torrentGetter.SetMagnet(magnet)
	fmt.Println("Get magnet torrent file done")
	// get torrent file by magnet
	filename := a.torrentGetter.GetTorrent()
	if filename == "" {
		fmt.Println("获取这个magnet 对应的torrent file失败")
		return errors.New("cannot find this magnet corresponding torrent")
	}
	// Get torrent by magnet should insensible
	err := a.GetTorrent(filename)
	if err != nil {
		return err
	}

	return nil
}

func (a *App) GetTorrent(filename string) error {
	// 将通过magnet 下载的torrent 文件 获取到torrent 对象
	t, err := a.client.AddTorrentFromFile(filename)
	fmt.Println("Add torrent obj to app")
	if err != nil {
		fmt.Println("解析torrent 文件失败 因为：", err)
		return err
	}
	// Add tracker list to torrent obj
	tracker := util.NewTracker("./tracker.txt")
	trackers := tracker.GetTrackerList()
	t.AddTrackers(trackers)
	// get torrent file hash which is just downloaded by magnet and add torrent obj to app
	// this filename is download path with join torrent file name such as C:\goproj\peer2HttpDemo\torrents/
	// DD5B2337F90EE4D34012F0C270825B9EFF6A7960.torrent

	filename = path.Base(filename)
	fmt.Println("file name only without suffix with postfix", filename)
	hash := strings.TrimSuffix(filename, path.Ext(filename))
	fmt.Println("file name only ", hash)
	// wait something I don't know
	<-t.GotInfo()
	t.DownloadAll()
	fmt.Println("准备torrent 对象完成")
	for _, file := range t.Files() {
		fmt.Println(file.Path(), "DEBUG 列出所有文件")
	}
	a.torrents[hash] = t
	return nil
}

func (a *App) GetFiles(hash string) []string {
	files := make([]string, 0)
	for key, value := range a.torrents {
		fmt.Println(key, "key", "下面是这个magnet里面的文件名", "进来的hash", hash, key == hash, "等于吗")
		if key != hash {
			continue
		}
		hashFiles := &file{
			file: make(map[string]*torrent.File, 0),
		}
		for _, f := range value.Files() {
			files = append(files, f.Path())
			hashFiles.file[f.Path()] = f
		}
		a.files[hash] = hashFiles
	}
	return files
}

func (a *App) ContentServer(w http.ResponseWriter, r *http.Request, hash string, filename string) {
	f, ok := a.getDownloadFileObj(hash, filename)
	if !ok {
		fmt.Println("没找到这个hash 对应的这个文件名")
	}
	if a.files[hash].reader == nil {
		reader := f.NewReader()
		a.files[hash].reader = reader
	}
	http.ServeContent(w, r, filename, time.Now(), a.files[hash].reader)
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

func (a *App) DownloadFile(hash, filePath, asFileName string) {
	fmt.Println("get hash in downloadFile", hash, "and filePath", filePath)
	f, ok := a.getDownloadFileObj(hash, filePath)
	if !ok {
		fmt.Println("没找到这个hash 对应的这个文件名")
	}

	r := f.NewReader()
	defer r.Close()
	file, err := os.OpenFile(asFileName, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		fmt.Println("创建文件失败， 因为", err)
	}
	outFile := bufio.NewWriter(file)
	n, err := io.Copy(outFile, r)
	fmt.Println("等待下载")
	if err != nil {
		fmt.Println("io copy 的时候出现错误 因为", err)
	}
	fmt.Println("一共这么大", n/1024/1024)
}

func (a *App) ReadFromHead(hash, filePath string) io.Reader {
	fmt.Println("get hash in downloadFile", hash, "and filePath", filePath)
	thisTorrent := a.torrents[hash]
	totalPiece := thisTorrent.NumPieces()
	fmt.Println("一共这么多块", totalPiece)
	priorityIndex := totalPiece / 4
	for i := 0; i <= priorityIndex; i++ {
		thisTorrent.Piece(i).SetPriority(torrent.PiecePriorityNow)
	}
	f, _ := a.getDownloadFileObj(hash, filePath)
	r := f.NewReader()
	defer r.Close()

	return r

}

func (a *App) name() {

}

func (a *App) ReadFromCurrent(hash, filePath string, position int64) {
	//begin := f.BeginPieceIndex()
	//end := f.EndPieceIndex()
	//completed := f.BytesCompleted()
}
