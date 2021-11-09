package models

import (
	"math/rand"
	"time"
)

type ShortenedLink struct {
	URL            string `json:"url"`
	ShortPath      string `json:"short_path" gorm:"primaryKey;unique"`
	ExpirationTime int64  `json:"expiration_time"`
}

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ§$!-.+*"

var seeded = false

func GenerateShortPath() string {
	if !seeded {
		rand.Seed(time.Now().UnixNano())
		seeded = true
	}

	b := make([]byte, 3)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
