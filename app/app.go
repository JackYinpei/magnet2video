package app

import (
	"fmt"
	"github.com/anacrolix/torrent"
	"strings"
)

type App struct {
	client *torrent.Client

	torrents map[string]*torrent.Torrent
}

func New() (*App, error) {
	var client *torrent.Client

	clientConfig := torrent.NewDefaultClientConfig()
	client, err := torrent.NewClient(clientConfig)
	if err != nil {
		panic("初始化torrent client 失败")
	}

	var app = &App{
		client:   client,
		torrents: map[string]*torrent.Torrent{},
	}
	return app, nil
}

func (a *App) AddMagnet(magnet string) {
	if strings.HasPrefix(magnet, "magnet:") {
		t, err := a.client.AddMagnet(magnet)
		fmt.Println("wait magnet to get info")
		<-t.GotInfo()
		fmt.Println("Got info done")
		if err != nil {
			fmt.Println("ERROR client添加magnet 失败")
		}
		a.torrents[magnet] = t
	} else {
		fmt.Println("啊哦 输入的参数不太对")
	}
}

func (a *App) GetFiles(magnet string) []string {
	for k, v := range a.torrents {
		if magnet == k {
			//v.DownloadAll()
			files := v.Files()
			for _, file := range files {
				fmt.Printf("%v", file.Path())
				file.Download()
			}
		}
	}
	return nil
}
