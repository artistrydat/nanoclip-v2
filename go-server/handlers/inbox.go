package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"paperclip-go/models"
)

func InboxRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	rg.GET("", listInboxItems(db))
	rg.POST("", createInboxItem(db))
	rg.PATCH("/:itemId", updateInboxItem(db))
	rg.DELETE("/:itemId", deleteInboxItem(db))
}

func listInboxItems(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		companyID := c.Param("companyId")
		q := db.Where("company_id = ?", companyID).Order("created_at desc").Limit(100)
		if status := c.Query("status"); status != "" {
			q = q.Where("status = ?", status)
		} else {
			q = q.Where("status != 'archived'")
		}
		if agentID := c.Query("agentId"); agentID != "" {
			q = q.Where("agent_id = ?", agentID)
		}
		var items []models.InboxItem
		q.Find(&items)
		c.JSON(http.StatusOK, items)
	}
}

func createInboxItem(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Kind    string      `json:"kind" binding:"required"`
			Summary string      `json:"summary" binding:"required"`
			AgentID *string     `json:"agentId"`
			IssueID *string     `json:"issueId"`
			RunID   *string     `json:"runId"`
			Payload models.JSON `json:"payload"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		item := models.InboxItem{
			ID:        uuid.NewString(),
			CompanyID: c.Param("companyId"),
			Kind:      req.Kind,
			Summary:   req.Summary,
			AgentID:   req.AgentID,
			IssueID:   req.IssueID,
			RunID:     req.RunID,
			Payload:   req.Payload,
			Status:    "unread",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		db.Create(&item)
		c.JSON(http.StatusCreated, item)
	}
}

func updateInboxItem(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Status *string `json:"status"`
		}
		c.ShouldBindJSON(&req)
		updates := map[string]interface{}{"updated_at": time.Now()}
		if req.Status != nil {
			updates["status"] = *req.Status
			if *req.Status == "read" {
				updates["read_at"] = time.Now()
			}
		}
		db.Model(&models.InboxItem{}).
			Where("id = ? AND company_id = ?", c.Param("itemId"), c.Param("companyId")).
			Updates(updates)
		var item models.InboxItem
		db.First(&item, "id = ?", c.Param("itemId"))
		c.JSON(http.StatusOK, item)
	}
}

func deleteInboxItem(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		db.Model(&models.InboxItem{}).
			Where("id = ? AND company_id = ?", c.Param("itemId"), c.Param("companyId")).
			Updates(map[string]interface{}{"status": "archived", "updated_at": time.Now()})
		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}
