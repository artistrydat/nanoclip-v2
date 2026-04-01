package handlers

import (
        "net/http"
        "time"

        "github.com/gin-gonic/gin"
        "github.com/google/uuid"
        "gorm.io/gorm"
        "paperclip-go/models"
)

// AccessRoutes registers /companies/:companyId/agent-configurations
func AccessRoutes(rg *gin.RouterGroup, db *gorm.DB) {
        rg.GET("", listAgentConfigurations(db))
        rg.POST("", createAgentConfiguration(db))
        rg.PATCH("/:configId", updateAgentConfiguration(db))
        rg.DELETE("/:configId", deleteAgentConfiguration(db))
}

// JoinRequestRoutes registers /companies/:companyId/join-requests
// In local_trusted mode there are no external join requests; return empty lists.
func JoinRequestRoutes(rg *gin.RouterGroup, _ *gorm.DB) {
        rg.GET("", func(c *gin.Context) {
                c.JSON(http.StatusOK, []interface{}{})
        })
        rg.POST("/:requestId/approve", func(c *gin.Context) {
                c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
        })
        rg.POST("/:requestId/reject", func(c *gin.Context) {
                c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
        })
}

// InviteRoutes registers invite management under a company
func InviteRoutes(rg *gin.RouterGroup, db *gorm.DB) {
        rg.GET("", listInvites(db))
        rg.POST("", createInvite(db))
        rg.DELETE("/:inviteId", deleteInvite(db))
        rg.POST("/accept", acceptInvite(db))
}

type agentConfigRow struct {
        ID        string      `json:"id"`
        CompanyID string      `json:"companyId"`
        AgentID   string      `json:"agentId"`
        Config    models.JSON `json:"config"`
        CreatedAt time.Time   `json:"createdAt"`
        UpdatedAt time.Time   `json:"updatedAt"`
}

func listAgentConfigurations(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var agents []models.Agent
                db.Where("company_id = ?", c.Param("companyId")).
                        Select("id, name, adapter_type, adapter_config, runtime_config, status, created_at, updated_at").
                        Order("created_at asc").Find(&agents)
                c.JSON(http.StatusOK, agents)
        }
}

func createAgentConfiguration(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                c.JSON(http.StatusCreated, gin.H{"created": true})
        }
}

func updateAgentConfiguration(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                c.JSON(http.StatusOK, gin.H{"updated": true})
        }
}

func deleteAgentConfiguration(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                c.JSON(http.StatusOK, gin.H{"success": true})
        }
}

func listInvites(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var invites []models.Invite
                db.Where("company_id = ? AND used_at IS NULL", c.Param("companyId")).
                        Order("created_at desc").Find(&invites)
                c.JSON(http.StatusOK, invites)
        }
}

func createInvite(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var req struct {
                        Email string `json:"email" binding:"required"`
                        Role  string `json:"role"`
                }
                if err := c.ShouldBindJSON(&req); err != nil {
                        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                        return
                }
                if req.Role == "" {
                        req.Role = "member"
                }
                now := time.Now()
                expires := now.Add(7 * 24 * time.Hour)
                invite := models.Invite{
                        ID:        uuid.NewString(),
                        CompanyID: c.Param("companyId"),
                        Email:     req.Email,
                        Role:      req.Role,
                        Token:     uuid.NewString(),
                        ExpiresAt: expires,
                        CreatedAt: now,
                        UpdatedAt: now,
                }
                db.Create(&invite)
                c.JSON(http.StatusCreated, invite)
        }
}

func deleteInvite(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                db.Where("id = ? AND company_id = ?", c.Param("inviteId"), c.Param("companyId")).
                        Delete(&models.Invite{})
                c.JSON(http.StatusOK, gin.H{"success": true})
        }
}

func acceptInvite(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var req struct {
                        Token string `json:"token" binding:"required"`
                }
                if err := c.ShouldBindJSON(&req); err != nil {
                        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                        return
                }
                var invite models.Invite
                if err := db.Where("token = ? AND used_at IS NULL AND expires_at > ?", req.Token, time.Now()).
                        First(&invite).Error; err != nil {
                        c.JSON(http.StatusNotFound, gin.H{"error": "invite not found or expired"})
                        return
                }
                now := time.Now()
                db.Model(&invite).Updates(map[string]interface{}{"used_at": now, "updated_at": now})
                c.JSON(http.StatusOK, invite)
        }
}
