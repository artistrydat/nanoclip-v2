package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"paperclip-go/middleware"
	"paperclip-go/models"
)

func ActivityRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	rg.GET("", listActivity(db))
}

func listActivity(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var logs []models.ActivityLog
		q := db.Where("company_id = ?", c.Param("companyId")).
			Order("created_at desc").Limit(100)
		if entityType := c.Query("entityType"); entityType != "" {
			q = q.Where("entity_type = ?", entityType)
		}
		if entityID := c.Query("entityId"); entityID != "" {
			q = q.Where("entity_id = ?", entityID)
		}
		if agentID := c.Query("agentId"); agentID != "" {
			q = q.Where("agent_id = ?", agentID)
		}
		q.Find(&logs)
		c.JSON(http.StatusOK, logs)
	}
}

func logActivity(db *gorm.DB, companyID string, actor *middleware.ActorInfo, action, entityType, entityID string, details models.JSON) {
	actorType := "system"
	actorID := "system"
	var agentID *string

	if actor != nil {
		switch actor.Type {
		case "user":
			actorType = "user"
			actorID = actor.UserID
		case "agent":
			actorType = "agent"
			actorID = actor.AgentID
			agentID = &actor.AgentID
		}
	}

	entry := models.ActivityLog{
		ID:         uuid.NewString(),
		CompanyID:  companyID,
		ActorType:  actorType,
		ActorID:    actorID,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		AgentID:    agentID,
		Details:    details,
		CreatedAt:  time.Now(),
	}
	db.Create(&entry)
}
