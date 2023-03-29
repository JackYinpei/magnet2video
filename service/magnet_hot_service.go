package service

import (
	"fmt"
	"peer2http/cache"
	"peer2http/db"
	"peer2http/serializer"
)

type MagnetHotService struct {
}

func (m *MagnetHotService) Get() serializer.Response {
	var magnets []db.Magnet

	mids, _ := cache.RedisClient.ZRevRange(cache.HotRankKey, 0, 9).Result()
	if len(mids) > 1 {
		// TODO 查询DB 中这些id 的magnet 信息到magnets里
		err := db.DB.Where("id in ?", mids).Find(&magnets).Error
		if err != nil {
			fmt.Println("emmm 这个数据库里没有id 是这些的magnet 吗？竟然查失败了", mids)
		}
	}
	magnetNames := make([]string, 0)
	for _, magnet := range magnets {
		magnetNames = append(magnetNames, magnet.Magnet)
	}

	return serializer.Response{
		Status: 20001,
		Msg:    "OK",
		Data:   magnetNames,
	}
}
