package handlers

import (
        "fmt"
        "net/http"
        "strings"
        "time"
        "unicode"

        "github.com/gin-gonic/gin"
        "github.com/google/uuid"
        "gorm.io/gorm"
        "paperclip-go/middleware"
        "paperclip-go/models"
)

// derivePrefixFromName extracts up to 4 uppercase letters from the company name.
func derivePrefixFromName(name string) string {
        var letters []rune
        for _, r := range strings.ToUpper(name) {
                if unicode.IsLetter(r) {
                        letters = append(letters, r)
                        if len(letters) == 4 {
                                break
                        }
                }
        }
        if len(letters) == 0 {
                return "CO"
        }
        return string(letters)
}

// uniquePrefix ensures the prefix is not already taken, appending a number if needed.
func uniquePrefix(db *gorm.DB, base string) string {
        candidate := base
        for i := 2; i <= 99; i++ {
                var count int64
                db.Model(&models.Company{}).Where("issue_prefix = ?", candidate).Count(&count)
                if count == 0 {
                        return candidate
                }
                candidate = fmt.Sprintf("%s%d", base, i)
        }
        return candidate
}

func CompanyRoutes(rg *gin.RouterGroup, db *gorm.DB) {
        rg.GET("", listCompanies(db))
        rg.POST("", createCompany(db))
        // /stats must be registered before /:companyId to avoid param capture
        rg.GET("/stats", globalCompanyStats(db))
        rg.POST("/import/preview", importPreview())
        rg.POST("/import", importCompany(db))
        rg.GET("/:companyId", getCompany(db))
        rg.PATCH("/:companyId", updateCompany(db))
        rg.PATCH("/:companyId/branding", updateCompanyBranding(db))
        rg.DELETE("/:companyId", deleteCompany(db))
        rg.POST("/:companyId/pause", pauseCompany(db))
        rg.POST("/:companyId/resume", resumeCompany(db))
        rg.POST("/:companyId/archive", archiveCompany(db))
}

func listCompanies(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                actor := middleware.GetActor(c)
                var companies []models.Company
                q := db.Order("created_at desc")
                if actor != nil && actor.UserID != "" {
                        var role models.InstanceUserRole
                        isAdmin := db.Where("user_id = ? AND role = 'instance_admin'", actor.UserID).First(&role).Error == nil
                        if !isAdmin {
                                var membershipIDs []string
                                db.Model(&models.CompanyMembership{}).
                                        Where("user_id = ?", actor.UserID).
                                        Pluck("company_id", &membershipIDs)
                                q = q.Where("id IN ?", membershipIDs)
                        }
                }
                q.Find(&companies)
                c.JSON(http.StatusOK, companies)
        }
}

func globalCompanyStats(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var total, active, paused int64
                db.Model(&models.Company{}).Count(&total)
                db.Model(&models.Company{}).Where("status = 'active'").Count(&active)
                db.Model(&models.Company{}).Where("status = 'paused'").Count(&paused)
                c.JSON(http.StatusOK, gin.H{
                        "total":  total,
                        "active": active,
                        "paused": paused,
                })
        }
}

func importPreview() gin.HandlerFunc {
        return func(c *gin.Context) {
                c.JSON(http.StatusOK, gin.H{"valid": true, "companies": []interface{}{}})
        }
}

func importCompany(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                c.JSON(http.StatusOK, gin.H{"imported": 0})
        }
}

func getCompany(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var company models.Company
                if err := db.First(&company, "id = ?", c.Param("companyId")).Error; err != nil {
                        c.JSON(http.StatusNotFound, gin.H{"error": "company not found"})
                        return
                }
                c.JSON(http.StatusOK, company)
        }
}

type createCompanyRequest struct {
        Name        string  `json:"name" binding:"required"`
        Description *string `json:"description"`
        IssuePrefix *string `json:"issuePrefix"`
        BrandColor  *string `json:"brandColor"`
}

func createCompany(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var req createCompanyRequest
                if err := c.ShouldBindJSON(&req); err != nil {
                        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                        return
                }

                prefix := derivePrefixFromName(req.Name)
                if req.IssuePrefix != nil && *req.IssuePrefix != "" {
                        prefix = *req.IssuePrefix
                }
                prefix = uniquePrefix(db, prefix)

                company := models.Company{
                        ID:          uuid.NewString(),
                        Name:        req.Name,
                        Description: req.Description,
                        Status:      "active",
                        IssuePrefix: prefix,
                        BrandColor:  req.BrandColor,
                        CreatedAt:   time.Now(),
                        UpdatedAt:   time.Now(),
                }
                if err := db.Create(&company).Error; err != nil {
                        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                        return
                }

                actor := middleware.GetActor(c)
                if actor != nil && actor.UserID != "" {
                        db.Create(&models.CompanyMembership{
                                ID:        uuid.NewString(),
                                CompanyID: company.ID,
                                UserID:    actor.UserID,
                                Role:      "owner",
                                CreatedAt: time.Now(),
                                UpdatedAt: time.Now(),
                        })
                }

                logActivity(db, company.ID, actor, "created", "company", company.ID, nil)
                c.JSON(http.StatusCreated, company)
        }
}

type updateCompanyRequest struct {
        Name                             *string `json:"name"`
        Description                      *string `json:"description"`
        Status                           *string `json:"status"`
        BrandColor                       *string `json:"brandColor"`
        BudgetMonthlyCents               *int    `json:"budgetMonthlyCents"`
        RequireBoardApprovalForNewAgents *bool   `json:"requireBoardApprovalForNewAgents"`
}

func updateCompany(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var company models.Company
                if err := db.First(&company, "id = ?", c.Param("companyId")).Error; err != nil {
                        c.JSON(http.StatusNotFound, gin.H{"error": "company not found"})
                        return
                }
                var req updateCompanyRequest
                if err := c.ShouldBindJSON(&req); err != nil {
                        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                        return
                }
                updates := map[string]interface{}{"updated_at": time.Now()}
                if req.Name != nil {
                        updates["name"] = *req.Name
                }
                if req.Description != nil {
                        updates["description"] = *req.Description
                }
                if req.Status != nil {
                        updates["status"] = *req.Status
                }
                if req.BrandColor != nil {
                        updates["brand_color"] = *req.BrandColor
                }
                if req.BudgetMonthlyCents != nil {
                        updates["budget_monthly_cents"] = *req.BudgetMonthlyCents
                }
                if req.RequireBoardApprovalForNewAgents != nil {
                        updates["require_board_approval_for_new_agents"] = *req.RequireBoardApprovalForNewAgents
                }
                db.Model(&company).Updates(updates)
                db.First(&company, "id = ?", company.ID)

                actor := middleware.GetActor(c)
                logActivity(db, company.ID, actor, "updated", "company", company.ID, nil)
                c.JSON(http.StatusOK, company)
        }
}

func updateCompanyBranding(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var req struct {
                        BrandColor *string `json:"brandColor"`
                        LogoURL    *string `json:"logoUrl"`
                }
                c.ShouldBindJSON(&req)
                updates := map[string]interface{}{"updated_at": time.Now()}
                if req.BrandColor != nil {
                        updates["brand_color"] = *req.BrandColor
                }
                if req.LogoURL != nil {
                        updates["logo_url"] = *req.LogoURL
                }
                db.Model(&models.Company{}).Where("id = ?", c.Param("companyId")).Updates(updates)
                var company models.Company
                db.First(&company, "id = ?", c.Param("companyId"))
                c.JSON(http.StatusOK, company)
        }
}

func deleteCompany(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                id := c.Param("companyId")
                if err := db.Where("id = ?", id).Delete(&models.Company{}).Error; err != nil {
                        c.JSON(http.StatusNotFound, gin.H{"error": "company not found"})
                        return
                }
                c.JSON(http.StatusOK, gin.H{"success": true})
        }
}

func pauseCompany(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var req struct {
                        Reason *string `json:"reason"`
                }
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
                db.Model(&models.Company{}).Where("id = ?", c.Param("companyId")).Updates(updates)
                var company models.Company
                db.First(&company, "id = ?", c.Param("companyId"))
                c.JSON(http.StatusOK, company)
        }
}

func resumeCompany(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                now := time.Now()
                db.Model(&models.Company{}).Where("id = ?", c.Param("companyId")).Updates(map[string]interface{}{
                        "status":       "active",
                        "paused_at":    nil,
                        "pause_reason": nil,
                        "updated_at":   now,
                })
                var company models.Company
                db.First(&company, "id = ?", c.Param("companyId"))
                c.JSON(http.StatusOK, company)
        }
}

func archiveCompany(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                now := time.Now()
                db.Model(&models.Company{}).Where("id = ?", c.Param("companyId")).Updates(map[string]interface{}{
                        "status":      "archived",
                        "archived_at": now,
                        "updated_at":  now,
                })
                var company models.Company
                db.First(&company, "id = ?", c.Param("companyId"))
                c.JSON(http.StatusOK, company)
        }
}
