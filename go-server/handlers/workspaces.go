package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"paperclip-go/models"
)

// CompanyWorkspaceRoutes registers /companies/:companyId/execution-workspaces
func CompanyWorkspaceRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	rg.GET("", listCompanyWorkspaces(db))
	rg.POST("", createWorkspace(db))
}

// GlobalWorkspaceRoutes registers /execution-workspaces/:workspaceId
func GlobalWorkspaceRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	rg.GET("/:workspaceId", getWorkspace(db))
	rg.PATCH("/:workspaceId", updateWorkspace(db))
	rg.GET("/:workspaceId/close-readiness", getWorkspaceCloseReadiness(db))
	rg.POST("/:workspaceId/close", closeWorkspace(db))
	rg.GET("/:workspaceId/workspace-operations", listWorkspaceOps(db))
	rg.POST("/:workspaceId/workspace-operations", createWorkspaceOp(db))
}

func listCompanyWorkspaces(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var workspaces []models.ExecutionWorkspace
		q := db.Where("company_id = ?", c.Param("companyId")).Order("created_at desc").Limit(50)
		if status := c.Query("status"); status != "" {
			q = q.Where("status = ?", status)
		}
		if agentID := c.Query("agentId"); agentID != "" {
			q = q.Where("agent_id = ?", agentID)
		}
		q.Find(&workspaces)
		c.JSON(http.StatusOK, workspaces)
	}
}

func createWorkspace(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			AgentID        string      `json:"agentId" binding:"required"`
			HeartbeatRunID *string     `json:"heartbeatRunId"`
			IssueID        *string     `json:"issueId"`
			Kind           *string     `json:"kind"`
			Config         models.JSON `json:"config"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		kind := "execution"
		if req.Kind != nil {
			kind = *req.Kind
		}
		workspace := models.ExecutionWorkspace{
			ID:             uuid.NewString(),
			CompanyID:      c.Param("companyId"),
			AgentID:        req.AgentID,
			HeartbeatRunID: req.HeartbeatRunID,
			IssueID:        req.IssueID,
			Kind:           kind,
			Status:         "open",
			Config:         req.Config,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		db.Create(&workspace)
		c.JSON(http.StatusCreated, workspace)
	}
}

func getWorkspace(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var workspace models.ExecutionWorkspace
		if err := db.First(&workspace, "id = ?", c.Param("workspaceId")).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "workspace not found"})
			return
		}
		c.JSON(http.StatusOK, workspace)
	}
}

func updateWorkspace(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Status   *string     `json:"status"`
			Metadata models.JSON `json:"metadata"`
		}
		c.ShouldBindJSON(&req)
		updates := map[string]interface{}{"updated_at": time.Now()}
		if req.Status != nil {
			updates["status"] = *req.Status
			if *req.Status == "closed" {
				updates["closed_at"] = time.Now()
			}
		}
		if req.Metadata != nil {
			updates["metadata"] = req.Metadata
		}
		db.Model(&models.ExecutionWorkspace{}).Where("id = ?", c.Param("workspaceId")).Updates(updates)
		var workspace models.ExecutionWorkspace
		db.First(&workspace, "id = ?", c.Param("workspaceId"))
		c.JSON(http.StatusOK, workspace)
	}
}

func getWorkspaceCloseReadiness(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var workspace models.ExecutionWorkspace
		if err := db.First(&workspace, "id = ?", c.Param("workspaceId")).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "workspace not found"})
			return
		}
		// Check for pending operations
		var pendingOps int64
		db.Model(&models.WorkspaceOperation{}).
			Where("execution_workspace_id = ? AND status IN ('pending','running')", workspace.ID).
			Count(&pendingOps)
		c.JSON(http.StatusOK, gin.H{
			"ready":       pendingOps == 0,
			"pendingOps":  pendingOps,
			"workspaceId": workspace.ID,
		})
	}
}

func closeWorkspace(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		now := time.Now()
		db.Model(&models.ExecutionWorkspace{}).Where("id = ?", c.Param("workspaceId")).
			Updates(map[string]interface{}{
				"status":     "closed",
				"closed_at":  now,
				"updated_at": now,
			})
		var workspace models.ExecutionWorkspace
		db.First(&workspace, "id = ?", c.Param("workspaceId"))
		c.JSON(http.StatusOK, workspace)
	}
}

func listWorkspaceOps(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var ops []models.WorkspaceOperation
		db.Where("execution_workspace_id = ?", c.Param("workspaceId")).
			Order("created_at asc").Find(&ops)
		c.JSON(http.StatusOK, ops)
	}
}

func createWorkspaceOp(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Kind    string      `json:"kind" binding:"required"`
			Payload models.JSON `json:"payload"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get workspace for companyId/agentId
		var workspace models.ExecutionWorkspace
		if err := db.First(&workspace, "id = ?", c.Param("workspaceId")).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "workspace not found"})
			return
		}

		op := models.WorkspaceOperation{
			ID:                   uuid.NewString(),
			ExecutionWorkspaceID: workspace.ID,
			CompanyID:            workspace.CompanyID,
			AgentID:              workspace.AgentID,
			Kind:                 req.Kind,
			Status:               "pending",
			Payload:              req.Payload,
			CreatedAt:            time.Now(),
			UpdatedAt:            time.Now(),
		}
		db.Create(&op)
		c.JSON(http.StatusCreated, op)
	}
}
