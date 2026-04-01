package handlers

import (
        "net/http"
        "time"

        "github.com/gin-gonic/gin"
        "github.com/google/uuid"
        "gorm.io/gorm"
        "paperclip-go/models"
)

// GlobalGoalRoutes handles /api/goals/:goalId (no company scope, companyId via query param)
func GlobalGoalRoutes(rg *gin.RouterGroup, db *gorm.DB) {
        rg.GET("/:goalId", getGlobalGoal(db))
        rg.PATCH("/:goalId", updateGlobalGoal(db))
}

func getGlobalGoal(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var goal models.Goal
                q := db.Where("id = ?", c.Param("goalId"))
                if cid := c.Query("companyId"); cid != "" {
                        q = q.Where("company_id = ?", cid)
                }
                if err := q.First(&goal).Error; err != nil {
                        c.JSON(http.StatusNotFound, gin.H{"error": "goal not found"})
                        return
                }
                c.JSON(http.StatusOK, goal)
        }
}

func updateGlobalGoal(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var goal models.Goal
                q := db.Where("id = ?", c.Param("goalId"))
                if cid := c.Query("companyId"); cid != "" {
                        q = q.Where("company_id = ?", cid)
                }
                if err := q.First(&goal).Error; err != nil {
                        c.JSON(http.StatusNotFound, gin.H{"error": "goal not found"})
                        return
                }
                var req struct {
                        Title        *string `json:"title"`
                        Description  *string `json:"description"`
                        Level        *string `json:"level"`
                        Status       *string `json:"status"`
                        OwnerAgentID *string `json:"ownerAgentId"`
                }
                c.ShouldBindJSON(&req)
                updates := map[string]interface{}{"updated_at": time.Now()}
                if req.Title != nil {
                        updates["title"] = *req.Title
                }
                if req.Description != nil {
                        updates["description"] = req.Description
                }
                if req.Level != nil {
                        updates["level"] = *req.Level
                }
                if req.Status != nil {
                        updates["status"] = *req.Status
                }
                if req.OwnerAgentID != nil {
                        updates["owner_agent_id"] = req.OwnerAgentID
                }
                db.Model(&goal).Updates(updates)
                db.First(&goal, "id = ?", goal.ID)
                c.JSON(http.StatusOK, goal)
        }
}

func GoalRoutes(rg *gin.RouterGroup, db *gorm.DB) {
        rg.GET("", listGoals(db))
        rg.POST("", createGoal(db))
        rg.GET("/:goalId", getGoal(db))
        rg.PATCH("/:goalId", updateGoal(db))
        rg.DELETE("/:goalId", deleteGoal(db))
}

func listGoals(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var goals []models.Goal
                db.Where("company_id = ?", c.Param("companyId")).
                        Order("created_at asc").Find(&goals)
                c.JSON(http.StatusOK, goals)
        }
}

func getGoal(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var goal models.Goal
                if err := db.First(&goal, "id = ? AND company_id = ?",
                        c.Param("goalId"), c.Param("companyId")).Error; err != nil {
                        c.JSON(http.StatusNotFound, gin.H{"error": "goal not found"})
                        return
                }
                c.JSON(http.StatusOK, goal)
        }
}

func createGoal(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var req struct {
                        Title        string  `json:"title" binding:"required"`
                        Description  *string `json:"description"`
                        Level        *string `json:"level"`
                        Status       *string `json:"status"`
                        ParentID     *string `json:"parentId"`
                        OwnerAgentID *string `json:"ownerAgentId"`
                }
                if err := c.ShouldBindJSON(&req); err != nil {
                        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                        return
                }
                level := "task"
                if req.Level != nil {
                        level = *req.Level
                }
                status := "planned"
                if req.Status != nil {
                        status = *req.Status
                }
                goal := models.Goal{
                        ID:           uuid.NewString(),
                        CompanyID:    c.Param("companyId"),
                        Title:        req.Title,
                        Description:  req.Description,
                        Level:        level,
                        Status:       status,
                        ParentID:     req.ParentID,
                        OwnerAgentID: req.OwnerAgentID,
                        CreatedAt:    time.Now(),
                        UpdatedAt:    time.Now(),
                }
                db.Create(&goal)
                c.JSON(http.StatusCreated, goal)
        }
}

func updateGoal(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var goal models.Goal
                if err := db.First(&goal, "id = ? AND company_id = ?",
                        c.Param("goalId"), c.Param("companyId")).Error; err != nil {
                        c.JSON(http.StatusNotFound, gin.H{"error": "goal not found"})
                        return
                }
                var req struct {
                        Title        *string `json:"title"`
                        Description  *string `json:"description"`
                        Level        *string `json:"level"`
                        Status       *string `json:"status"`
                        OwnerAgentID *string `json:"ownerAgentId"`
                }
                c.ShouldBindJSON(&req)
                updates := map[string]interface{}{"updated_at": time.Now()}
                if req.Title != nil {
                        updates["title"] = *req.Title
                }
                if req.Description != nil {
                        updates["description"] = req.Description
                }
                if req.Level != nil {
                        updates["level"] = *req.Level
                }
                if req.Status != nil {
                        updates["status"] = *req.Status
                }
                if req.OwnerAgentID != nil {
                        updates["owner_agent_id"] = req.OwnerAgentID
                }
                db.Model(&goal).Updates(updates)
                db.First(&goal, "id = ?", goal.ID)
                c.JSON(http.StatusOK, goal)
        }
}

func deleteGoal(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                db.Where("id = ? AND company_id = ?",
                        c.Param("goalId"), c.Param("companyId")).Delete(&models.Goal{})
                c.JSON(http.StatusOK, gin.H{"success": true})
        }
}
