package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"paperclip-go/models"
)

func RoutineRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	rg.GET("", listRoutines(db))
	rg.POST("", createRoutine(db))
	rg.GET("/:routineId", getRoutine(db))
	rg.PATCH("/:routineId", updateRoutine(db))
	rg.DELETE("/:routineId", deleteRoutine(db))
	rg.GET("/:routineId/triggers", listTriggers(db))
	rg.POST("/:routineId/triggers", createTrigger(db))
}

func listRoutines(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var routines []models.Routine
		db.Where("company_id = ?", c.Param("companyId")).
			Order("created_at desc").Find(&routines)
		c.JSON(http.StatusOK, routines)
	}
}

func getRoutine(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var routine models.Routine
		if err := db.First(&routine, "id = ? AND company_id = ?",
			c.Param("routineId"), c.Param("companyId")).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "routine not found"})
			return
		}
		c.JSON(http.StatusOK, routine)
	}
}

func createRoutine(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Title           string  `json:"title" binding:"required"`
			Description     *string `json:"description"`
			ProjectID       string  `json:"projectId" binding:"required"`
			GoalID          *string `json:"goalId"`
			AssigneeAgentID string  `json:"assigneeAgentId" binding:"required"`
			Priority        *string `json:"priority"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		priority := "medium"
		if req.Priority != nil {
			priority = *req.Priority
		}
		routine := models.Routine{
			ID:              uuid.NewString(),
			CompanyID:       c.Param("companyId"),
			ProjectID:       req.ProjectID,
			GoalID:          req.GoalID,
			Title:           req.Title,
			Description:     req.Description,
			AssigneeAgentID: req.AssigneeAgentID,
			Priority:        priority,
			Status:          "active",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		db.Create(&routine)
		c.JSON(http.StatusCreated, routine)
	}
}

func updateRoutine(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var routine models.Routine
		if err := db.First(&routine, "id = ? AND company_id = ?",
			c.Param("routineId"), c.Param("companyId")).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "routine not found"})
			return
		}
		var req struct {
			Title           *string `json:"title"`
			Description     *string `json:"description"`
			Status          *string `json:"status"`
			AssigneeAgentID *string `json:"assigneeAgentId"`
			Priority        *string `json:"priority"`
		}
		c.ShouldBindJSON(&req)
		updates := map[string]interface{}{"updated_at": time.Now()}
		if req.Title != nil {
			updates["title"] = *req.Title
		}
		if req.Description != nil {
			updates["description"] = req.Description
		}
		if req.Status != nil {
			updates["status"] = *req.Status
		}
		if req.AssigneeAgentID != nil {
			updates["assignee_agent_id"] = *req.AssigneeAgentID
		}
		if req.Priority != nil {
			updates["priority"] = *req.Priority
		}
		db.Model(&routine).Updates(updates)
		db.First(&routine, "id = ?", routine.ID)
		c.JSON(http.StatusOK, routine)
	}
}

func deleteRoutine(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		db.Where("id = ? AND company_id = ?",
			c.Param("routineId"), c.Param("companyId")).Delete(&models.Routine{})
		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

func listTriggers(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var triggers []models.RoutineTrigger
		db.Where("routine_id = ? AND company_id = ?",
			c.Param("routineId"), c.Param("companyId")).Find(&triggers)
		c.JSON(http.StatusOK, triggers)
	}
}

func createTrigger(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Kind           string  `json:"kind" binding:"required"`
			Label          *string `json:"label"`
			CronExpression *string `json:"cronExpression"`
			Timezone       *string `json:"timezone"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		trigger := models.RoutineTrigger{
			ID:             uuid.NewString(),
			CompanyID:      c.Param("companyId"),
			RoutineID:      c.Param("routineId"),
			Kind:           req.Kind,
			Label:          req.Label,
			Enabled:        true,
			CronExpression: req.CronExpression,
			Timezone:       req.Timezone,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		db.Create(&trigger)
		c.JSON(http.StatusCreated, trigger)
	}
}
