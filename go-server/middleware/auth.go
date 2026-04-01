package middleware

import (
        "net/http"
        "strings"
        "time"

        "github.com/gin-gonic/gin"
        "github.com/golang-jwt/jwt/v5"
        "gorm.io/gorm"
        "paperclip-go/models"
)

type ActorInfo struct {
        UserID    string
        AgentID   string
        Type      string // "user" | "agent"
        CompanyID string
}

const ActorKey = "actor"

var deploymentMode = "local_trusted"

func SetDeploymentMode(mode string) {
        deploymentMode = mode
}

func Auth(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                // In local_trusted mode, ensure a system user exists and use it
                if deploymentMode == "local_trusted" {
                        actor := resolveActor(c, db)
                        if actor == nil {
                                // Auto-provision a local system user if none exists
                                actor = ensureLocalSystemUser(db)
                        }
                        if actor != nil {
                                c.Set(ActorKey, actor)
                        }
                        c.Next()
                        return
                }

                actor := resolveActor(c, db)
                if actor != nil {
                        c.Set(ActorKey, actor)
                }
                c.Next()
        }
}

func RequireAuth() gin.HandlerFunc {
        return func(c *gin.Context) {
                // In local_trusted mode, requests are always considered authenticated
                if deploymentMode == "local_trusted" {
                        c.Next()
                        return
                }

                actor, exists := c.Get(ActorKey)
                if !exists || actor == nil {
                        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
                        c.Abort()
                        return
                }
                c.Next()
        }
}

// ensureLocalSystemUser returns (or creates) the built-in local admin user,
// provisioning both the user record and an instance_admin role.
func ensureLocalSystemUser(db *gorm.DB) *ActorInfo {
        const localUserID = "local-system-user"
        var user models.User
        if err := db.First(&user, "id = ?", localUserID).Error; err != nil {
                now := time.Now()
                user = models.User{
                        ID:        localUserID,
                        Name:      "Local Admin",
                        Email:     "admin@localhost",
                        CreatedAt: now,
                        UpdatedAt: now,
                }
                db.Create(&user)
        }
        // Ensure admin role exists for this user
        var role models.InstanceUserRole
        if db.Where("user_id = ? AND role = 'instance_admin'", localUserID).First(&role).Error != nil {
                now := time.Now()
                db.Create(&models.InstanceUserRole{
                        ID:        "local-system-admin-role",
                        UserID:    localUserID,
                        Role:      "instance_admin",
                        CreatedAt: now,
                        UpdatedAt: now,
                })
        }
        return &ActorInfo{UserID: localUserID, Type: "user"}
}

func GetActor(c *gin.Context) *ActorInfo {
        if v, exists := c.Get(ActorKey); exists {
                if a, ok := v.(*ActorInfo); ok {
                        return a
                }
        }
        return nil
}

func resolveActor(c *gin.Context, db *gorm.DB) *ActorInfo {
        // 1. Session cookie / Bearer token (user sessions)
        token := extractBearerToken(c)
        if token == "" {
                token, _ = c.Cookie("paperclip_session")
        }

        if token != "" {
                var session models.Session
                now := time.Now()
                if err := db.Where("token = ? AND expires_at > ?", token, now).First(&session).Error; err == nil {
                        return &ActorInfo{UserID: session.UserID, Type: "user"}
                }
        }

        // 2. Agent JWT (Bearer with "agent." prefix or standard JWT)
        bearer := extractBearerToken(c)
        if bearer != "" {
                if actor := resolveAgentJWT(bearer, db); actor != nil {
                        return actor
                }
        }

        return nil
}

func extractBearerToken(c *gin.Context) string {
        header := c.GetHeader("Authorization")
        if strings.HasPrefix(header, "Bearer ") {
                return strings.TrimPrefix(header, "Bearer ")
        }
        return ""
}

var jwtSecret = []byte("paperclip-dev-secret-change-in-production")

func SetJWTSecret(secret string) {
        jwtSecret = []byte(secret)
}

type AgentClaims struct {
        AgentID   string `json:"agentId"`
        CompanyID string `json:"companyId"`
        jwt.RegisteredClaims
}

func resolveAgentJWT(tokenStr string, db *gorm.DB) *ActorInfo {
        claims := &AgentClaims{}
        token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
                return jwtSecret, nil
        })
        if err != nil || !token.Valid {
                return nil
        }
        // Verify agent exists
        var agent models.Agent
        if err := db.Where("id = ? AND company_id = ?", claims.AgentID, claims.CompanyID).First(&agent).Error; err != nil {
                return nil
        }
        return &ActorInfo{AgentID: claims.AgentID, CompanyID: claims.CompanyID, Type: "agent"}
}

func IssueAgentJWT(agentID, companyID string) (string, error) {
        claims := AgentClaims{
                AgentID:   agentID,
                CompanyID: companyID,
                RegisteredClaims: jwt.RegisteredClaims{
                        ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)),
                        IssuedAt:  jwt.NewNumericDate(time.Now()),
                },
        }
        token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
        return token.SignedString(jwtSecret)
}
