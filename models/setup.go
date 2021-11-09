package models

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectToDatabase() {
	dsn := "root:pw@tcp(127.0.0.1:3306)/linkshortener"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("Database connection failed")
	}

	err = db.AutoMigrate(
		&ShortenedLink{},
		&User{},
		&Session{},
	)
	if err != nil {
		panic("Couldn't auto migrate DB models")
	}

	DB = db
}
