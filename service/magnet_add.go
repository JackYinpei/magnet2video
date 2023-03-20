package service

import (
	"fmt"
	"peer2http/db"
	"peer2http/serializer"
)

type MagnetService struct {
	Magnet string `json:"magnet" form:"magnet" binding:"required"`
	Title  string `json:"title" form:"title"`
}

func (service *MagnetService) Create(userId uint) serializer.Response {
	magnet := db.Magnet{
		Title:  service.Title,
		Magnet: service.Magnet,
		UserID: userId,
	}
	// TODO 若数据库已经存在这个magnet 处理情况
	magnets := make([]db.Magnet, 0)
	result := db.DB.Where("magnet = ?", magnet.Magnet).Find(&magnets)
	if result.Error == nil {
		if len(magnets) > 1 {
			return serializer.Response{
				Status: 40001,
				Msg:    "你这个用户啊， 你你你 你已经创建过这个magnet 了",
			}
		}
	} else {
		fmt.Println("数据库查询失败", result.Error)
	}
	err := db.DB.Create(&magnet).Error
	if err != nil {
		return serializer.Response{
			Status: 40001,
			Msg:    "数据库添加magnet失败",
		}
	}
	return serializer.Response{
		Data: magnet,
	}
}
