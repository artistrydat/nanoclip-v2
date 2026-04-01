package handlers

import (
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"paperclip-go/models"
)

func SkillRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	rg.GET("", listSkills(db))
	rg.POST("", createSkill(db))
	rg.POST("/import", importSkillFromSource(db))
	rg.GET("/:skillId", getSkill(db))
	rg.PATCH("/:skillId", updateSkill(db))
	rg.DELETE("/:skillId", deleteSkill(db))
}

func listSkills(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var skills []models.CompanySkill
		db.Where("company_id = ?", c.Param("companyId")).
			Order("created_at asc").Find(&skills)
		c.JSON(http.StatusOK, skills)
	}
}

func getSkill(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var skill models.CompanySkill
		if err := db.First(&skill, "id = ? AND company_id = ?",
			c.Param("skillId"), c.Param("companyId")).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "skill not found"})
			return
		}
		c.JSON(http.StatusOK, skill)
	}
}

func createSkill(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name        string      `json:"name" binding:"required"`
			Description *string     `json:"description"`
			Kind        *string     `json:"kind"`
			Content     *string     `json:"content"`
			Config      models.JSON `json:"config"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		kind := "document"
		if req.Kind != nil {
			kind = *req.Kind
		}
		skill := models.CompanySkill{
			ID:          uuid.NewString(),
			CompanyID:   c.Param("companyId"),
			Name:        req.Name,
			Description: req.Description,
			Kind:        kind,
			Content:     req.Content,
			Config:      req.Config,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := db.Create(&skill).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, skill)
	}
}

// githubToRawURL converts a github.com blob URL to a raw.githubusercontent.com URL.
// It also accepts raw.githubusercontent.com URLs unchanged.
// Returns the raw URL and the inferred skill name.
func githubToRawURL(source string) (rawURL string, skillName string, err error) {
	s := strings.TrimSpace(source)

	// Already a raw URL
	if strings.HasPrefix(s, "https://raw.githubusercontent.com/") {
		urlPath := strings.TrimPrefix(s, "https://raw.githubusercontent.com/")
		parts := strings.SplitN(urlPath, "/", 4) // org/repo/branch/rest
		if len(parts) < 4 {
			return "", "", fmt.Errorf("unrecognised raw GitHub URL: %s", s)
		}
		filePath := parts[3]
		skillName = skillNameFromPath(filePath, parts[1])
		return s, skillName, nil
	}

	// github.com/org/repo/blob/branch/path
	if strings.HasPrefix(s, "https://github.com/") || strings.HasPrefix(s, "http://github.com/") {
		urlPath := strings.TrimPrefix(s, "https://github.com/")
		urlPath = strings.TrimPrefix(urlPath, "http://github.com/")
		// Strip query string / fragment
		if i := strings.IndexAny(urlPath, "?#"); i != -1 {
			urlPath = urlPath[:i]
		}
		parts := strings.SplitN(urlPath, "/", 4) // org/repo/blob/rest OR org/repo (no file)
		if len(parts) < 4 || parts[2] != "blob" {
			return "", "", fmt.Errorf("paste the URL of a SKILL.md file on GitHub (e.g. github.com/org/repo/blob/main/path/SKILL.md)")
		}
		// parts[3] = branch/rest-of-path
		branchAndPath := parts[3]
		rawURL = fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", parts[0], parts[1], branchAndPath)
		filePath := path.Join(strings.SplitN(branchAndPath, "/", 2)[1:]...) // strip branch prefix
		skillName = skillNameFromPath(filePath, parts[1])
		return rawURL, skillName, nil
	}

	return "", "", fmt.Errorf("unsupported source URL — paste a github.com blob URL pointing to a SKILL.md file")
}

// skillNameFromPath returns a human-readable name for the skill from the file path.
// For "skills/publish-to-pages/SKILL.md" → "publish-to-pages"
// For "SKILL.md" → repoName
func skillNameFromPath(filePath, repoName string) string {
	dir := path.Dir(filePath)
	if dir == "" || dir == "." {
		return strings.ReplaceAll(repoName, "-", " ")
	}
	base := path.Base(dir)
	return strings.ReplaceAll(base, "-", " ")
}

func importSkillFromSource(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Source string `json:"source" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		rawURL, skillName, err := githubToRawURL(req.Source)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Fetch the raw content
		httpClient := &http.Client{Timeout: 15 * time.Second}
		resp, err := httpClient.Get(rawURL)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("failed to fetch skill: %v", err)})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			c.JSON(http.StatusBadRequest, gin.H{"error": "SKILL.md not found at that URL — check the link and try again"})
			return
		}
		if resp.StatusCode != http.StatusOK {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("GitHub returned HTTP %d", resp.StatusCode)})
			return
		}

		bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024)) // 512 KB cap
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to read skill content"})
			return
		}
		content := string(bodyBytes)

		now := time.Now()
		skill := models.CompanySkill{
			ID:        uuid.NewString(),
			CompanyID: c.Param("companyId"),
			Name:      skillName,
			Kind:      "document",
			Content:   &content,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := db.Create(&skill).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"imported": []models.CompanySkill{skill},
			"warnings": []string{},
		})
	}
}

func updateSkill(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var skill models.CompanySkill
		if err := db.First(&skill, "id = ? AND company_id = ?",
			c.Param("skillId"), c.Param("companyId")).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "skill not found"})
			return
		}
		var req struct {
			Name        *string     `json:"name"`
			Description *string     `json:"description"`
			Kind        *string     `json:"kind"`
			Content     *string     `json:"content"`
			Config      models.JSON `json:"config"`
		}
		c.ShouldBindJSON(&req)
		updates := map[string]interface{}{"updated_at": time.Now()}
		if req.Name != nil {
			updates["name"] = *req.Name
		}
		if req.Description != nil {
			updates["description"] = req.Description
		}
		if req.Kind != nil {
			updates["kind"] = *req.Kind
		}
		if req.Content != nil {
			updates["content"] = req.Content
		}
		if req.Config != nil {
			updates["config"] = req.Config
		}
		db.Model(&skill).Updates(updates)
		db.First(&skill, "id = ?", skill.ID)
		c.JSON(http.StatusOK, skill)
	}
}

func deleteSkill(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		db.Where("id = ? AND company_id = ?",
			c.Param("skillId"), c.Param("companyId")).Delete(&models.CompanySkill{})
		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}
