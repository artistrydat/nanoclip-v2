package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"paperclip-go/models"
)

func AssetRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	rg.GET("", listAssets(db))
	rg.POST("", createAsset(db))
	rg.GET("/:assetId", getAsset(db))
	rg.PATCH("/:assetId", updateAsset(db))
	rg.DELETE("/:assetId", deleteAsset(db))
}

func listAssets(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		companyID := c.Param("companyId")
		q := db.Where("company_id = ?", companyID).Order("created_at desc").Limit(200)
		if kind := c.Query("kind"); kind != "" {
			q = q.Where("kind = ?", kind)
		}
		if agentID := c.Query("agentId"); agentID != "" {
			q = q.Where("agent_id = ?", agentID)
		}
		var assets []models.Asset
		q.Find(&assets)
		c.JSON(http.StatusOK, assets)
	}
}

func createAsset(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name    string  `json:"name" binding:"required"`
			Kind    string  `json:"kind" binding:"required"`
			URL     *string `json:"url"`
			AgentID *string `json:"agentId"`
			IssueID *string `json:"issueId"`
			RunID   *string `json:"runId"`
			Size    *int64  `json:"size"`
			MimeType *string `json:"mimeType"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		asset := models.Asset{
			ID:        uuid.NewString(),
			CompanyID: c.Param("companyId"),
			Name:      req.Name,
			Kind:      req.Kind,
			URL:       req.URL,
			AgentID:   req.AgentID,
			IssueID:   req.IssueID,
			RunID:     req.RunID,
			Size:      req.Size,
			MimeType:  req.MimeType,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := db.Create(&asset).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, asset)
	}
}

func getAsset(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var asset models.Asset
		if err := db.First(&asset, "id = ? AND company_id = ?",
			c.Param("assetId"), c.Param("companyId")).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
			return
		}
		c.JSON(http.StatusOK, asset)
	}
}

func updateAsset(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name *string `json:"name"`
			URL  *string `json:"url"`
		}
		c.ShouldBindJSON(&req)
		updates := map[string]interface{}{"updated_at": time.Now()}
		if req.Name != nil {
			updates["name"] = *req.Name
		}
		if req.URL != nil {
			updates["url"] = *req.URL
		}
		db.Model(&models.Asset{}).
			Where("id = ? AND company_id = ?", c.Param("assetId"), c.Param("companyId")).
			Updates(updates)
		var asset models.Asset
		db.First(&asset, "id = ?", c.Param("assetId"))
		c.JSON(http.StatusOK, asset)
	}
}

func deleteAsset(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		db.Where("id = ? AND company_id = ?",
			c.Param("assetId"), c.Param("companyId")).Delete(&models.Asset{})
		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}
