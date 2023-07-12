package service

import (
	"peer2http/db"
	"peer2http/serializer"
)

type UserLoginService struct {
	UserName string `json:"username" form:"username" binding:"required,min=3,max=20"`
	Password string `json:"password" form:"password" binding:"required,min=3,max=20"`
}

func (service *UserLoginService) Login() (db.User, *serializer.Response) {
	var user db.User
	if err := db.DB.Where("username = ?", service.UserName).First(&user).Error; err != nil {
		return user, &serializer.Response{
			Status: 40001,
			Msg:    "账号错误",
		}
	}
	return user, nil
}
