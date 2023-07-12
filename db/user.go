package db

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func createUserTable() {
	if !DB.Migrator().HasTable(&User{}) {
		DB.Migrator().AutoMigrate(&User{})
	}
	DB.AutoMigrate(&User{})
}

// type User struct {
// 	gorm.Model
// 	Username string   `json:"username" grom:"username"`
// 	Password string   `json:"password" gorm:"password"`
// 	Status   string   `json:"status" gorm:"status"`
// 	Avatar   string   `gorm:"size:1000"`
// 	Magnets  []Magnet `gorm:"foreignKey:ID"`
// }

type User struct {
	gorm.Model
	Username string `json:"username" grom:"username"`
	Password string `json:"password" gorm:"password"`
	Status   string `json:"status" gorm:"status"`
	Avatar   string `gorm:"size:1000"`
	Magnets  []Magnet
	Shares   []Share
}

const (
	PasswordCost        = 12
	Active       string = "active"
	InActive     string = "inactive"
	Suspend      string = "suspend"
)

func GetUser(ID interface{}) (User, error) {
	var user User
	result := DB.First(&user, ID)
	return user, result.Error
}

func (user *User) SetPassWord(password string) error {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), PasswordCost)
	if err != nil {
		return err
	}
	user.Password = string(bytes)
	return nil
}

func (user *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	return err == nil
}
