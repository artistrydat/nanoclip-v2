package handlers

import (
        "net/http"
        "time"

        "github.com/gin-gonic/gin"
        "github.com/google/uuid"
        "gorm.io/gorm"
        "paperclip-go/middleware"
        "paperclip-go/models"
        mw "paperclip-go/middleware"
)

func AgentRoutes(rg *gin.RouterGroup, db *gorm.DB) {
        rg.GET("", listAgents(db))
        rg.POST("", createAgent(db))
        rg.GET("/:agentId", getAgent(db))
        rg.PATCH("/:agentId", updateAgent(db))
        rg.DELETE("/:agentId", deleteAgent(db))
        rg.POST("/:agentId/pause", pauseAgent(db))
        rg.POST("/:agentId/resume", resumeAgent(db))
        rg.POST("/:agentId/wakeup", wakeupAgent(db))
        rg.GET("/:agentId/runs", listAgentRuns(db))
        rg.GET("/:agentId/jwt", getAgentJWT(db))
}

func AgentHireRoutes(rg *gin.RouterGroup, db *gorm.DB) {
        rg.POST("", hireAgent(db))
}

func listAgents(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                companyID := c.Param("companyId")
                var agents []models.Agent
                db.Where("company_id = ?", companyID).Order("created_at asc").Find(&agents)
                c.JSON(http.StatusOK, wrapAgents(agents))
        }
}

func getAgent(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var agent models.Agent
                if err := db.First(&agent, "id = ? AND company_id = ?",
                        c.Param("agentId"), c.Param("companyId")).Error; err != nil {
                        c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
                        return
                }
                c.JSON(http.StatusOK, wrapAgent(&agent))
        }
}

type createAgentRequest struct {
        Name          string       `json:"name" binding:"required"`
        Role          *string      `json:"role"`
        Title         *string      `json:"title"`
        Icon          *string      `json:"icon"`
        ReportsTo     *string      `json:"reportsTo"`
        Capabilities  *string      `json:"capabilities"`
        AdapterType   *string      `json:"adapterType"`
        AdapterConfig models.JSON  `json:"adapterConfig"`
}

func createAgent(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var req createAgentRequest
                if err := c.ShouldBindJSON(&req); err != nil {
                        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                        return
                }
                adapterType := "process"
                if req.AdapterType != nil {
                        adapterType = *req.AdapterType
                }
                role := "general"
                if req.Role != nil {
                        role = *req.Role
                }
                agent := models.Agent{
                        ID:            uuid.NewString(),
                        CompanyID:     c.Param("companyId"),
                        Name:          req.Name,
                        Role:          role,
                        Title:         req.Title,
                        Icon:          req.Icon,
                        Status:        "idle",
                        ReportsTo:     req.ReportsTo,
                        Capabilities:  req.Capabilities,
                        AdapterType:   adapterType,
                        AdapterConfig: req.AdapterConfig,
                        Permissions:   models.JSON{},
                        RuntimeConfig: models.JSON{},
                        CreatedAt:     time.Now(),
                        UpdatedAt:     time.Now(),
                }
                if err := db.Create(&agent).Error; err != nil {
                        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                        return
                }
                actor := middleware.GetActor(c)
                logActivity(db, agent.CompanyID, actor, "created", "agent", agent.ID, nil)
                c.JSON(http.StatusCreated, wrapAgent(&agent))
        }
}

type updateAgentRequest struct {
        Name               *string     `json:"name"`
        Role               *string     `json:"role"`
        Title              *string     `json:"title"`
        Icon               *string     `json:"icon"`
        Status             *string     `json:"status"`
        ReportsTo          *string     `json:"reportsTo"`
        Capabilities       *string     `json:"capabilities"`
        AdapterType        *string     `json:"adapterType"`
        AdapterConfig      models.JSON `json:"adapterConfig"`
        RuntimeConfig      models.JSON `json:"runtimeConfig"`
        BudgetMonthlyCents *int        `json:"budgetMonthlyCents"`
}

func updateAgent(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var agent models.Agent
                if err := db.First(&agent, "id = ? AND company_id = ?",
                        c.Param("agentId"), c.Param("companyId")).Error; err != nil {
                        c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
                        return
                }
                var req updateAgentRequest
                c.ShouldBindJSON(&req)

                updates := map[string]interface{}{"updated_at": time.Now()}
                if req.Name != nil {
                        updates["name"] = *req.Name
                }
                if req.Role != nil {
                        updates["role"] = *req.Role
                }
                if req.Title != nil {
                        updates["title"] = req.Title
                }
                if req.Icon != nil {
                        updates["icon"] = req.Icon
                }
                if req.Status != nil {
                        updates["status"] = *req.Status
                }
                if req.ReportsTo != nil {
                        updates["reports_to"] = req.ReportsTo
                }
                if req.Capabilities != nil {
                        updates["capabilities"] = req.Capabilities
                }
                if req.AdapterType != nil {
                        updates["adapter_type"] = *req.AdapterType
                }
                if req.AdapterConfig != nil {
                        // Merge into existing config — preserve fields not touched by this update
                        merged := models.JSON{}
                        for k, v := range agent.AdapterConfig {
                                merged[k] = v
                        }
                        for k, v := range req.AdapterConfig {
                                merged[k] = v
                        }
                        updates["adapter_config"] = merged
                }
                if req.RuntimeConfig != nil {
                        // Merge into existing runtime config
                        mergedRT := models.JSON{}
                        for k, v := range agent.RuntimeConfig {
                                mergedRT[k] = v
                        }
                        for k, v := range req.RuntimeConfig {
                                mergedRT[k] = v
                        }
                        updates["runtime_config"] = mergedRT
                }
                if req.BudgetMonthlyCents != nil {
                        updates["budget_monthly_cents"] = *req.BudgetMonthlyCents
                }
                db.Model(&agent).Updates(updates)
                db.First(&agent, "id = ?", agent.ID)
                actor := mw.GetActor(c)
                logActivity(db, agent.CompanyID, actor, "updated", "agent", agent.ID, nil)
                c.JSON(http.StatusOK, wrapAgent(&agent))
        }
}

func deleteAgent(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                if err := db.Where("id = ? AND company_id = ?",
                        c.Param("agentId"), c.Param("companyId")).Delete(&models.Agent{}).Error; err != nil {
                        c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
                        return
                }
                c.JSON(http.StatusOK, gin.H{"success": true})
        }
}

func pauseAgent(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var req struct{ Reason *string `json:"reason"` }
                c.ShouldBindJSON(&req)
                now := time.Now()
                updates := map[string]interface{}{
                        "status":     "paused",
                        "paused_at":  now,
                        "updated_at": now,
                }
                if req.Reason != nil {
                        updates["pause_reason"] = *req.Reason
                }
                db.Model(&models.Agent{}).Where("id = ? AND company_id = ?",
                        c.Param("agentId"), c.Param("companyId")).Updates(updates)
                var agent models.Agent
                db.First(&agent, "id = ?", c.Param("agentId"))
                c.JSON(http.StatusOK, agent)
        }
}

func resumeAgent(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                db.Model(&models.Agent{}).Where("id = ? AND company_id = ?",
                        c.Param("agentId"), c.Param("companyId")).Updates(map[string]interface{}{
                        "status":       "idle",
                        "paused_at":    nil,
                        "pause_reason": nil,
                        "updated_at":   time.Now(),
                })
                var agent models.Agent
                db.First(&agent, "id = ?", c.Param("agentId"))
                c.JSON(http.StatusOK, agent)
        }
}

func wakeupAgent(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var req struct{ Reason *string `json:"reason"` }
                c.ShouldBindJSON(&req)
                wakeup := models.AgentWakeupRequest{
                        ID:        uuid.NewString(),
                        AgentID:   c.Param("agentId"),
                        CompanyID: c.Param("companyId"),
                        Reason:    req.Reason,
                        CreatedAt: time.Now(),
                }
                db.Create(&wakeup)
                c.JSON(http.StatusOK, wakeup)
        }
}

func listAgentRuns(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var runs []models.HeartbeatRun
                db.Where("agent_id = ? AND company_id = ?",
                        c.Param("agentId"), c.Param("companyId")).
                        Order("created_at desc").Limit(50).Find(&runs)
                c.JSON(http.StatusOK, runs)
        }
}

func hireAgent(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var req createAgentRequest
                if err := c.ShouldBindJSON(&req); err != nil {
                        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                        return
                }
                adapterType := "process"
                if req.AdapterType != nil {
                        adapterType = *req.AdapterType
                }
                role := "general"
                if req.Role != nil {
                        role = *req.Role
                }
                agent := models.Agent{
                        ID:            uuid.NewString(),
                        CompanyID:     c.Param("companyId"),
                        Name:          req.Name,
                        Role:          role,
                        Title:         req.Title,
                        Icon:          req.Icon,
                        Status:        "idle",
                        ReportsTo:     req.ReportsTo,
                        Capabilities:  req.Capabilities,
                        AdapterType:   adapterType,
                        AdapterConfig: req.AdapterConfig,
                        Permissions:   models.JSON{},
                        RuntimeConfig: models.JSON{},
                        CreatedAt:     time.Now(),
                        UpdatedAt:     time.Now(),
                }
                if err := db.Create(&agent).Error; err != nil {
                        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                        return
                }
                actor := middleware.GetActor(c)
                logActivity(db, agent.CompanyID, actor, "created", "agent", agent.ID, nil)
                c.JSON(http.StatusCreated, gin.H{
                        "agent":    wrapAgent(&agent),
                        "approval": nil,
                })
        }
}

func getAgentJWT(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var agent models.Agent
                if err := db.First(&agent, "id = ? AND company_id = ?",
                        c.Param("agentId"), c.Param("companyId")).Error; err != nil {
                        c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
                        return
                }
                token, err := mw.IssueAgentJWT(agent.ID, agent.CompanyID)
                if err != nil {
                        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue JWT"})
                        return
                }
                c.JSON(http.StatusOK, gin.H{"token": token, "agentId": agent.ID, "companyId": agent.CompanyID})
        }
}
