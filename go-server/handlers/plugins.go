package handlers

import (
        "encoding/json"
        "net/http"
        "strconv"
        "time"

        "github.com/gin-gonic/gin"
        "github.com/google/uuid"
        "gorm.io/gorm"
        "paperclip-go/models"
        "paperclip-go/services"
)

func PluginRoutes(rg *gin.RouterGroup, db *gorm.DB) {
        rg.GET("", listPlugins(db))
        rg.POST("", createPlugin(db))
        rg.GET("/ui-contributions", getPluginUIContributions(db))
        rg.GET("/examples", listExamples())
        rg.POST("/install", installPlugin(db))
        rg.GET("/:pluginId", getPlugin(db))
        rg.PATCH("/:pluginId", updatePlugin(db))
        rg.DELETE("/:pluginId", deletePlugin(db))
        rg.POST("/:pluginId/enable", enablePlugin(db))
        rg.POST("/:pluginId/disable", disablePlugin(db))
        rg.GET("/:pluginId/health", pluginHealth(db))
        rg.GET("/:pluginId/dashboard", pluginDashboard(db))
        rg.GET("/:pluginId/config", getPluginConfig(db))
        rg.POST("/:pluginId/config", savePluginConfig(db))
        rg.POST("/:pluginId/config/test", testPluginConfig(db))
        rg.GET("/:pluginId/logs", pluginLogs(db))
}

func listPlugins(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var plugins []models.Plugin
                q := db.Order("created_at asc")
                if status := c.Query("status"); status != "" {
                        q = q.Where("status = ?", status)
                }
                q.Find(&plugins)
                if plugins == nil {
                        plugins = []models.Plugin{}
                }
                c.JSON(http.StatusOK, plugins)
        }
}

func getPluginUIContributions(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var plugins []models.Plugin
                db.Where("enabled = ? AND ui_contributions IS NOT NULL AND status = ?", true, "ready").Find(&plugins)
                contributions := []interface{}{}
                for _, p := range plugins {
                        if p.UIContributions != nil {
                                contributions = append(contributions, p.UIContributions)
                        }
                }
                c.JSON(http.StatusOK, contributions)
        }
}

func listExamples() gin.HandlerFunc {
        return func(c *gin.Context) {
                // Expose the bundled Telegram plugin as an available example
                examples := []map[string]interface{}{
                        {
                                "packageName": services.TelegramPluginPackageName(),
                                "pluginKey":   services.TelegramPluginKey(),
                                "displayName": "Telegram Bot",
                                "description": "Bidirectional Telegram integration: push notifications, bot commands, inline approve/reject, and reply routing.",
                                "localPath":   services.TelegramPluginPackageName(),
                                "tag":         "example",
                        },
                }
                c.JSON(http.StatusOK, examples)
        }
}

func installPlugin(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var req struct {
                        PackageName string `json:"packageName"`
                        Version     string `json:"version"`
                        IsLocalPath bool   `json:"isLocalPath"`
                }
                if err := c.ShouldBindJSON(&req); err != nil {
                        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                        return
                }

                // Only the Telegram plugin is supported as a built-in
                if req.PackageName != services.TelegramPluginPackageName() {
                        c.JSON(http.StatusBadRequest, gin.H{"error": "only paperclip-plugin-telegram is supported as a built-in plugin"})
                        return
                }

                // Check if already installed
                var existing models.Plugin
                if db.Where("package_name = ?", req.PackageName).First(&existing).Error == nil {
                        c.JSON(http.StatusOK, existing)
                        return
                }

                manifest := services.TelegramManifest()
                categoriesBytes, _ := json.Marshal([]string{"connector", "automation"})
                ver := "0.2.4"

                plugin := models.Plugin{
                        ID:           uuid.NewString(),
                        PackageName:  services.TelegramPluginPackageName(),
                        PluginKey:    services.TelegramPluginKey(),
                        Name:         "Telegram Bot",
                        Version:      &ver,
                        Status:       "disabled",
                        Enabled:      true,
                        ManifestJSON: models.JSON(manifest),
                        Categories:   models.JSONAny(categoriesBytes),
                        CreatedAt:    time.Now(),
                        UpdatedAt:    time.Now(),
                }
                if err := db.Create(&plugin).Error; err != nil {
                        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                        return
                }
                c.JSON(http.StatusCreated, plugin)
        }
}

func getPlugin(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var plugin models.Plugin
                if err := db.First(&plugin, "id = ? OR plugin_key = ?", c.Param("pluginId"), c.Param("pluginId")).Error; err != nil {
                        c.JSON(http.StatusNotFound, gin.H{"error": "plugin not found"})
                        return
                }
                c.JSON(http.StatusOK, plugin)
        }
}

func createPlugin(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var req struct {
                        Name    string      `json:"name" binding:"required"`
                        Version *string     `json:"version"`
                        Config  models.JSON `json:"config"`
                }
                if err := c.ShouldBindJSON(&req); err != nil {
                        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                        return
                }
                plugin := models.Plugin{
                        ID:        uuid.NewString(),
                        Name:      req.Name,
                        Version:   req.Version,
                        Status:    "disabled",
                        Enabled:   true,
                        Config:    req.Config,
                        CreatedAt: time.Now(),
                        UpdatedAt: time.Now(),
                }
                db.Create(&plugin)
                c.JSON(http.StatusCreated, plugin)
        }
}

func updatePlugin(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var req struct {
                        Enabled *bool       `json:"enabled"`
                        Config  models.JSON `json:"config"`
                }
                c.ShouldBindJSON(&req)
                updates := map[string]interface{}{"updated_at": time.Now()}
                if req.Enabled != nil {
                        updates["enabled"] = *req.Enabled
                }
                if req.Config != nil {
                        updates["config"] = req.Config
                }
                db.Model(&models.Plugin{}).Where("id = ?", c.Param("pluginId")).Updates(updates)
                var plugin models.Plugin
                db.First(&plugin, "id = ?", c.Param("pluginId"))
                c.JSON(http.StatusOK, plugin)
        }
}

func deletePlugin(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                db.Where("id = ?", c.Param("pluginId")).Delete(&models.Plugin{})
                c.JSON(http.StatusOK, gin.H{"ok": true})
        }
}

func enablePlugin(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                pluginID := c.Param("pluginId")
                updates := map[string]interface{}{
                        "status":     "ready",
                        "enabled":    true,
                        "last_error": nil,
                        "updated_at": time.Now(),
                }
                db.Model(&models.Plugin{}).Where("id = ?", pluginID).Updates(updates)

                // Reload telegram service if this is the telegram plugin
                var plugin models.Plugin
                if db.First(&plugin, "id = ?", pluginID).Error == nil {
                        if plugin.PluginKey == services.TelegramPluginKey() && services.GlobalTelegramService != nil {
                                services.GlobalTelegramService.Reload()
                        }
                }
                c.JSON(http.StatusOK, gin.H{"ok": true})
        }
}

func disablePlugin(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var req struct {
                        Reason string `json:"reason"`
                }
                c.ShouldBindJSON(&req)
                reason := "disabled by operator"
                if req.Reason != "" {
                        reason = req.Reason
                }
                updates := map[string]interface{}{
                        "status":     "disabled",
                        "enabled":    false,
                        "last_error": reason,
                        "updated_at": time.Now(),
                }
                db.Model(&models.Plugin{}).Where("id = ?", c.Param("pluginId")).Updates(updates)
                c.JSON(http.StatusOK, gin.H{"ok": true})
        }
}

func pluginHealth(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                pluginID := c.Param("pluginId")
                var plugin models.Plugin
                if err := db.First(&plugin, "id = ?", pluginID).Error; err != nil {
                        c.JSON(http.StatusNotFound, gin.H{"error": "plugin not found"})
                        return
                }

                healthy := plugin.Status == "ready"
                checks := []map[string]interface{}{
                        {"name": "Plugin status", "passed": healthy},
                }
                if plugin.PluginKey == services.TelegramPluginKey() {
                        tokenConfigured := false
                        if plugin.Config != nil {
                                raw, _ := json.Marshal(plugin.Config)
                                var cfg map[string]interface{}
                                if json.Unmarshal(raw, &cfg) == nil {
                                        tok, _ := cfg["botToken"].(string)
                                        tokenConfigured = tok != ""
                                }
                        }
                        checks = append(checks, map[string]interface{}{
                                "name":   "Bot token configured",
                                "passed": tokenConfigured,
                        })
                }

                result := map[string]interface{}{
                        "pluginId": pluginID,
                        "status":   plugin.Status,
                        "healthy":  healthy,
                        "checks":   checks,
                }
                if plugin.LastError != nil {
                        result["lastError"] = *plugin.LastError
                }
                c.JSON(http.StatusOK, result)
        }
}

func pluginDashboard(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                pluginID := c.Param("pluginId")
                var plugin models.Plugin
                if err := db.First(&plugin, "id = ?", pluginID).Error; err != nil {
                        c.JSON(http.StatusNotFound, gin.H{"error": "plugin not found"})
                        return
                }

                // Recent log entries as "job runs" approximation
                var logs []models.PluginLog
                db.Where("plugin_id = ?", pluginID).Order("created_at desc").Limit(10).Find(&logs)

                recentJobRuns := []interface{}{}
                for _, l := range logs {
                        status := "success"
                        if l.Level == "error" || l.Level == "warn" {
                                status = "failed"
                        }
                        recentJobRuns = append(recentJobRuns, map[string]interface{}{
                                "id":         l.ID,
                                "jobId":      l.ID,
                                "jobKey":     "telegram-event",
                                "trigger":    "event",
                                "status":     status,
                                "durationMs": nil,
                                "error":      nil,
                                "startedAt":  nil,
                                "finishedAt": nil,
                                "createdAt":  l.CreatedAt.Format(time.RFC3339),
                        })
                }

                workerStatus := "stopped"
                if plugin.Status == "ready" {
                        workerStatus = "running"
                }
                worker := map[string]interface{}{
                        "status":            workerStatus,
                        "pid":               nil,
                        "uptime":            nil,
                        "consecutiveCrashes": 0,
                        "totalCrashes":       0,
                        "pendingRequests":    0,
                        "lastCrashAt":        nil,
                        "nextRestartAt":      nil,
                }

                healthy := plugin.Status == "ready"
                health := map[string]interface{}{
                        "pluginId": pluginID,
                        "status":   plugin.Status,
                        "healthy":  healthy,
                        "checks":   []interface{}{},
                }

                c.JSON(http.StatusOK, map[string]interface{}{
                        "pluginId":               pluginID,
                        "worker":                 worker,
                        "recentJobRuns":          recentJobRuns,
                        "recentWebhookDeliveries": []interface{}{},
                        "health":                 health,
                        "checkedAt":              time.Now().Format(time.RFC3339),
                })
        }
}

func getPluginConfig(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                pluginID := c.Param("pluginId")
                var plugin models.Plugin
                if err := db.First(&plugin, "id = ?", pluginID).Error; err != nil {
                        c.JSON(http.StatusNotFound, gin.H{"error": "plugin not found"})
                        return
                }
                if plugin.Config == nil {
                        c.JSON(http.StatusOK, nil)
                        return
                }
                c.JSON(http.StatusOK, map[string]interface{}{
                        "pluginId":   pluginID,
                        "configJson": plugin.Config,
                })
        }
}

func savePluginConfig(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                pluginID := c.Param("pluginId")
                var req struct {
                        ConfigJSON models.JSON `json:"configJson"`
                }
                if err := c.ShouldBindJSON(&req); err != nil {
                        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                        return
                }
                db.Model(&models.Plugin{}).Where("id = ?", pluginID).Updates(map[string]interface{}{
                        "config":     req.ConfigJSON,
                        "updated_at": time.Now(),
                })

                // Reload Telegram service if this is the telegram plugin
                var plugin models.Plugin
                if db.First(&plugin, "id = ?", pluginID).Error == nil {
                        if plugin.PluginKey == services.TelegramPluginKey() && services.GlobalTelegramService != nil {
                                services.GlobalTelegramService.Reload()
                        }
                }

                c.JSON(http.StatusOK, map[string]interface{}{
                        "pluginId":   pluginID,
                        "configJson": req.ConfigJSON,
                })
        }
}

func testPluginConfig(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var req struct {
                        ConfigJSON map[string]interface{} `json:"configJson"`
                }
                if err := c.ShouldBindJSON(&req); err != nil {
                        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                        return
                }
                // Validate bot token is present for the Telegram plugin
                tok, _ := req.ConfigJSON["botToken"].(string)
                if tok == "" {
                        c.JSON(http.StatusOK, gin.H{"valid": false, "message": "botToken is required"})
                        return
                }
                c.JSON(http.StatusOK, gin.H{"valid": true})
        }
}

func pluginLogs(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                pluginID := c.Param("pluginId")
                limit := 50
                if l := c.Query("limit"); l != "" {
                        if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 500 {
                                limit = n
                        }
                }
                var logs []models.PluginLog
                q := db.Where("plugin_id = ?", pluginID).Order("created_at desc").Limit(limit)
                if level := c.Query("level"); level != "" {
                        q = q.Where("level = ?", level)
                }
                q.Find(&logs)
                if logs == nil {
                        logs = []models.PluginLog{}
                }
                c.JSON(http.StatusOK, logs)
        }
}
