package handlers

import (
	_ "embed"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"paperclip-go/models"
)

//go:embed embedded_skills/mariadb.md
var mariadbSkillContent string

//go:embed embedded_skills/nanoclip-integration.md
var nanoclipIntegrationSkillContent string

type permissionSkill struct {
	name        string
	key         string
	description string
	content     string
}

var permissionSkillDefs = []permissionSkill{
	{
		name:        "MariaDB",
		key:         "mariadb",
		description: "Interact with the local MariaDB database — read, write, and manage NanoClip data.",
		content:     mariadbSkillContent,
	},
	{
		name:        "NanoClip Integration",
		key:         "nanoclip-integration",
		description: "Use the NanoClip REST API to manage issues, agents, approvals, and more.",
		content:     nanoclipIntegrationSkillContent,
	},
}

// upsertPermissionSkills ensures the MariaDB and NanoClip Integration skills exist for the
// company, then returns their skill keys so the caller can add them to the agent's desiredSkills.
func upsertPermissionSkills(db *gorm.DB, companyID string) []string {
	keys := make([]string, 0, len(permissionSkillDefs))
	now := time.Now()

	for _, def := range permissionSkillDefs {
		var skill models.CompanySkill
		err := db.Where("company_id = ? AND name = ?", companyID, def.name).First(&skill).Error
		if err != nil {
			content := def.content
			descCopy := def.description
			skill = models.CompanySkill{
				ID:          uuid.NewString(),
				CompanyID:   companyID,
				Name:        def.name,
				Description: &descCopy,
				Kind:        "document",
				Content:     &content,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			db.Create(&skill)
		}
		keys = append(keys, def.key)
	}

	return keys
}

// mergeDesiredSkills adds newKeys into existing desiredSkills without duplication.
func mergeDesiredSkills(existing []interface{}, newKeys []string) []interface{} {
	seen := map[string]bool{}
	for _, v := range existing {
		if s, ok := v.(string); ok {
			seen[s] = true
		}
	}
	result := make([]interface{}, len(existing))
	copy(result, existing)
	for _, k := range newKeys {
		if !seen[k] {
			result = append(result, k)
			seen[k] = true
		}
	}
	return result
}
