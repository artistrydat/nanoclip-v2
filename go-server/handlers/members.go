package handlers

import (
        "net/http"
        "time"

        "github.com/gin-gonic/gin"
        "github.com/google/uuid"
        "gorm.io/gorm"
        "paperclip-go/models"
)

func MemberRoutes(rg *gin.RouterGroup, db *gorm.DB) {
        rg.GET("", listMembers(db))
        rg.POST("", addMember(db))
        rg.PATCH("/:userId", updateMember(db))
        rg.DELETE("/:userId", removeMember(db))
}

func OrgRoutes(rg *gin.RouterGroup, db *gorm.DB) {
        rg.GET("", getOrgChart(db))
}

type memberWithUser struct {
        models.CompanyMembership
        User *models.User `json:"user,omitempty"`
}

func listMembers(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                companyID := c.Param("companyId")
                var memberships []models.CompanyMembership
                db.Where("company_id = ?", companyID).Order("created_at asc").Find(&memberships)

                result := make([]gin.H, 0, len(memberships))
                for _, m := range memberships {
                        var user models.User
                        db.First(&user, "id = ?", m.UserID)
                        result = append(result, gin.H{
                                "id":        m.ID,
                                "companyId": m.CompanyID,
                                "userId":    m.UserID,
                                "role":      m.Role,
                                "createdAt": m.CreatedAt,
                                "updatedAt": m.UpdatedAt,
                                "user":      user,
                        })
                }
                c.JSON(http.StatusOK, result)
        }
}

func addMember(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var req struct {
                        UserID string `json:"userId"`
                        Email  string `json:"email"`
                        Role   string `json:"role"`
                }
                if err := c.ShouldBindJSON(&req); err != nil {
                        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                        return
                }
                if req.Role == "" {
                        req.Role = "member"
                }

                // Find user by ID or email
                userID := req.UserID
                if userID == "" && req.Email != "" {
                        var user models.User
                        if err := db.Where("email = ?", req.Email).First(&user).Error; err != nil {
                                c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
                                return
                        }
                        userID = user.ID
                }
                if userID == "" {
                        c.JSON(http.StatusBadRequest, gin.H{"error": "userId or email required"})
                        return
                }

                // Check not already a member
                var existing models.CompanyMembership
                if db.Where("company_id = ? AND user_id = ?", c.Param("companyId"), userID).First(&existing).Error == nil {
                        c.JSON(http.StatusConflict, gin.H{"error": "already a member"})
                        return
                }

                membership := models.CompanyMembership{
                        ID:        uuid.NewString(),
                        CompanyID: c.Param("companyId"),
                        UserID:    userID,
                        Role:      req.Role,
                        CreatedAt: time.Now(),
                        UpdatedAt: time.Now(),
                }
                db.Create(&membership)
                c.JSON(http.StatusCreated, membership)
        }
}

func updateMember(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var req struct {
                        Role string `json:"role" binding:"required"`
                }
                if err := c.ShouldBindJSON(&req); err != nil {
                        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                        return
                }
                db.Model(&models.CompanyMembership{}).
                        Where("company_id = ? AND user_id = ?", c.Param("companyId"), c.Param("userId")).
                        Updates(map[string]interface{}{"role": req.Role, "updated_at": time.Now()})
                var m models.CompanyMembership
                db.Where("company_id = ? AND user_id = ?", c.Param("companyId"), c.Param("userId")).First(&m)
                c.JSON(http.StatusOK, m)
        }
}

func removeMember(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                db.Where("company_id = ? AND user_id = ?", c.Param("companyId"), c.Param("userId")).
                        Delete(&models.CompanyMembership{})
                c.JSON(http.StatusOK, gin.H{"success": true})
        }
}

type orgNode struct {
        ID      string     `json:"id"`
        Name    string     `json:"name"`
        Role    string     `json:"role"`
        Status  string     `json:"status"`
        Reports []*orgNode `json:"reports"`
}

func getOrgChart(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                companyID := c.Param("companyId")
                var agents []models.Agent
                db.Where("company_id = ?", companyID).Order("created_at asc").Find(&agents)

                nodeMap := make(map[string]*orgNode, len(agents))
                for i := range agents {
                        a := &agents[i]
                        nodeMap[a.ID] = &orgNode{
                                ID:      a.ID,
                                Name:    a.Name,
                                Role:    a.Role,
                                Status:  a.Status,
                                Reports: []*orgNode{},
                        }
                }

                var roots []*orgNode
                for i := range agents {
                        a := &agents[i]
                        node := nodeMap[a.ID]
                        if a.ReportsTo != nil && *a.ReportsTo != "" {
                                if parent, ok := nodeMap[*a.ReportsTo]; ok {
                                        parent.Reports = append(parent.Reports, node)
                                        continue
                                }
                        }
                        roots = append(roots, node)
                }
                if roots == nil {
                        roots = []*orgNode{}
                }
                c.JSON(http.StatusOK, roots)
        }
}
