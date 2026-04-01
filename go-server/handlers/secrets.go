package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"paperclip-go/models"
)

// CompanySecretRoutes registers secret routes scoped to a company
func CompanySecretRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	rg.GET("", listCompanySecrets(db))
	rg.POST("", createCompanySecret(db))
}

// GlobalSecretRoutes registers secret routes by secret ID (no company scope)
func GlobalSecretRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	rg.GET("/:secretId", getSecret(db))
	rg.PATCH("/:secretId", updateSecret(db))
	rg.POST("/:secretId/rotate", rotateSecret(db))
	rg.DELETE("/:secretId", deleteSecret(db))
}

func listCompanySecrets(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var secrets []models.CompanySecret
		db.Where("company_id = ?", c.Param("companyId")).
			Order("created_at asc").Find(&secrets)
		// Mask values in response
		for i := range secrets {
			secrets[i].Value = "[redacted]"
		}
		c.JSON(http.StatusOK, secrets)
	}
}

func createCompanySecret(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Key         string  `json:"key" binding:"required"`
			Value       string  `json:"value" binding:"required"`
			Description *string `json:"description"`
			Kind        *string `json:"kind"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		kind := "env"
		if req.Kind != nil {
			kind = *req.Kind
		}
		secret := models.CompanySecret{
			ID:          uuid.NewString(),
			CompanyID:   c.Param("companyId"),
			Key:         req.Key,
			Value:       req.Value,
			Description: req.Description,
			Kind:        kind,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := db.Create(&secret).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		secret.Value = "[redacted]"
		c.JSON(http.StatusCreated, secret)
	}
}

func getSecret(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var secret models.CompanySecret
		if err := db.First(&secret, "id = ?", c.Param("secretId")).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "secret not found"})
			return
		}
		secret.Value = "[redacted]"
		c.JSON(http.StatusOK, secret)
	}
}

func updateSecret(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Key         *string `json:"key"`
			Value       *string `json:"value"`
			Description *string `json:"description"`
		}
		c.ShouldBindJSON(&req)
		updates := map[string]interface{}{"updated_at": time.Now()}
		if req.Key != nil {
			updates["key"] = *req.Key
		}
		if req.Value != nil {
			updates["value"] = *req.Value
		}
		if req.Description != nil {
			updates["description"] = *req.Description
		}
		db.Model(&models.CompanySecret{}).Where("id = ?", c.Param("secretId")).Updates(updates)
		var secret models.CompanySecret
		db.First(&secret, "id = ?", c.Param("secretId"))
		secret.Value = "[redacted]"
		c.JSON(http.StatusOK, secret)
	}
}

func rotateSecret(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Value string `json:"value" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		now := time.Now()
		db.Model(&models.CompanySecret{}).Where("id = ?", c.Param("secretId")).
			Updates(map[string]interface{}{"value": req.Value, "updated_at": now})
		c.JSON(http.StatusOK, gin.H{"success": true, "rotatedAt": now})
	}
}

func deleteSecret(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		db.Where("id = ?", c.Param("secretId")).Delete(&models.CompanySecret{})
		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}
