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
	db.DB.First(&magnet).Where("UserID = ?", mf.UserID)
	fmt.Printf("%v user id == %v 这个的所有的magnet %v %v %v \n", magnet, mf.UserID, mf.Magnet == magnet.Magnet, magnet.Magnet, mf.Magnet)
	return magnet.Magnet == mf.Magnet
}

func (mf *MagnetListService) GetMagnetFiles() serializer.Response {
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
