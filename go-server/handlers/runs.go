package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"paperclip-go/models"
	"paperclip-go/services"
)

func RunRoutes(rg *gin.RouterGroup, db *gorm.DB, hb *services.HeartbeatService) {
	rg.GET("", listRuns(db))
	rg.GET("/:runId", getRun(db))
	rg.POST("/trigger", triggerRun(db, hb))
}

func listRuns(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var runs []models.HeartbeatRun
		q := db.Where("company_id = ?", c.Param("companyId")).
			Order("created_at desc").Limit(100)
		if agentID := c.Query("agentId"); agentID != "" {
			q = q.Where("agent_id = ?", agentID)
		}
		if status := c.Query("status"); status != "" {
			q = q.Where("status = ?", status)
		}
		q.Find(&runs)
		c.JSON(http.StatusOK, runs)
	}
}

func getRun(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var run models.HeartbeatRun
		if err := db.First(&run, "id = ? AND company_id = ?",
			c.Param("runId"), c.Param("companyId")).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
			return
		}
		c.JSON(http.StatusOK, run)
	}
}

func triggerRun(db *gorm.DB, hb *services.HeartbeatService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			AgentID string  `json:"agentId" binding:"required"`
			IssueID *string `json:"issueId"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var issue *models.Issue
		if req.IssueID != nil {
			var i models.Issue
			if err := db.First(&i, "id = ?", *req.IssueID).Error; err == nil {
				issue = &i
			}
		}

		run, err := hb.TriggerRun(req.AgentID, c.Param("companyId"), issue)
		if err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, run)
	}
}
