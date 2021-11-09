package models

import (
	"encoding/base64"
	"github.com/gin-gonic/gin"
	"net/http"
)

type User struct {
	ID       uint   `json:"-" gorm:"primaryKey"`
	Username string `json:"username" gorm:"unique"`
	Password string `json:"-"`
	IsAdmin  bool   `json:"is_admin"`
}

func CreateAdminUser(password string) error {
	password, err := HashPassword(password)
	if err != nil {
		return err
	}

	user := User{
		Username: "admin",
		Password: password,
		IsAdmin:  true,
	}
	DB.Create(&user)

	return nil
}

type Session struct {
	Token  string `json:"token" gorm:"primaryKey;unique"`
	User   User   `json:"user" gorm:"foreignKey:UserID"`
	UserID uint   `json:"-"`
}

func NewSession(user *User) (*Session, error) {
	token, err := GenerateRandomBytes(32)
	if err != nil {
		return nil, err
	}

	session := &Session{
		User:  *user,
		Token: base64.URLEncoding.EncodeToString(token),
	}

	// save session to gorm
	if err := DB.Create(session).Error; err != nil {
		return nil, err
	}

	return session, nil
}

func (user *User) CheckPassword(password string) bool {
	val, _ := VerifyPassword(password, user.Password)
	return val
}

func RequiresAuth(handler func(*gin.Context, *Session)) func(*gin.Context) {
	return func(ctx *gin.Context) {
		token, err := ctx.Cookie("LinkSession")
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "not logged in"})
			return
		}

		// get the session for the session token
		var session Session
		err = DB.Preload("User").Where("token = ?", token).First(&session).Error
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
			return
		}

		handler(ctx, &session)
	}
}
