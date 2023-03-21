package db

import "gorm.io/gorm"

type Magnet struct {
	gorm.Model
	Title  string
	Magnet string
	UserID uint `gorm:"ForeignKey:UserID;AssociationForeignKey:ID"`
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
