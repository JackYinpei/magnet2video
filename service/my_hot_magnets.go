package service

import (
	"peer2http/cache"
	"peer2http/serializer"
)

type MyHotMangetsService struct {
	Limit        int `form:"limit"`
	Start        int `form:"start"`
	MagnetString []string
}

func (m *MyHotMangetsService) GetMyLove(userID uint) serializer.Response {
	if m.Limit == 0 {
		m.Limit = 6
	}
	cache.RedisClient.
}