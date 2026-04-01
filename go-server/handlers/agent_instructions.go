package handlers

import (
        "crypto/rand"
        "encoding/hex"
        "fmt"
        "net/http"
        "os"
        "path/filepath"
        "strings"
        "time"

        "github.com/gin-gonic/gin"
        "github.com/google/uuid"
        "gorm.io/gorm"
        "paperclip-go/models"
)

// instructionsDir returns the per-agent instructions directory.
func instructionsDir(agentID string) (string, error) {
        home, err := os.UserHomeDir()
        if err != nil {
                return "", err
        }
        dir := filepath.Join(home, ".nanoclip", "agents", agentID, "instructions")
        if err := os.MkdirAll(dir, 0755); err != nil {
                return "", err
        }
        return dir, nil
}

// safeRelPath validates and cleans the relative file path.
func safeRelPath(raw string) (string, error) {
        clean := filepath.Clean(raw)
        if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
                return "", fmt.Errorf("invalid path")
        }
        return clean, nil
}

// GetInstructionsBundle handles GET /agents/:agentId/instructions-bundle
func GetInstructionsBundle(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                agentID := c.Param("agentId")
                companyID := c.Query("companyId")

                _, status, err := resolveAgentByParam(db, agentID, companyID)
                if err != nil {
                        c.JSON(status, gin.H{"error": err.Error()})
                        return
                }

                dir, err := instructionsDir(agentID)
                if err != nil {
                        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                        return
                }

                // Ensure AGENTS.md exists with a default stub
                defaultFile := filepath.Join(dir, "AGENTS.md")
                if _, err := os.Stat(defaultFile); os.IsNotExist(err) {
                        os.WriteFile(defaultFile, []byte("# Agent Instructions\n\nDescribe what this agent should do.\n"), 0644)
                }

                // Walk directory and collect files
                type fileEntry struct {
                        Path string `json:"path"`
                        Size int64  `json:"size"`
                }
                var files []fileEntry
                filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
                        if err != nil || info.IsDir() {
                                return nil
                        }
                        rel, err := filepath.Rel(dir, path)
                        if err != nil {
                                return nil
                        }
                        files = append(files, fileEntry{Path: rel, Size: info.Size()})
                        return nil
                })

                c.JSON(http.StatusOK, gin.H{
                        "mode":            "managed",
                        "managedRootPath": "",
                        "rootPath":        "",
                        "entryFile":       "AGENTS.md",
                        "files":           files,
                })
        }
}

// UpdateInstructionsBundle handles PATCH /agents/:agentId/instructions-bundle
func UpdateInstructionsBundle(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                agentID := c.Param("agentId")
                companyID := c.Query("companyId")

                _, status, err := resolveAgentByParam(db, agentID, companyID)
                if err != nil {
                        c.JSON(status, gin.H{"error": err.Error()})
                        return
                }

                dir, err := instructionsDir(agentID)
                if err != nil {
                        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                        return
                }

                var files []gin.H
                filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
                        if err != nil || info.IsDir() {
                                return nil
                        }
                        rel, err := filepath.Rel(dir, path)
                        if err != nil {
                                return nil
                        }
                        files = append(files, gin.H{"path": rel})
                        return nil
                })

                c.JSON(http.StatusOK, gin.H{
                        "mode":            "managed",
                        "managedRootPath": "",
                        "rootPath":        "",
                        "entryFile":       "AGENTS.md",
                        "files":           files,
                })
        }
}

// GetInstructionsFile handles GET /agents/:agentId/instructions-bundle/file?path=...
func GetInstructionsFile(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                agentID := c.Param("agentId")
                companyID := c.Query("companyId")
                rawPath := c.Query("path")
                if rawPath == "" {
                        rawPath = "AGENTS.md"
                }

                _, status, err := resolveAgentByParam(db, agentID, companyID)
                if err != nil {
                        c.JSON(status, gin.H{"error": err.Error()})
                        return
                }

                relPath, err := safeRelPath(rawPath)
                if err != nil {
                        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
                        return
                }

                dir, err := instructionsDir(agentID)
                if err != nil {
                        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                        return
                }

                content, err := os.ReadFile(filepath.Join(dir, relPath))
                if err != nil {
                        if os.IsNotExist(err) {
                                c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
                        } else {
                                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                        }
                        return
                }

                c.JSON(http.StatusOK, gin.H{
                        "path":    relPath,
                        "content": string(content),
                })
        }
}

// SaveInstructionsFile handles PUT /agents/:agentId/instructions-bundle/file
func SaveInstructionsFile(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                agentID := c.Param("agentId")
                companyID := c.Query("companyId")

                _, status, err := resolveAgentByParam(db, agentID, companyID)
                if err != nil {
                        c.JSON(status, gin.H{"error": err.Error()})
                        return
                }

                var body struct {
                        Path    string `json:"path"`
                        Content string `json:"content"`
                }
                if err := c.ShouldBindJSON(&body); err != nil || body.Path == "" {
                        c.JSON(http.StatusBadRequest, gin.H{"error": "path and content required"})
                        return
                }

                relPath, err := safeRelPath(body.Path)
                if err != nil {
                        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
                        return
                }

                dir, err := instructionsDir(agentID)
                if err != nil {
                        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                        return
                }

                fullPath := filepath.Join(dir, relPath)
                os.MkdirAll(filepath.Dir(fullPath), 0755)
                if err := os.WriteFile(fullPath, []byte(body.Content), 0644); err != nil {
                        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                        return
                }

                c.JSON(http.StatusOK, gin.H{
                        "path":    relPath,
                        "content": body.Content,
                })
        }
}

// DeleteInstructionsFile handles DELETE /agents/:agentId/instructions-bundle/file?path=...
func DeleteInstructionsFile(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                agentID := c.Param("agentId")
                companyID := c.Query("companyId")
                rawPath := c.Query("path")

                _, status, err := resolveAgentByParam(db, agentID, companyID)
                if err != nil {
                        c.JSON(status, gin.H{"error": err.Error()})
                        return
                }

                relPath, err := safeRelPath(rawPath)
                if err != nil || relPath == "AGENTS.md" {
                        c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete entry file"})
                        return
                }

                dir, err := instructionsDir(agentID)
                if err != nil {
                        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                        return
                }

                if err := os.Remove(filepath.Join(dir, relPath)); err != nil && !os.IsNotExist(err) {
                        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                        return
                }

                c.JSON(http.StatusOK, gin.H{"ok": true})
        }
}

// CreateAgentKey handles POST /agents/:agentId/keys
func CreateAgentKey(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                agentID := c.Param("agentId")
                companyID := c.Query("companyId")

                agent, status, err := resolveAgentByParam(db, agentID, companyID)
                if err != nil {
                        c.JSON(status, gin.H{"error": err.Error()})
                        return
                }

                var body struct {
                        Name string `json:"name"`
                }
                c.ShouldBindJSON(&body)
                label := body.Name
                if label == "" {
                        label = "Default"
                }

                // Generate a random token
                raw := make([]byte, 32)
                rand.Read(raw)
                token := "pc_" + hex.EncodeToString(raw)

                key := models.AgentAPIKey{
                        ID:        uuid.NewString(),
                        AgentID:   agent.ID,
                        CompanyID: agent.CompanyID,
                        KeyHash:   token, // in local_trusted mode we store it plainly
                        Label:     &label,
                        CreatedAt: time.Now(),
                }
                if err := db.Create(&key).Error; err != nil {
                        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                        return
                }

                c.JSON(http.StatusCreated, gin.H{
                        "id":        key.ID,
                        "key":       token,
                        "token":     token,
                        "name":      label,
                        "createdAt": key.CreatedAt,
                })
        }
}

// RevokeAgentKey handles DELETE /agents/:agentId/keys/:keyId
func RevokeAgentKey(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                agentID := c.Param("agentId")
                companyID := c.Query("companyId")
                keyID := c.Param("keyId")

                agent, status, err := resolveAgentByParam(db, agentID, companyID)
                if err != nil {
                        c.JSON(status, gin.H{"error": err.Error()})
                        return
                }

                now := time.Now()
                result := db.Model(&models.AgentAPIKey{}).
                        Where("id = ? AND agent_id = ?", keyID, agent.ID).
                        Update("revoked_at", now)
                if result.Error != nil {
                        c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
                        return
                }
                if result.RowsAffected == 0 {
                        c.JSON(http.StatusNotFound, gin.H{"error": "key not found"})
                        return
                }

                c.JSON(http.StatusOK, gin.H{"ok": true})
        }
}
