package db

import (
	"peer2http/cache"
	"strconv"

	"gorm.io/gorm"
)

//	type Magnet struct {
//		gorm.Model
//		Title  string
//		Magnet string
//		UserID uint
//		User   User `gorm:"ForeignKey:ID"`
//		Share  bool
//	}
type Magnet struct {
	gorm.Model
	Title          string
	Magnet         string
	UserID         uint
	User           User `gorm:"foreignKey:UserID"`
	Shares         []Share
	ShareCondition bool
	Count          uint
}

func (magnet *Magnet) Usage() uint64 {
	// TODO use redis to display usage
	return 0
}

func createMagnetTable() {
	if !DB.Migrator().HasTable(&Magnet{}) {
		DB.Migrator().AutoMigrate(&Magnet{})
	}
	DB.AutoMigrate(&Magnet{})
}

func (magnet *Magnet) AddView() {
	// 增加magnet的点击数
	cache.UserMagnetViewCountPlusOne(magnet.UserID, magnet.Magnet)
	cache.RedisClient.Incr(cache.MagnetViewKey(magnet.ID))
	// 增加这个magnet ID 在hot rank里的排名
	cache.RedisClient.ZIncrBy(cache.HotRankKey, 1, strconv.Itoa(int(magnet.ID)))
}
