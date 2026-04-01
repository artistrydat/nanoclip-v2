package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"paperclip-go/middleware"
	"paperclip-go/models"
	"paperclip-go/ws"
)

func ApprovalRoutes(rg *gin.RouterGroup, db *gorm.DB, hub *ws.Hub) {
	rg.GET("", listApprovals(db))
	rg.POST("", createApproval(db))
	rg.GET("/:approvalId", getApproval(db))
	rg.POST("/:approvalId/approve", approveApproval(db, hub))
	rg.POST("/:approvalId/reject", rejectApproval(db, hub))
}

func listApprovals(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var approvals []models.Approval
		q := db.Where("company_id = ?", c.Param("companyId")).Order("created_at desc")
		if status := c.Query("status"); status != "" {
			q = q.Where("status = ?", status)
		}
		q.Find(&approvals)
		c.JSON(http.StatusOK, approvals)
	}
}

func getApproval(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var approval models.Approval
		if err := db.First(&approval, "id = ? AND company_id = ?",
			c.Param("approvalId"), c.Param("companyId")).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "approval not found"})
			return
		}
		c.JSON(http.StatusOK, approval)
	}
}

func createApproval(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Type    string      `json:"type" binding:"required"`
			Payload models.JSON `json:"payload"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		actor := middleware.GetActor(c)
		approval := models.Approval{
			ID:        uuid.NewString(),
			CompanyID: c.Param("companyId"),
			Type:      req.Type,
			Status:    "pending",
			Payload:   req.Payload,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if actor != nil {
			if actor.Type == "user" {
				approval.RequestedByUserID = &actor.UserID
			} else if actor.Type == "agent" {
				approval.RequestedByAgentID = &actor.AgentID
			}
		}
		db.Create(&approval)
		c.JSON(http.StatusCreated, approval)
	}
}

func approveApproval(db *gorm.DB, hub *ws.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct{ Note *string `json:"note"` }
		c.ShouldBindJSON(&req)
		actor := middleware.GetActor(c)
		now := time.Now()

		updates := map[string]interface{}{
			"status":     "approved",
			"decided_at": now,
			"updated_at": now,
		}
		if actor != nil && actor.UserID != "" {
			updates["decided_by_user_id"] = actor.UserID
		}
		if req.Note != nil {
			updates["decision_note"] = *req.Note
		}
		db.Model(&models.Approval{}).Where("id = ? AND company_id = ?",
			c.Param("approvalId"), c.Param("companyId")).Updates(updates)

		var approval models.Approval
		db.First(&approval, "id = ?", c.Param("approvalId"))
		hub.Publish(ws.LiveEvent{Type: "approval.updated", Payload: approval})
		c.JSON(http.StatusOK, approval)
	}
}

func rejectApproval(db *gorm.DB, hub *ws.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct{ Note *string `json:"note"` }
		c.ShouldBindJSON(&req)
		actor := middleware.GetActor(c)
		now := time.Now()

		updates := map[string]interface{}{
			"status":     "rejected",
			"decided_at": now,
			"updated_at": now,
		}
		if actor != nil && actor.UserID != "" {
			updates["decided_by_user_id"] = actor.UserID
		}
		if req.Note != nil {
			updates["decision_note"] = *req.Note
		}
		db.Model(&models.Approval{}).Where("id = ? AND company_id = ?",
			c.Param("approvalId"), c.Param("companyId")).Updates(updates)

		var approval models.Approval
		db.First(&approval, "id = ?", c.Param("approvalId"))
		hub.Publish(ws.LiveEvent{Type: "approval.updated", Payload: approval})
		c.JSON(http.StatusOK, approval)
	}
}
