package util

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	path2 "path"
	"strings"
)

//http://itorrents.org/torrent/哈希值（40位大写字母）.torrent
//http://storetorrents.com/hash/哈希值（40位大写字母）

type Magnet2TorrentGetter interface {
	// SetMagnet 输入给定的磁力链接
	SetMagnet(string2 string)
	// GetTorrent 获取torrent 文件 给定torrent 文件下载到这个路径
	GetTorrent() string
}

type ITorrentsDownloader struct {
	magnet          string
	path            string
	torrentPath     string
	torrentHttpPath string
	serverPath      string
}

func NewDownloader(path string) *ITorrentsDownloader {
	return &ITorrentsDownloader{
		magnet:          "",
		path:            path,
		torrentPath:     "",
		torrentHttpPath: "",
		serverPath:      "http://itorrents.org/torrent",
	}
}

func NewDownload(path string) Magnet2TorrentGetter {
	return &ITorrentsDownloader{
		magnet:          "",
		path:            path,
		torrentPath:     "",
		torrentHttpPath: "",
		serverPath:      "http://itorrents.org/torrent",
	}
}

func (I *ITorrentsDownloader) SetMagnet(string2 string) {
	I.magnet = string2
	I.getHash()
}

func (I *ITorrentsDownloader) GetTorrent() string {
	// 拼接出Torrent文件 的Http url 路径
	I.concatTorrentHttpPath()
	// 通过http 请求 下载那个链接
	strURL := I.torrentHttpPath
	resp, err := http.Head(strURL)
	if err != nil {
		fmt.Println("resp, err := http.Head(strURL)  报错: strURL = ", strURL)
		log.Fatalln(err)
	}
	fmt.Println(strURL, "torrent"+
		" 路径地址")

	// fmt.Printf("%#v\n", resp)
	fileLength := int(resp.ContentLength)

	req, err := http.NewRequest("GET", strURL, nil)
	if err != nil {
		fmt.Println(err)
		log.Fatal(err)
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", 0, fileLength))
	// fmt.Printf("%#v", req)

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("http.DefaultClient.Do(req)", "error")
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// 创建文件

	filename := path2.Base(strURL)
	filename = path2.Join(I.path, filename)
	flags := os.O_CREATE | os.O_WRONLY
	f, err := os.OpenFile(filename, flags, 0666)
	if err != nil {
		fmt.Println("创建文件失败", err)
		log.Fatal("err")
	}
	defer f.Close()

	// 写入数据
	buf := make([]byte, 16*1024)
	_, err = io.CopyBuffer(f, resp.Body, buf)
	if err != nil {
		if err == io.EOF {
			fmt.Println("io.EOF")
			return ""
		}
		fmt.Println(err)
		log.Fatal(err)
	}
	return filename
}

// 将magnet信息去除开头 得到纯hash数字
func (I *ITorrentsDownloader) getHash() {
	//	magnet:?xt=urn:btih:DD5B2337F90EE4D34012F0C270825B9EFF6A7960
	splitList := strings.Split(I.magnet, ":")
	fmt.Println(splitList, "分割出来的路径", "hash: ", splitList[len(splitList)-1])
	I.magnet = splitList[len(splitList)-1]
}

func (I *ITorrentsDownloader) concatTorrentHttpPath() {
	I.torrentHttpPath = I.serverPath + "/" + I.magnet + ".torrent"
}
