package handlers

import (
        "regexp"
        "strings"

        "gorm.io/gorm"
        "paperclip-go/models"
)

var agentNonAlphanumRe = regexp.MustCompile(`[^a-z0-9]+`)

func agentSlugifyName(name string) string {
        s := strings.ToLower(name)
        s = agentNonAlphanumRe.ReplaceAllString(s, "-")
        return strings.Trim(s, "-")
}

func computeAgentUrlKey(name, id string) string {
        slug := agentSlugifyName(name)
        shortID := strings.ReplaceAll(id, "-", "")
        if len(shortID) > 8 {
                shortID = shortID[:8]
        }
        if slug != "" {
                return slug + "-" + shortID
        }
        return shortID
}

type agentResponse struct {
        *models.Agent
        UrlKey string `json:"urlKey"`
}

func wrapAgent(a *models.Agent) agentResponse {
        return agentResponse{Agent: a, UrlKey: computeAgentUrlKey(a.Name, a.ID)}
}

func wrapAgents(agents []models.Agent) []agentResponse {
        out := make([]agentResponse, len(agents))
        for i := range agents {
                out[i] = agentResponse{Agent: &agents[i], UrlKey: computeAgentUrlKey(agents[i].Name, agents[i].ID)}
        }
        return out
}

// isAgentSubordinate returns true if targetID is a direct or indirect report of superiorID
// within companyID. Uses iterative BFS over the reports_to chain (avoids deep recursion).
func isAgentSubordinate(db *gorm.DB, companyID, superiorID, targetID string) bool {
        if superiorID == targetID {
                return false
        }
        visited := map[string]bool{}
        queue := []string{superiorID}
        for len(queue) > 0 {
                current := queue[0]
                queue = queue[1:]
                if visited[current] {
                        continue
                }
                visited[current] = true
                var reports []models.Agent
                db.Select("id").Where("company_id = ? AND reports_to = ?", companyID, current).Find(&reports)
                for _, r := range reports {
                        if r.ID == targetID {
                                return true
                        }
                        if !visited[r.ID] {
                                queue = append(queue, r.ID)
                        }
                }
        }
        return false
}
