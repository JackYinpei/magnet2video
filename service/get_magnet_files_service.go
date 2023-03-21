package service

import (
	"fmt"
	"peer2http/app"
	"peer2http/db"
	"peer2http/serializer"
	"peer2http/util"
)

type MagnetListService struct {
	Magnet string `form:"magnet" binding:"required"`
	UserID uint   `form:"user_id" binding:"required"`
}

func (mf *MagnetListService) OwnThis() bool {
	var magnet db.Magnet
	db.DB.Preload("User").Where("Id = ?", mf.UserID).Find(&magnet)
	fmt.Printf("%v user id == %v 这个的所有的magnet", magnet, mf.UserID)
	return magnet.Magnet == mf.Magnet
}

func (mf *MagnetListService) GetMagnetService() serializer.Response {
	if !mf.OwnThis() {
		return serializer.Response{
			Status: 200,
			Msg:    "你还没添加过这个magnet 或者这个不是你的magnet",
		}
	}
	fileNames := app.AppObj.GetFiles(util.GetHash(mf.Magnet))
	return serializer.Response{
		Status: 200,
		Msg:    "success",
		Data:   fileNames,
	}
}
