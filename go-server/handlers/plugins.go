package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"paperclip-go/models"
)

func PluginRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	rg.GET("", listPlugins(db))
	rg.POST("", createPlugin(db))
	rg.GET("/ui-contributions", getPluginUIContributions(db))
	rg.GET("/:pluginId", getPlugin(db))
	rg.PATCH("/:pluginId", updatePlugin(db))
	rg.DELETE("/:pluginId", deletePlugin(db))
}

func listPlugins(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var plugins []models.Plugin
		db.Where("enabled = ?", true).Order("created_at asc").Find(&plugins)
		c.JSON(http.StatusOK, plugins)
	}
}

func getPluginUIContributions(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var plugins []models.Plugin
		db.Where("enabled = ? AND ui_contributions IS NOT NULL", true).Find(&plugins)
		contributions := []interface{}{}
		for _, p := range plugins {
			if p.UIContributions != nil {
				contributions = append(contributions, p.UIContributions)
			}
		}
		c.JSON(http.StatusOK, contributions)
	}
}

func getPlugin(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var plugin models.Plugin
		if err := db.First(&plugin, "id = ?", c.Param("pluginId")).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "plugin not found"})
			return
		}
		c.JSON(http.StatusOK, plugin)
	}
}

func createPlugin(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name    string      `json:"name" binding:"required"`
			Version *string     `json:"version"`
			Config  models.JSON `json:"config"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		plugin := models.Plugin{
			ID:        uuid.NewString(),
			Name:      req.Name,
			Version:   req.Version,
			Enabled:   true,
			Config:    req.Config,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		db.Create(&plugin)
		c.JSON(http.StatusCreated, plugin)
	}
}

func updatePlugin(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Enabled *bool       `json:"enabled"`
			Config  models.JSON `json:"config"`
		}
		c.ShouldBindJSON(&req)
		updates := map[string]interface{}{"updated_at": time.Now()}
		if req.Enabled != nil {
			updates["enabled"] = *req.Enabled
		}
		if req.Config != nil {
			updates["config"] = req.Config
		}
		db.Model(&models.Plugin{}).Where("id = ?", c.Param("pluginId")).Updates(updates)
		var plugin models.Plugin
		db.First(&plugin, "id = ?", c.Param("pluginId"))
		c.JSON(http.StatusOK, plugin)
	}
}

func deletePlugin(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		db.Where("id = ?", c.Param("pluginId")).Delete(&models.Plugin{})
		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}
