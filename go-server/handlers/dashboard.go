package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"paperclip-go/models"
)

func DashboardRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	rg.GET("", getDashboard(db))
}

func SidebarRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	rg.GET("", getSidebarBadges(db))
}

func getDashboard(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		companyID := c.Param("companyId")

		// Agent counts by status
		var agentStats []struct {
			Status string
			Count  int64
		}
		db.Model(&models.Agent{}).
			Select("status, count(*) as count").
			Where("company_id = ?", companyID).
			Group("status").
			Scan(&agentStats)

		agentByStatus := map[string]int64{}
		for _, s := range agentStats {
			agentByStatus[s.Status] = s.Count
		}
		agentRunning := agentByStatus["running"]
		agentActive := agentByStatus["idle"] + agentByStatus["active"]
		agentPaused := agentByStatus["paused"]
		agentError := agentByStatus["error"]

		// Issue counts by status
		var issueStats []struct {
			Status string
			Count  int64
		}
		db.Model(&models.Issue{}).
			Select("status, count(*) as count").
			Where("company_id = ? AND hidden_at IS NULL", companyID).
			Group("status").
			Scan(&issueStats)

		issueByStatus := map[string]int64{}
		for _, s := range issueStats {
			issueByStatus[s.Status] = s.Count
		}
		tasksInProgress := issueByStatus["in_progress"]
		tasksOpen := issueByStatus["todo"] + issueByStatus["backlog"]
		tasksBlocked := issueByStatus["blocked"]

		// Monthly spend
		startOfMonth := time.Now().UTC().Truncate(24 * time.Hour).AddDate(0, 0, -time.Now().Day()+1)
		var monthlySpend struct{ Total int64 }
		db.Model(&models.CostEvent{}).
			Select("COALESCE(SUM(cost_cents), 0) as total").
			Where("company_id = ? AND occurred_at >= ?", companyID, startOfMonth).
			Scan(&monthlySpend)

		// Monthly budget from company
		var company models.Company
		db.First(&company, "id = ?", companyID)
		monthBudgetCents := company.BudgetMonthlyCents
		var monthUtilizationPercent int64
		if monthBudgetCents > 0 {
			monthUtilizationPercent = int64(float64(monthlySpend.Total) / float64(monthBudgetCents) * 100)
		}

		// Pending approvals
		var pendingApprovals int64
		db.Model(&models.Approval{}).
			Where("company_id = ? AND status = 'pending'", companyID).
			Count(&pendingApprovals)

		c.JSON(http.StatusOK, gin.H{
			"pendingApprovals": pendingApprovals,
			"agents": gin.H{
				"active":  agentActive,
				"running": agentRunning,
				"paused":  agentPaused,
				"error":   agentError,
			},
			"tasks": gin.H{
				"inProgress": tasksInProgress,
				"open":       tasksOpen,
				"blocked":    tasksBlocked,
			},
			"costs": gin.H{
				"monthSpendCents":         monthlySpend.Total,
				"monthBudgetCents":        monthBudgetCents,
				"monthUtilizationPercent": monthUtilizationPercent,
			},
			"budgets": gin.H{
				"activeIncidents":  0,
				"pausedAgents":     0,
				"pausedProjects":   0,
				"pendingApprovals": 0,
			},
		})
	}
}

func getSidebarBadges(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		companyID := c.Param("companyId")

		var pendingApprovals int64
		db.Model(&models.Approval{}).
			Where("company_id = ? AND status = 'pending'", companyID).
			Count(&pendingApprovals)

		var inProgressIssues int64
		db.Model(&models.Issue{}).
			Where("company_id = ? AND status = 'in_progress' AND hidden_at IS NULL", companyID).
			Count(&inProgressIssues)

		var activeRuns int64
		db.Model(&models.HeartbeatRun{}).
			Where("company_id = ? AND status IN ('queued','running')", companyID).
			Count(&activeRuns)

		c.JSON(http.StatusOK, gin.H{
			"pendingApprovals": pendingApprovals,
			"inProgressIssues": inProgressIssues,
			"activeRuns":       activeRuns,
		})
	}
}
