package service

import (
	"peer2http/db"
	"peer2http/serializer"
)

type UserRegisterService struct {
	UserName        string `form:"username" json:"username" binding:"required,min=3,max=20"`
	Password        string `form:"password" json:"password" binding:"required,min=3,max=20"`
	PasswordConfirm string `form:"password_confirm" json:"password_confirm" binding:"required,min=3,max=20"`
}

func (service *UserRegisterService) Valid() *serializer.Response {
	if service.Password != service.PasswordConfirm {
		return &serializer.Response{
			Status: 40001,
			Msg:    "两次输入的密码不同",
		}
	}
	count := int64(0)
	db.DB.Model(&db.User{}).Where("username = ?", service.UserName).Count(&count)
	if count > 0 {
		return &serializer.Response{
			Status: 40001,
			Msg:    "用户名已经被用啦",
		}
	}
	return nil
}

func (service *UserRegisterService) Register() (db.User, *serializer.Response) {
	user := db.User{
		Username: service.UserName,
		Status:   db.Active,
	}
	if err := service.Valid(); err != nil {
		return user, err
	}
	if err := user.SetPassWord(service.Password); err != nil {
		return user, &serializer.Response{
			Status: 40002,
			Msg:    "密码加密失败",
		}
	}
	if err := db.DB.Create(&user).Error; err != nil {
		return user, &serializer.Response{
			Status: 40002,
			Msg:    "注册失败， 写数据库失败",
		}
	}
	return user, nil
}
