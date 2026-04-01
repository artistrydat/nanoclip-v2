package handlers

import (
        "fmt"
        "net/http"
        "regexp"
        "strings"
        "time"

        "github.com/gin-gonic/gin"
        "gorm.io/gorm"
        "paperclip-go/models"
        mw "paperclip-go/middleware"
)

// GlobalAgentRoutes handles /api/agents/:agentId (no company scope, uses query param)
func GlobalAgentRoutes(rg *gin.RouterGroup, db *gorm.DB) {
        rg.GET("/:agentId", getGlobalAgent(db))
        rg.PATCH("/:agentId", updateGlobalAgent(db))
        rg.GET("/:agentId/runtime-state", getAgentRuntimeState(db))
        rg.POST("/:agentId/runtime-state/reset-session", resetAgentSession(db))
        rg.GET("/:agentId/skills", getAgentSkills(db))
        rg.GET("/:agentId/keys", getAgentKeys(db))
        rg.POST("/:agentId/keys", CreateAgentKey(db))
        rg.DELETE("/:agentId/keys/:keyId", RevokeAgentKey(db))
        rg.GET("/:agentId/config-revisions", getAgentConfigRevisions(db))
        rg.POST("/:agentId/pause", pauseGlobalAgent(db))
        rg.POST("/:agentId/resume", resumeGlobalAgent(db))
        rg.PATCH("/:agentId/permissions", updateAgentPermissions(db))
        // Instructions bundle
        rg.GET("/:agentId/instructions-bundle", GetInstructionsBundle(db))
        rg.PATCH("/:agentId/instructions-bundle", UpdateInstructionsBundle(db))
        rg.GET("/:agentId/instructions-bundle/file", GetInstructionsFile(db))
        rg.PUT("/:agentId/instructions-bundle/file", SaveInstructionsFile(db))
        rg.DELETE("/:agentId/instructions-bundle/file", DeleteInstructionsFile(db))
}

var uuidRe = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var shortHexRe = regexp.MustCompile(`^[0-9a-f]{8}$`)

func isUUIDLike(s string) bool {
        return uuidRe.MatchString(s)
}

func normalizeAgentKey(key string) string {
        re := regexp.MustCompile(`[^a-z0-9-]`)
        return re.ReplaceAllString(strings.ToLower(key), "-")
}

// resolveAgentByParam resolves an agent by full UUID or derived URL-key slug.
func resolveAgentByParam(db *gorm.DB, param, companyID string) (*models.Agent, int, error) {
        q := db
        if companyID != "" {
                q = q.Where("company_id = ?", companyID)
        }

        if isUUIDLike(param) {
                var agent models.Agent
                if err := q.Where("id = ?", param).First(&agent).Error; err != nil {
                        return nil, http.StatusNotFound, fmt.Errorf("agent not found")
                }
                return &agent, http.StatusOK, nil
        }

        // Slug format: {name-slug}-{shortId} where shortId = first 8 hex chars of UUID (no dashes).
        // The UUID string always starts with 8 hex chars so: id LIKE 'b8b8ffa6-%'
        parts := strings.Split(param, "-")
        shortID := parts[len(parts)-1]
        if !shortHexRe.MatchString(shortID) {
                return nil, http.StatusNotFound, fmt.Errorf("agent not found")
        }

        var agents []models.Agent
        if err := q.Where("id LIKE ?", shortID+"-%").Find(&agents).Error; err != nil || len(agents) == 0 {
                return nil, http.StatusNotFound, fmt.Errorf("agent not found")
        }

        normalizedParam := normalizeAgentKey(param)
        var matches []models.Agent
        for _, a := range agents {
                if normalizeAgentKey(computeAgentUrlKey(a.Name, a.ID)) == normalizedParam {
                        matches = append(matches, a)
                }
        }

        if len(matches) == 0 {
                return nil, http.StatusNotFound, fmt.Errorf("agent not found")
        }
        if len(matches) > 1 {
                return nil, http.StatusConflict, fmt.Errorf("ambiguous agent identifier")
        }
        return &matches[0], http.StatusOK, nil
}

func getGlobalAgent(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                agentID := c.Param("agentId")
                companyID := c.Query("companyId")

                agent, status, err := resolveAgentByParam(db, agentID, companyID)
                if err != nil {
                        c.JSON(status, gin.H{"error": err.Error()})
                        return
                }
                c.JSON(http.StatusOK, wrapAgent(agent))
        }
}

func updateGlobalAgent(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                agentID := c.Param("agentId")
                companyID := c.Query("companyId")

                agent, status, err := resolveAgentByParam(db, agentID, companyID)
                if err != nil {
                        c.JSON(status, gin.H{"error": err.Error()})
                        return
                }

                var req struct {
                        Name          *string     `json:"name"`
                        Status        *string     `json:"status"`
                        Title         *string     `json:"title"`
                        Icon          *string     `json:"icon"`
                        AdapterType   *string     `json:"adapterType"`
                        AdapterConfig models.JSON `json:"adapterConfig"`
                        RuntimeConfig models.JSON `json:"runtimeConfig"`
                }
                c.ShouldBindJSON(&req)

                updates := map[string]interface{}{"updated_at": time.Now()}
                if req.Name != nil {
                        updates["name"] = *req.Name
                }
                if req.Status != nil {
                        updates["status"] = *req.Status
                }
                if req.Title != nil {
                        updates["title"] = req.Title
                }
                if req.Icon != nil {
                        updates["icon"] = req.Icon
                }
                if req.AdapterType != nil {
                        updates["adapter_type"] = *req.AdapterType
                }
                if req.AdapterConfig != nil {
                        updates["adapter_config"] = req.AdapterConfig
                }
                if req.RuntimeConfig != nil {
                        updates["runtime_config"] = req.RuntimeConfig
                }
                db.Model(agent).Updates(updates)
                db.First(agent, "id = ?", agent.ID)

                actor := mw.GetActor(c)
                logActivity(db, agent.CompanyID, actor, "updated", "agent", agent.ID, nil)
                c.JSON(http.StatusOK, wrapAgent(agent))
        }
}

func getAgentRuntimeState(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                agentID := c.Param("agentId")
                companyID := c.Query("companyId")

                agent, status, err := resolveAgentByParam(db, agentID, companyID)
                if err != nil {
                        c.JSON(status, gin.H{"error": err.Error()})
                        return
                }

                // Fetch the last heartbeat run for timing info
                var lastRun models.HeartbeatRun
                lastRunErr := db.Where("agent_id = ?", agent.ID).
                        Order("created_at desc").First(&lastRun).Error

                var lastRunAt *string
                var currentRunID *string
                if lastRunErr == nil {
                        ts := lastRun.CreatedAt.Format(time.RFC3339)
                        lastRunAt = &ts
                        if lastRun.Status == "running" {
                                currentRunID = &lastRun.ID
                        }
                }

                c.JSON(http.StatusOK, gin.H{
                        "status":                  agent.Status,
                        "lastRunAt":               lastRunAt,
                        "currentRunId":            currentRunID,
                        "totalInputTokens":        0,
                        "totalOutputTokens":       0,
                        "totalCachedInputTokens":  0,
                        "totalCostCents":          0,
                })
        }
}

func resetAgentSession(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                // Stub — session management is handled by the adapter layer
                c.JSON(http.StatusOK, gin.H{"ok": true})
        }
}

// getAgentSkills returns the skills registered for an agent, derived from its adapter config.
func getAgentSkills(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                agentID := c.Param("agentId")
                companyID := c.Query("companyId")

                agent, status, err := resolveAgentByParam(db, agentID, companyID)
                if err != nil {
                        c.JSON(status, gin.H{"error": err.Error()})
                        return
                }

                // Skills are stored in agent's AdapterConfig["skills"] or returned empty
                skills := []gin.H{}
                if rawSkills, ok := agent.AdapterConfig["skills"]; ok {
                        if skillList, ok := rawSkills.([]interface{}); ok {
                                for _, s := range skillList {
                                        if sm, ok := s.(map[string]interface{}); ok {
                                                skills = append(skills, gin.H{
                                                        "key":         sm["key"],
                                                        "name":        sm["name"],
                                                        "description": sm["description"],
                                                })
                                        }
                                }
                        }
                }

                c.JSON(http.StatusOK, gin.H{"skills": skills, "entries": []map[string]interface{}{}, "desiredSkills": []string{}, "warnings": []string{}, "mode": "unsupported"})
        }
}

// getAgentKeys returns the API keys for an agent.
func getAgentKeys(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                agentID := c.Param("agentId")
                companyID := c.Query("companyId")

                agent, status, err := resolveAgentByParam(db, agentID, companyID)
                if err != nil {
                        c.JSON(status, gin.H{"error": err.Error()})
                        return
                }

                var keys []models.AgentAPIKey
                db.Where("agent_id = ?", agent.ID).Order("created_at desc").Find(&keys)

                // Never return raw key values — only metadata
                out := make([]gin.H, 0, len(keys))
                for _, k := range keys {
                        out = append(out, gin.H{
                                "id":          k.ID,
                                "agentId":     k.AgentID,
                                "label":      k.Label,
                                "companyId":  k.CompanyID,
                                "lastUsedAt":  k.LastUsedAt,
                                "revokedAt":  k.RevokedAt,
                                "createdAt":   k.CreatedAt,
                        })
                }
                c.JSON(http.StatusOK, out)
        }
}

// getAgentConfigRevisions returns a history of config revisions for an agent.
// Currently returns an empty list since revision tracking is not yet implemented.
func getAgentConfigRevisions(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                agentID := c.Param("agentId")
                companyID := c.Query("companyId")

                _, status, err := resolveAgentByParam(db, agentID, companyID)
                if err != nil {
                        c.JSON(status, gin.H{"error": err.Error()})
                        return
                }

                c.JSON(http.StatusOK, []gin.H{})
        }
}

func pauseGlobalAgent(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                agent, status, err := resolveAgentByParam(db, c.Param("agentId"), c.Query("companyId"))
                if err != nil {
                        c.JSON(status, gin.H{"error": err.Error()})
                        return
                }
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
                db.Model(&models.Agent{}).Where("id = ?", agent.ID).Updates(updates)
                db.First(agent, "id = ?", agent.ID)
                c.JSON(http.StatusOK, agent)
        }
}

func resumeGlobalAgent(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                agent, status, err := resolveAgentByParam(db, c.Param("agentId"), c.Query("companyId"))
                if err != nil {
                        c.JSON(status, gin.H{"error": err.Error()})
                        return
                }
                db.Model(&models.Agent{}).Where("id = ?", agent.ID).Updates(map[string]interface{}{
                        "status":       "idle",
                        "paused_at":    nil,
                        "pause_reason": nil,
                        "updated_at":   time.Now(),
                })
                db.First(agent, "id = ?", agent.ID)
                c.JSON(http.StatusOK, agent)
        }
}

func updateAgentPermissions(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                agent, status, err := resolveAgentByParam(db, c.Param("agentId"), c.Query("companyId"))
                if err != nil {
                        c.JSON(status, gin.H{"error": err.Error()})
                        return
                }

                var req struct {
                        CanCreateAgents *bool `json:"canCreateAgents"`
                        CanAssignTasks  *bool `json:"canAssignTasks"`
                }
                if err := c.ShouldBindJSON(&req); err != nil {
                        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
                        return
                }

                perms := models.JSON{}
                if agent.Permissions != nil {
                        for k, v := range agent.Permissions {
                                perms[k] = v
                        }
                }
                if req.CanCreateAgents != nil {
                        perms["canCreateAgents"] = *req.CanCreateAgents
                }
                if req.CanAssignTasks != nil {
                        perms["canAssignTasks"] = *req.CanAssignTasks
                }

                db.Model(&models.Agent{}).Where("id = ?", agent.ID).Updates(map[string]interface{}{
                        "permissions": perms,
                        "updated_at":  time.Now(),
                })
                db.First(agent, "id = ?", agent.ID)

                actor := mw.GetActor(c)
                logActivity(db, agent.CompanyID, actor, "updated", "agent", agent.ID, nil)
                c.JSON(http.StatusOK, wrapAgent(agent))
        }
}
