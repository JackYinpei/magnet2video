package service

import (
	"peer2http/db"
	"peer2http/serializer"
)

type ListMagnetsService struct {
	Limit   int `form:"limit"`
	Start   int `form:"start"`
	Magnets []MagnetService
}

func (list *ListMagnetsService) Create(userID uint) serializer.Response {
	magnets := []db.Magnet{}
	total := int64(0)
	if list.Limit == 0 {
		list.Limit = 6
	}

	if err := db.DB.Model(db.Magnet{}).Count(&total).Error; err != nil {
		return serializer.Response{
			Status: 40001,
			Msg:    "与数据库的链接出现了错误",
		}
	}

	if err := db.DB.Limit(list.Limit).Offset(list.Start).Find(&magnets).Error; err != nil {
		return serializer.Response{
			Status: 50000,
			Msg:    "数据库错误",
			Error:  err.Error(),
		}
	}
	return serializer.BuildListResponse(serializer.BuildMagnetList(magnets), uint(total))
}
