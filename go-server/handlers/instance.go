package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"paperclip-go/models"
)

func InstanceRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	rg.GET("/settings/general", getInstanceSettings(db, "general"))
	rg.PATCH("/settings/general", updateInstanceSettings(db, "general"))
	rg.GET("/settings/experimental", getInstanceSettings(db, "experimental"))
	rg.PATCH("/settings/experimental", updateInstanceSettings(db, "experimental"))
	rg.GET("/scheduler-heartbeats", getSchedulerHeartbeats(db))
	rg.GET("/users", listInstanceUsers(db))
	rg.PATCH("/users/:userId/role", updateInstanceUserRole(db))
}

func getInstanceSettings(db *gorm.DB, section string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var settings []models.InstanceSetting
		db.Where("section = ?", section).Find(&settings)

		result := map[string]interface{}{}
		for _, s := range settings {
			result[s.Key] = s.Value
		}
		c.JSON(http.StatusOK, result)
	}
}

func updateInstanceSettings(db *gorm.DB, section string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req map[string]interface{}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		now := time.Now()
		for k, v := range req {
			var setting models.InstanceSetting
			err := db.Where("section = ? AND key = ?", section, k).First(&setting).Error
			if err != nil {
				// Create new
				valStr := ""
				if v != nil {
					switch val := v.(type) {
					case string:
						valStr = val
					case bool:
						if val {
							valStr = "true"
						} else {
							valStr = "false"
						}
					default:
						valStr = ""
					}
				}
				db.Create(&models.InstanceSetting{
					ID:        uuid.NewString(),
					Section:   section,
					Key:       k,
					Value:     valStr,
					CreatedAt: now,
					UpdatedAt: now,
				})
			} else {
				// Update existing
				valStr := ""
				if v != nil {
					switch val := v.(type) {
					case string:
						valStr = val
					case bool:
						if val {
							valStr = "true"
						} else {
							valStr = "false"
						}
					}
				}
				db.Model(&setting).Updates(map[string]interface{}{"value": valStr, "updated_at": now})
			}
		}

		// Return updated settings
		var settings []models.InstanceSetting
		db.Where("section = ?", section).Find(&settings)
		result := map[string]interface{}{}
		for _, s := range settings {
			result[s.Key] = s.Value
		}
		c.JSON(http.StatusOK, result)
	}
}

func getSchedulerHeartbeats(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var agents []models.Agent
		db.Where("role = 'scheduler' OR adapter_type = 'scheduler'").
			Order("created_at asc").Find(&agents)
		c.JSON(http.StatusOK, agents)
	}
}

func listInstanceUsers(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var users []models.User
		db.Order("created_at asc").Find(&users)

		type userWithRole struct {
			models.User
			InstanceRole *string `json:"instanceRole,omitempty"`
		}
		result := make([]userWithRole, 0, len(users))
		for _, u := range users {
			var role models.InstanceUserRole
			uwr := userWithRole{User: u}
			if db.Where("user_id = ?", u.ID).First(&role).Error == nil {
				uwr.InstanceRole = &role.Role
			}
			result = append(result, uwr)
		}
		c.JSON(http.StatusOK, result)
	}
}

func updateInstanceUserRole(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Role string `json:"role" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		userID := c.Param("userId")
		now := time.Now()

		var role models.InstanceUserRole
		if db.Where("user_id = ?", userID).First(&role).Error != nil {
			db.Create(&models.InstanceUserRole{
				ID:        uuid.NewString(),
				UserID:    userID,
				Role:      req.Role,
				CreatedAt: now,
				UpdatedAt: now,
			})
		} else {
			db.Model(&role).Updates(map[string]interface{}{"role": req.Role, "updated_at": now})
		}
		c.JSON(http.StatusOK, gin.H{"userId": userID, "role": req.Role})
	}
}
