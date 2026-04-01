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

func ProjectRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	rg.GET("", listProjects(db))
	rg.POST("", createProject(db))
	rg.GET("/:projectId", getProject(db))
	rg.PATCH("/:projectId", updateProject(db))
	rg.DELETE("/:projectId", deleteProject(db))
}

func listProjects(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var projects []models.Project
		q := db.Where("company_id = ? AND archived_at IS NULL", c.Param("companyId")).
			Order("created_at desc")
		if status := c.Query("status"); status != "" {
			q = q.Where("status = ?", status)
		}
		q.Find(&projects)
		c.JSON(http.StatusOK, projects)
	}
}

func getProject(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var project models.Project
		if err := db.First(&project, "id = ? AND company_id = ?",
			c.Param("projectId"), c.Param("companyId")).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}
		c.JSON(http.StatusOK, project)
	}
}

type createProjectRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
	GoalID      *string `json:"goalId"`
	LeadAgentID *string `json:"leadAgentId"`
	TargetDate  *string `json:"targetDate"`
	Color       *string `json:"color"`
	Status      *string `json:"status"`
}

func createProject(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createProjectRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		status := "backlog"
		if req.Status != nil {
			status = *req.Status
		}
		project := models.Project{
			ID:          uuid.NewString(),
			CompanyID:   c.Param("companyId"),
			GoalID:      req.GoalID,
			Name:        req.Name,
			Description: req.Description,
			Status:      status,
			LeadAgentID: req.LeadAgentID,
			TargetDate:  req.TargetDate,
			Color:       req.Color,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := db.Create(&project).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		actor := middleware.GetActor(c)
		logActivity(db, project.CompanyID, actor, "created", "project", project.ID, nil)
		c.JSON(http.StatusCreated, project)
	}
}

func updateProject(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var project models.Project
		if err := db.First(&project, "id = ? AND company_id = ?",
			c.Param("projectId"), c.Param("companyId")).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}
		var req struct {
			Name        *string `json:"name"`
			Description *string `json:"description"`
			Status      *string `json:"status"`
			GoalID      *string `json:"goalId"`
			LeadAgentID *string `json:"leadAgentId"`
			Color       *string `json:"color"`
			TargetDate  *string `json:"targetDate"`
		}
		c.ShouldBindJSON(&req)

		updates := map[string]interface{}{"updated_at": time.Now()}
		if req.Name != nil {
			updates["name"] = *req.Name
		}
		if req.Description != nil {
			updates["description"] = req.Description
		}
		if req.Status != nil {
			updates["status"] = *req.Status
		}
		if req.GoalID != nil {
			updates["goal_id"] = req.GoalID
		}
		if req.LeadAgentID != nil {
			updates["lead_agent_id"] = req.LeadAgentID
		}
		if req.Color != nil {
			updates["color"] = req.Color
		}
		if req.TargetDate != nil {
			updates["target_date"] = req.TargetDate
		}
		db.Model(&project).Updates(updates)
		db.First(&project, "id = ?", project.ID)
		c.JSON(http.StatusOK, project)
	}
}

func deleteProject(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		now := time.Now()
		db.Model(&models.Project{}).Where("id = ? AND company_id = ?",
			c.Param("projectId"), c.Param("companyId")).
			Updates(map[string]interface{}{"archived_at": now, "updated_at": now})
		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}
