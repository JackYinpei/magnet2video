package db

import "gorm.io/gorm"

type Share struct {
	gorm.Model
	UserID   uint
	MagnetID uint
	User     User   `gorm:"foreignKey:UserID"`
	Magnet   Magnet `gorm:"foreignKey:MagnetID"`
}
