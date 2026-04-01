package handlers

import (
	"regexp"
	"strings"

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
