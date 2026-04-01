package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"paperclip-go/models"
)

func BudgetRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	rg.GET("/overview", getBudgetOverview(db))
}

func getBudgetOverview(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		companyID := c.Param("companyId")

		var agents []models.Agent
		db.Where("company_id = ? AND status != 'terminated'", companyID).Find(&agents)

		type PolicySummary struct {
			PolicyID           string  `json:"policyId"`
			CompanyID          string  `json:"companyId"`
			ScopeType          string  `json:"scopeType"`
			ScopeID            string  `json:"scopeId"`
			ScopeName          string  `json:"scopeName"`
			Metric             string  `json:"metric"`
			WindowKind         string  `json:"windowKind"`
			Amount             int     `json:"amount"`
			ObservedAmount     int     `json:"observedAmount"`
			RemainingAmount    int     `json:"remainingAmount"`
			UtilizationPercent float64 `json:"utilizationPercent"`
			WarnPercent        int     `json:"warnPercent"`
			HardStopEnabled    bool    `json:"hardStopEnabled"`
			NotifyEnabled      bool    `json:"notifyEnabled"`
			IsActive           bool    `json:"isActive"`
			Status             string  `json:"status"`
		}

		policies := make([]PolicySummary, 0, len(agents))
		for _, a := range agents {
			monthly := a.BudgetMonthlyCents
			spent := a.SpentMonthlyCents
			remaining := monthly - spent
			if remaining < 0 {
				remaining = 0
			}
			var utilPct float64
			if monthly > 0 {
				utilPct = float64(spent) / float64(monthly) * 100
			}
			status := "ok"
			if monthly > 0 && spent >= monthly {
				status = "hard_stop"
			}
			policies = append(policies, PolicySummary{
				PolicyID:           a.ID,
				CompanyID:          companyID,
				ScopeType:          "agent",
				ScopeID:            a.ID,
				ScopeName:          a.Name,
				Metric:             "billed_cents",
				WindowKind:         "calendar_month_utc",
				Amount:             monthly,
				ObservedAmount:     spent,
				RemainingAmount:    remaining,
				UtilizationPercent: utilPct,
				WarnPercent:        80,
				HardStopEnabled:    monthly > 0,
				NotifyEnabled:      monthly > 0,
				IsActive:           monthly > 0,
				Status:             status,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"policies":  policies,
			"incidents": []interface{}{},
		})
	}
}
