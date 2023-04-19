package app

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"peer2http/util"
	"strings"
	"sync"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"go.etcd.io/bbolt"
	"golang.org/x/time/rate"
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

	// add torrent to torrents will need lock
	mu sync.RWMutex
	// bolt.DB 创建bolt.db 当服务器退出后重启，可以从bolt.db 里加载client 信息
	db *bbolt.DB
	// trackers add a torrent obj need add tracker
	trackers    [][]string
	PostLimiter rate.Limiter
	PlayLimiter rate.Limiter
}

const (
	dbName       = ".app.bolt.db"
	dbBucketInfo = "torrent_info"
)

type file struct {
	file   map[string]*torrent.File
	reader torrent.Reader
}

func New(path string) (*App, error) {
	// don't know the client config well so create a default client
	client := util.NewClient()
	getter := util.NewDownload(path)
	loadDB, err := connectBoltDB("/data/go_proj/magnet2video/my.bolt.db")
	tracker := util.NewTracker("./tracker.txt")
	trackers := tracker.GetTrackerList()
	AppObj = &App{
		client:        client,
		torrents:      make(map[string]*torrent.Torrent),
		files:         make(map[string]*file, 0),
		torrentGetter: getter,
		db:            loadDB,
		trackers:      trackers,
		PostLimiter:   *rate.NewLimiter(1, 1),
		PlayLimiter:   *rate.NewLimiter(1, 1),
	}
	if err == nil {
		AppObj.Load()
	} else {
		fmt.Println("load client metainfo failed cause", err, "skip load db")
		// panci for test
		panic("load db faild")
	}
	return AppObj, nil
}

// background task to delete AppObj.files
func Background() context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for {
			select {
			case <-ticker.C:
				AppObj.mu.Lock()
				// delete selected k and v TODO
				AppObj.mu.Unlock()
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
	return cancel
}

func (a *App) Load() error {
	// control does not create too many goroutines max 32
	sema := make(chan struct{}, 32)
	defer close(sema)
	return a.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(dbBucketInfo))

		// I don't care about the key of the metainfo
		return b.ForEach(func(hashB, v []byte) error {
			var err error
			var mi *metainfo.MetaInfo
			// declare but not used
			// var t *torrent.Torrent
			hash := string(hashB[:])
			mi, err = metainfo.Load(bytes.NewReader(v))
			if err != nil {
				fmt.Println("启动服务 读取metainfo from bolt db 的一个magnet 失败 因为", err)
				// 只是读取一个失败 不至于panic 所以返回nil
				return nil
			}
			fmt.Println("metainfo readed from bbolt db here is the metainfo info TLTR")

			sema <- struct{}{}

			go func() {
				defer func() {
					<-sema
				}()
				t, err := a.client.AddTorrent(mi)
				if err != nil {
					fmt.Println("add torrent from metainfo error for", err)
				}
				// control multi goroutine lock
				a.mu.Lock()
				t.AddTrackers(a.trackers)
				a.torrents[hash] = t
				a.mu.Unlock()
			}()
			return nil
		})
	})
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
	if err != nil {
		fmt.Println("解析torrent 文件失败 因为：", err)
		return err
	}
	// get torrent file hash which is just downloaded by magnet and add torrent obj to app
	// this filename is download path with join torrent file name such as C:\goproj\peer2HttpDemo\torrents/
	// DD5B2337F90EE4D34012F0C270825B9EFF6A7960.torrent

	filename = path.Base(filename)
	hash := strings.TrimSuffix(filename, path.Ext(filename))
	fmt.Println("file name only ", hash, "file name only without suffix with postfix", filename)
	// wait get torrent metainfo
	<-t.GotInfo()
	fmt.Println("准备torrent 对象完成")
	// write hash and  torrent obj pari need add a lock
	a.mu.Lock()
	var mi = t.Metainfo()
	var buf = bytes.NewBuffer(nil)
	err = mi.Write(buf)
	if err != nil {
		fmt.Println("write metainfo to bbolt db fail cause ", err)
	}
	err = a.db.Update(func(tx *bbolt.Tx) error {
		var b = tx.Bucket([]byte(dbBucketInfo))
		fmt.Println("Here put t.InfoHash().Bytes() into db torrentInfoName: ", t.Info().Name, "here is magnet hash", hash)
		// TODO I want to put magnet hash as key Done!
		// return b.Put(t.InfoHash().Bytes(), buf.Bytes())
		return b.Put([]byte(hash), buf.Bytes())
	})
	if err != nil {
		fmt.Println("put metainfo bytes to db fail cause", err)
	}
	// Add tracker list to torrent obj
	t.AddTrackers(a.trackers)
	a.torrents[hash] = t
	a.mu.Unlock()
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
	// TODO It works after the following line added with no panci info but it still should have a better resolution to add hashfile obj into a.files map
	// plan A
	// a.GetFiles(hash)
	// if a.files[hash].reader == nil {
	// 	reader := f.NewReader()
	// 	a.files[hash].reader = reader
	// }
	// fmt.Println("zhe li ying gai you fan ying le", w.Header())
	// http.ServeContent(w, r, filename, time.Now(), a.files[hash].reader)

	// plan B
	fileReader := f.NewReader()
	filelength := f.Length()
	fip := f.FileInfo().Path
	var name string
	if len(fip) == 0 {
		name = f.DisplayPath()
	} else if len(fip) == 1 {
		name = fip[0]
	} else {
		name = fip[len(fip)-1]
	}
	fmt.Println(filelength, name, "before serve http file")
	if filelength > 0 {
		fileReader.SetReadahead((filelength * 10) / 100)
	}
	w.Header().Set("Content-Disposition", `filename="`+url.PathEscape(name)+`"`)
	_, err := fileReader.Seek(0, 0)
	if err != nil {
		fmt.Println(err)
	}
	defer fileReader.Close()
	http.ServeContent(w, r, filename, time.Now(), fileReader)
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
