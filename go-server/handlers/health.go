package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"paperclip-go/models"
)

const ServerVersion = "0.1.0-go"

func HealthHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var runningCount int64
		db.Model(&models.HeartbeatRun{}).
			Where("status IN ('queued','running')").
			Count(&runningCount)

		var userCount int64
		db.Model(&models.User{}).Count(&userCount)

		bootstrapStatus := "ready"
		if userCount == 0 {
			bootstrapStatus = "bootstrap_pending"
		}

		c.JSON(http.StatusOK, gin.H{
			"status":          "ok",
			"version":         ServerVersion,
			"deploymentMode":  "local_trusted",
			"deploymentExposure": "private",
			"authReady":       true,
			"bootstrapStatus": bootstrapStatus,
			"activeRuns":      runningCount,
			"features": gin.H{
				"companyDeletionEnabled": true,
			},
		})
	}
}
