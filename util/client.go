package util

import (
	"fmt"
	"github.com/anacrolix/torrent"
)

func NewClient() *torrent.Client {
	clientConfig := torrent.NewDefaultClientConfig()
	clientConfig.DataDir = "./"
	clientConfig.NoUpload = true
	clientConfig.DisableTrackers = true
	clientConfig.DisableIPv6 = true
	client, err := torrent.NewClient(clientConfig)
	if err != nil {
		panic("创建client 失败")
	}
	fmt.Println("创建client 成功")
	return client
}
