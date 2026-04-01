package handlers

import (
        "net/http"
        "time"

        "github.com/gin-gonic/gin"
        "gorm.io/gorm"
        "paperclip-go/models"
)

func CostRoutes(rg *gin.RouterGroup, db *gorm.DB) {
        rg.GET("", listCosts(db))
        rg.GET("/summary", costSummary(db))
        rg.GET("/by-agent", costByAgent(db))
        rg.GET("/by-agent-model", costByAgentModel(db))
        rg.GET("/by-project", costByProject(db))
        rg.GET("/by-provider", costByProvider(db))
        rg.GET("/by-biller", costByBiller(db))
        rg.GET("/window-spend", costWindowSpend(db))
        rg.GET("/quota-windows", costQuotaWindows(db))
        rg.GET("/finance-summary", financeSummary(db))
        rg.GET("/finance-by-biller", financeByBiller(db))
        rg.GET("/finance-by-kind", financeByKind(db))
        rg.GET("/finance-events", financeEvents(db))
}

func parseCostTimeRange(c *gin.Context) (from time.Time, to time.Time, hasFrom bool, hasTo bool) {
        if s := c.Query("from"); s != "" {
                if t, err := time.Parse(time.RFC3339, s); err == nil {
                        from, hasFrom = t, true
                }
        }
        if s := c.Query("to"); s != "" {
                if t, err := time.Parse(time.RFC3339, s); err == nil {
                        to, hasTo = t, true
                }
        }
        return
}

func applyCostDateFilter(q *gorm.DB, companyID string, from time.Time, to time.Time, hasFrom, hasTo bool) *gorm.DB {
        q = q.Where("cost_events.company_id = ?", companyID)
        if hasFrom {
                q = q.Where("cost_events.occurred_at >= ?", from)
        }
        if hasTo {
                q = q.Where("cost_events.occurred_at <= ?", to)
        }
        return q
}

func listCosts(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                companyID := c.Param("companyId")
                var events []models.CostEvent
                q := db.Where("company_id = ?", companyID).Order("occurred_at desc").Limit(200)
                if agentID := c.Query("agentId"); agentID != "" {
                        q = q.Where("agent_id = ?", agentID)
                }
                if runID := c.Query("heartbeatRunId"); runID != "" {
                        q = q.Where("heartbeat_run_id = ?", runID)
                }
                q.Find(&events)
                c.JSON(http.StatusOK, events)
        }
}

func costSummary(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                companyID := c.Param("companyId")
                now := time.Now().UTC()
                startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

                var monthlyByAgent []struct {
                        AgentID    string `json:"agentId"`
                        TotalCents int64  `json:"totalCents"`
                        Runs       int64  `json:"runs"`
                }
                db.Model(&models.CostEvent{}).
                        Select("agent_id, SUM(cost_cents) as total_cents, COUNT(DISTINCT heartbeat_run_id) as runs").
                        Where("company_id = ? AND occurred_at >= ?", companyID, startOfMonth).
                        Group("agent_id").
                        Scan(&monthlyByAgent)

                var totalMonthly struct{ Total int64 }
                db.Model(&models.CostEvent{}).
                        Select("COALESCE(SUM(cost_cents), 0) as total").
                        Where("company_id = ? AND occurred_at >= ?", companyID, startOfMonth).
                        Scan(&totalMonthly)

                var totalAllTime struct{ Total int64 }
                db.Model(&models.CostEvent{}).
                        Select("COALESCE(SUM(cost_cents), 0) as total").
                        Where("company_id = ?", companyID).
                        Scan(&totalAllTime)

                c.JSON(http.StatusOK, gin.H{
                        "monthlyTotalCents": totalMonthly.Total,
                        "allTimeTotalCents": totalAllTime.Total,
                        "byAgent":           monthlyByAgent,
                        "periodStart":       startOfMonth,
                })
        }
}

func costByAgent(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                companyID := c.Param("companyId")
                from, to, hasFrom, hasTo := parseCostTimeRange(c)

                type row struct {
                        AgentID    string `json:"agentId"`
                        AgentName  string `json:"agentName"`
                        CostCents  int64  `json:"costCents"`
                        InputTokens  int64 `json:"inputTokens"`
                        OutputTokens int64 `json:"outputTokens"`
                }

                var results []row
                q := applyCostDateFilter(db.Model(&models.CostEvent{}), companyID, from, to, hasFrom, hasTo)
                q.Select("cost_events.agent_id, COALESCE(agents.name, '') as agent_name, SUM(cost_events.cost_cents) as cost_cents, SUM(cost_events.input_tokens) as input_tokens, SUM(cost_events.output_tokens) as output_tokens").
                        Joins("LEFT JOIN agents ON agents.id = cost_events.agent_id").
                        Group("cost_events.agent_id").
                        Order("cost_cents DESC").
                        Scan(&results)

                if results == nil {
                        results = []row{}
                }
                c.JSON(http.StatusOK, results)
        }
}

func costByAgentModel(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                companyID := c.Param("companyId")
                from, to, hasFrom, hasTo := parseCostTimeRange(c)

                type row struct {
                        AgentID      string `json:"agentId"`
                        AgentName    string `json:"agentName"`
                        Model        string `json:"model"`
                        CostCents    int64  `json:"costCents"`
                        InputTokens  int64  `json:"inputTokens"`
                        OutputTokens int64  `json:"outputTokens"`
                }

                var results []row
                q := applyCostDateFilter(db.Model(&models.CostEvent{}), companyID, from, to, hasFrom, hasTo)
                q.Select("cost_events.agent_id, COALESCE(agents.name, '') as agent_name, cost_events.model, SUM(cost_events.cost_cents) as cost_cents, SUM(cost_events.input_tokens) as input_tokens, SUM(cost_events.output_tokens) as output_tokens").
                        Joins("LEFT JOIN agents ON agents.id = cost_events.agent_id").
                        Group("cost_events.agent_id, cost_events.model").
                        Order("cost_cents DESC").
                        Scan(&results)

                if results == nil {
                        results = []row{}
                }
                c.JSON(http.StatusOK, results)
        }
}

func costByProject(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                companyID := c.Param("companyId")
                from, to, hasFrom, hasTo := parseCostTimeRange(c)

                type row struct {
                        ProjectID   string `json:"projectId"`
                        ProjectName string `json:"projectName"`
                        CostCents   int64  `json:"costCents"`
                }

                var results []row
                q := applyCostDateFilter(db.Model(&models.CostEvent{}), companyID, from, to, hasFrom, hasTo).
                        Where("cost_events.project_id IS NOT NULL AND cost_events.project_id != ''")
                q.Select("cost_events.project_id, COALESCE(projects.name, '') as project_name, SUM(cost_events.cost_cents) as cost_cents").
                        Joins("LEFT JOIN projects ON projects.id = cost_events.project_id").
                        Group("cost_events.project_id").
                        Order("cost_cents DESC").
                        Scan(&results)

                if results == nil {
                        results = []row{}
                }
                c.JSON(http.StatusOK, results)
        }
}

func costByProvider(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                companyID := c.Param("companyId")
                from, to, hasFrom, hasTo := parseCostTimeRange(c)

                type row struct {
                        Provider     string `json:"provider"`
                        Model        string `json:"model"`
                        CostCents    int64  `json:"costCents"`
                        InputTokens  int64  `json:"inputTokens"`
                        CachedInputTokens int64 `json:"cachedInputTokens"`
                        OutputTokens int64  `json:"outputTokens"`
                }

                var results []row
                q := applyCostDateFilter(db.Model(&models.CostEvent{}), companyID, from, to, hasFrom, hasTo)
                q.Select("provider, model, SUM(cost_cents) as cost_cents, SUM(input_tokens) as input_tokens, 0 as cached_input_tokens, SUM(output_tokens) as output_tokens").
                        Group("provider, model").
                        Order("cost_cents DESC").
                        Scan(&results)

                if results == nil {
                        results = []row{}
                }
                c.JSON(http.StatusOK, results)
        }
}

func costByBiller(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                companyID := c.Param("companyId")
                from, to, hasFrom, hasTo := parseCostTimeRange(c)

                type row struct {
                        BillerID     string `json:"billerId"`
                        BillerName   string `json:"billerName"`
                        CostCents    int64  `json:"costCents"`
                        InputTokens  int64  `json:"inputTokens"`
                        CachedInputTokens int64 `json:"cachedInputTokens"`
                        OutputTokens int64  `json:"outputTokens"`
                }

                var results []row
                q := applyCostDateFilter(db.Model(&models.CostEvent{}), companyID, from, to, hasFrom, hasTo)
                q.Select("provider as biller_id, provider as biller_name, SUM(cost_cents) as cost_cents, SUM(input_tokens) as input_tokens, 0 as cached_input_tokens, SUM(output_tokens) as output_tokens").
                        Group("provider").
                        Order("cost_cents DESC").
                        Scan(&results)

                if results == nil {
                        results = []row{}
                }
                c.JSON(http.StatusOK, results)
        }
}

func costWindowSpend(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                companyID := c.Param("companyId")

                type row struct {
                        WindowStart string `json:"windowStart"`
                        CostCents   int64  `json:"costCents"`
                }

                var results []row
                db.Model(&models.CostEvent{}).
                        Select("strftime('%Y-%m-%dT%H:00:00Z', occurred_at) as window_start, SUM(cost_cents) as cost_cents").
                        Where("company_id = ? AND occurred_at >= ?", companyID, time.Now().UTC().AddDate(0, 0, -30)).
                        Group("window_start").
                        Order("window_start ASC").
                        Scan(&results)

                if results == nil {
                        results = []row{}
                }
                c.JSON(http.StatusOK, results)
        }
}

func costQuotaWindows(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                c.JSON(http.StatusOK, []gin.H{})
        }
}

func financeSummary(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                companyID := c.Param("companyId")
                from, to, hasFrom, hasTo := parseCostTimeRange(c)

                var totalCents int64
                q := applyCostDateFilter(db.Model(&models.CostEvent{}), companyID, from, to, hasFrom, hasTo)
                var res struct{ Total int64 }
                q.Select("COALESCE(SUM(cost_cents), 0) as total").Scan(&res)
                totalCents = res.Total

                c.JSON(http.StatusOK, gin.H{
                        "creditCents":          int64(0),
                        "debitCents":           totalCents,
                        "netCents":             -totalCents,
                        "eventCount":           0,
                        "estimatedDebitCents":  int64(0),
                })
        }
}

func financeByBiller(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                c.JSON(http.StatusOK, []gin.H{})
        }
}

func financeByKind(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                c.JSON(http.StatusOK, []gin.H{})
        }
}

func financeEvents(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                c.JSON(http.StatusOK, []gin.H{})
        }
}
