package main

import (
	"LinkShortener/models"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func ping(ctx *gin.Context) {
	ctx.String(http.StatusOK, "LinkShortener v1 by xImAnton_")
}

func resolveAndRedirectLink(ctx *gin.Context) {
	shortcut := ctx.Param("shortcut")
	var link models.ShortenedLink
	err := models.DB.Where("short_path = ?", shortcut).First(&link).Error
	if err != nil {
		// todo: redirect to frontend 404 page
		ctx.JSON(http.StatusNotFound, gin.H{"error": "invalid link shortcut"})
		return
	}

	if link.ExpirationTime > 0 && link.ExpirationTime < time.Now().Unix() {
		// todo: redirect to frontend 404 page
		models.DB.Delete(&link)
		ctx.JSON(http.StatusNotFound, gin.H{"error": "invalid link shortcut"})
		return
	}

	ctx.Redirect(http.StatusPermanentRedirect, link.URL)
}

func createShortenedLink(ctx *gin.Context, _ *models.Session) {
	var link models.ShortenedLink
	err := ctx.BindJSON(&link)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	if link.ExpirationTime < 0 {
		link.ExpirationTime = -1
	}

	_, err = url.ParseRequestURI(link.URL)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid url"})
		return
	}
	exists := true
	customShortLink := len(link.ShortPath) >= 3

	for exists {
		if !customShortLink {
			link.ShortPath = models.GenerateShortPath()
		}

		err = models.DB.Create(&link).Error
		if err != nil {
			var mysqlErr *mysql.MySQLError
			if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
				if customShortLink {
					ctx.JSON(http.StatusBadRequest, gin.H{"error": "link already shortcut exists"})
					return
				} else {
					continue
				}
			}

			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "couldn't create link"})
			return
		}
		exists = false
	}

	ctx.JSON(http.StatusOK, link)
}

func loginUser(ctx *gin.Context) {
	var pl struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	err := ctx.BindJSON(&pl)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	var user models.User
	err = models.DB.Where("username = ?", pl.Username).First(&user).Error
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid username or password"})
		return
	}
	if !user.CheckPassword(pl.Password) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid username or password"})
		return
	}

	// create a new session for the user
	session, err := models.NewSession(&user)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not create session"})
		return
	}

	// set session cookie
	ctx.SetCookie("LinkSession", session.Token, 3600, "/", "", false, true)
	ctx.JSON(http.StatusOK, gin.H{"message": "logged in successfully"})
}

func testLogin(ctx *gin.Context, user *models.Session) {
	ctx.JSON(http.StatusOK, *user)
}

// endpoint for getting shortened links with pagination
func getShortenedLinks(ctx *gin.Context, _ *models.Session) {
	var links []models.ShortenedLink
	var count int64
	var err error

	page, err := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid page value"})
		return
	}

	perPage, err := strconv.Atoi(ctx.DefaultQuery("per_page", "32"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid per_page value"})
		return
	}

	err = models.DB.Model(&models.ShortenedLink{}).Count(&count).Error
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not get links"})
		return
	}

	err = models.DB.Limit(perPage).Offset((page - 1) * perPage).Find(&links).Error
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not get links"})
		return
	}

	ctx.JSON(http.StatusOK, links)
}

func removeLink(ctx *gin.Context, _ *models.Session) {
	shortcut := ctx.Param("shortcut")
	var link models.ShortenedLink
	err := models.DB.Where("short_path = ?", shortcut).First(&link).Error
	if err != nil {
		// todo: redirect to frontend 404 page
		ctx.JSON(http.StatusNotFound, gin.H{"error": "invalid link shortcut"})
		return
	}

	err = models.DB.Delete(&link).Error
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not delete link"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "link deleted"})
}

// logout handler
func logoutUser(ctx *gin.Context, session *models.Session) {
	// delete session
	err := models.DB.Delete(&session).Error
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not delete session"})
		return
	}

	// clear cookie
	ctx.SetCookie("LinkSession", "", -1, "/", "", false, true)

	ctx.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

func main() {
	r := gin.Default()
	r.GET("/", ping)
	r.GET("/:shortcut", resolveAndRedirectLink)
	r.POST("/shorten", models.RequiresAuth(createShortenedLink))
	r.POST("/login", loginUser)
	r.GET("/user", models.RequiresAuth(testLogin))
	r.GET("/links", models.RequiresAuth(getShortenedLinks))
	r.DELETE("/:shortcut", models.RequiresAuth(removeLink))
	r.GET("/logout", models.RequiresAuth(logoutUser))

	models.ConnectToDatabase()

	err := r.Run(":3000")
	if err != nil {
		return
	}
}
