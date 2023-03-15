package test

import (
	"fmt"
	"path"
	"peer2http/util"
	"strings"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestTrim(t *testing.T) {
	tracker := util.NewTracker("../tracker.txt")
	trackers := tracker.GetTrackerList()
	for _, t := range trackers {
		fmt.Println(t)
	}
}

func TestHttpGet(t *testing.T) {
	downloader := util.NewDownloader("C:\\goproj\\peer2HttpDemo\\torrents")
	downloader.SetMagnet("magnet:?xt=urn:btih:DD5B2337F90EE4D34012F0C270825B9EFF6A7960")
	fileName := downloader.GetTorrent()
	if fileName == "" {
		t.Failed()
	}
}

func TestTrimName(t *testing.T) {
	filename := "ubuntu-20.04.5-live-server-amd64.iso.torrent"
	fmt.Println("qian mian", filename)
	extend := path.Ext(filename)
	fmt.Println(extend)
	nameonly := strings.TrimSuffix(filename, extend)
	fmt.Println(nameonly)
}

type User struct {
	gorm.Model
	Username string `json:"username"`
	Password string `json:"password"`
}

func TestDBCon(t *testing.T) {
	dsn := "root:mysql@tcp(127.0.0.1:3306)/haojiahuo?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if !db.Migrator().HasTable(&User{}) {
		db.Migrator().AutoMigrate(&User{})
	}
	if err != nil {
		fmt.Println("初始化数据库失败 因为", err)
	}
	user := User{
		Username: "haojiahuo",
		Password: "haojiahuo",
	}
	id := db.Create(&user)
	fmt.Println(id, "创建的user 的id")
}
