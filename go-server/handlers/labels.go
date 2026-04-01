package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"paperclip-go/models"
)

func LabelRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	rg.GET("", listLabels(db))
	rg.POST("", createLabel(db))
}

func GlobalLabelRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	rg.DELETE("/:labelId", deleteLabel(db))
}

func listLabels(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var labels []models.Label
		db.Where("company_id = ?", c.Param("companyId")).Order("name asc").Find(&labels)
		c.JSON(http.StatusOK, labels)
	}
}

func createLabel(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name        string  `json:"name" binding:"required"`
			Color       *string `json:"color"`
			Description *string `json:"description"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		color := "#6b7280"
		if req.Color != nil {
			color = *req.Color
		}
		label := models.Label{
			ID:          uuid.NewString(),
			CompanyID:   c.Param("companyId"),
			Name:        req.Name,
			Color:       color,
			Description: req.Description,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := db.Create(&label).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, label)
	}
}

func deleteLabel(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := db.Delete(&models.Label{}, "id = ?", c.Param("labelId")).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "label not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}
