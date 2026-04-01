package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"paperclip-go/middleware"
	"paperclip-go/models"
)

func AuthRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	// UI calls /sign-up/email and /sign-in/email (Better-Auth style)
	rg.POST("/sign-up/email", signUp(db))
	rg.POST("/sign-in/email", signIn(db))
	// Plain variants for direct API use
	rg.POST("/sign-up", signUp(db))
	rg.POST("/sign-in", signIn(db))
	rg.POST("/sign-out", signOut(db))
	rg.GET("/get-session", getSession(db))
}

type signUpRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
}

func signUp(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req signUpRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var existing models.User
		if err := db.Where("email = ?", strings.ToLower(req.Email)).First(&existing).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
			return
		}

		now := time.Now()
		user := models.User{
			ID:        uuid.NewString(),
			Name:      req.Name,
			Email:     strings.ToLower(req.Email),
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := db.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
			return
		}

		passwordStr := string(hash)
		account := models.Account{
			ID:         uuid.NewString(),
			AccountID:  user.ID,
			ProviderID: "credential",
			UserID:     user.ID,
			Password:   &passwordStr,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		db.Create(&account)

		// First user becomes instance admin
		var roleCount int64
		db.Model(&models.InstanceUserRole{}).Count(&roleCount)
		if roleCount == 0 {
			db.Create(&models.InstanceUserRole{
				ID:        uuid.NewString(),
				UserID:    user.ID,
				Role:      "instance_admin",
				CreatedAt: now,
				UpdatedAt: now,
			})
		}

		session := createSession(db, user.ID, c)
		c.SetCookie("paperclip_session", session.Token, 30*24*3600, "/", "", false, true)
		c.JSON(http.StatusOK, gin.H{
			"user":    user,
			"session": session,
			"token":   session.Token,
		})
	}
}

type signInRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func signIn(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req signInRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var user models.User
		if err := db.Where("email = ?", strings.ToLower(req.Email)).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}

		var account models.Account
		if err := db.Where("user_id = ? AND provider_id = 'credential'", user.ID).First(&account).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}

		if account.Password == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(*account.Password), []byte(req.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}

		session := createSession(db, user.ID, c)
		c.SetCookie("paperclip_session", session.Token, 30*24*3600, "/", "", false, true)
		c.JSON(http.StatusOK, gin.H{
			"user":    user,
			"session": session,
			"token":   session.Token,
		})
	}
}

func signOut(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		actor := middleware.GetActor(c)
		if actor != nil && actor.UserID != "" {
			token, _ := c.Cookie("paperclip_session")
			if token != "" {
				db.Where("token = ?", token).Delete(&models.Session{})
			}
		}
		c.SetCookie("paperclip_session", "", -1, "/", "", false, true)
		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

func getSession(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		actor := middleware.GetActor(c)
		if actor == nil || actor.UserID == "" {
			c.JSON(http.StatusOK, gin.H{"session": nil, "user": nil})
			return
		}

		var user models.User
		if err := db.First(&user, "id = ?", actor.UserID).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{"session": nil, "user": nil})
			return
		}

		token, _ := c.Cookie("paperclip_session")
		var session models.Session
		db.Where("token = ?", token).First(&session)

		c.JSON(http.StatusOK, gin.H{
			"session": session,
			"user":    user,
		})
	}
}

func createSession(db *gorm.DB, userID string, c *gin.Context) models.Session {
	now := time.Now()
	ip := c.ClientIP()
	ua := c.Request.UserAgent()
	session := models.Session{
		ID:        uuid.NewString(),
		Token:     uuid.NewString() + uuid.NewString(),
		UserID:    userID,
		ExpiresAt: now.Add(30 * 24 * time.Hour),
		IPAddress: &ip,
		UserAgent: &ua,
		CreatedAt: now,
		UpdatedAt: now,
	}
	db.Create(&session)
	return session
}
