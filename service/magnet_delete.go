package service

import (
	"fmt"
	"peer2http/db"
	"peer2http/serializer"
)

type MagnetServiceDelete struct {
	Magnet string `form:"magnet" json:"magnet"`
	UserId uint   `form:"userid" json:"userid"`
}

func (del *MagnetServiceDelete) Delete() serializer.Response {
	magnet := db.Magnet{
		Magnet: del.Magnet,
		UserID: del.UserId,
	}
	err := db.DB.Where("Magnet = ?", del.Magnet).Delete(&magnet).Error
	fmt.Println(err, "error occur in db delete magnet ", magnet.Magnet, magnet.UserID)
	if err != nil {
		return serializer.Response{
			Status: 40001,
			Msg:    err.Error(),
		}
	}
	return serializer.Response{
		Status: 40001,
		Msg:    "OK 删除成功",
	}
}
