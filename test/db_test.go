package test

import (
	"fmt"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username string
	Password string
	Magnets  []Magnet
	Shares   []Share
}

type Magnet struct {
	gorm.Model
	UserID         uint
	Name           string
	URL            string
	User           User `gorm:"foreignKey:UserID"`
	Shares         []Share
	ShareCondition bool
}

type Share struct {
	gorm.Model
	UserID   uint
	MagnetID uint
	User     User   `gorm:"foreignKey:UserID"`
	Magnet   Magnet `gorm:"foreignKey:MagnetID"`
}

func TestDBMagnetUserShare(t *testing.T) {
	dsn := "root:mysql@tcp(127.0.0.1:3306)/ceshj?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if !db.Migrator().HasTable(&User{}) {
		db.Migrator().AutoMigrate(&User{})
	}
	if !db.Migrator().HasTable(&Magnet{}) {
		db.Migrator().AutoMigrate(&Magnet{})
	}
	if !db.Migrator().HasTable(&Share{}) {
		db.Migrator().AutoMigrate(&Share{})
	}
	db.Migrator().AutoMigrate(&Magnet{})
	if err != nil {
		fmt.Println("初始化数据库失败 因为", err)
	}
	user := User{
		Username: "haojiahuo3 share",
		Password: "haojiahuo3 share",
	}

	id := db.Create(&user)
	var userAfterCreate User
	db.First(&userAfterCreate, "username = ?", "haojiahuo3 share")
	magnet1 := Magnet{
		Name:           "haojiahuomagnet3",
		URL:            "yahoo",
		UserID:         8,
		ShareCondition: true,
	}
	db.Create(&magnet1)
	if magnet1.ShareCondition {
		share := Share{
			UserID:   8,
			MagnetID: 5,
		}
		fmt.Println("???? woshi share de ya ")
		db.Create(&share)
	}

	fmt.Println(id, "创建的user 的id")
	fmt.Println("??? 怎么说 都不打印信息的吗")
}
