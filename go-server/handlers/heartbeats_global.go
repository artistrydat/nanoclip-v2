package handlers

import (
        "fmt"
        "net/http"
        "strings"
        "time"

        "github.com/gin-gonic/gin"
        "github.com/google/uuid"
        "gorm.io/gorm"
        "paperclip-go/models"
)

// GlobalRunRoutes are routes for heartbeat runs not scoped to a company
func GlobalRunRoutes(rg *gin.RouterGroup, db *gorm.DB) {
        rg.GET("/:runId", getGlobalRun(db))
        rg.GET("/:runId/events", getRunEvents(db))
        rg.GET("/:runId/log", getRunLog(db))
        rg.GET("/:runId/workspace-operations", getRunWorkspaceOps(db))
        rg.GET("/:runId/issues", getRunIssues(db))
        rg.POST("/:runId/cancel", cancelRun(db))
}

// GlobalLiveRunRoutes for /companies/:companyId/live-runs
func GlobalLiveRunRoutes(rg *gin.RouterGroup, db *gorm.DB) {
        rg.GET("", listLiveRuns(db))
}

func getGlobalRun(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var run models.HeartbeatRun
                if err := db.First(&run, "id = ?", c.Param("runId")).Error; err != nil {
                        c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
                        return
                }
                c.JSON(http.StatusOK, run)
        }
}

func getRunEvents(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var events []models.HeartbeatRunEvent
                db.Where("heartbeat_run_id = ?", c.Param("runId")).
                        Order("sequence_number asc").Find(&events)
                c.JSON(http.StatusOK, events)
        }
}

func getRunLog(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                runID := c.Param("runId")
                offset := 0
                limitBytes := 256000
                if v := c.Query("offset"); v != "" {
                        fmt.Sscanf(v, "%d", &offset)
                }
                if v := c.Query("limitBytes"); v != "" {
                        fmt.Sscanf(v, "%d", &limitBytes)
                }

                var events []models.HeartbeatRunEvent
                db.Where("heartbeat_run_id = ?", runID).
                        Order("sequence_number asc").Find(&events)

                var lines []string
                for _, e := range events {
                        ts := e.CreatedAt.Format(time.RFC3339)
                        lines = append(lines, ts+" "+e.Kind+": "+e.Summary)
                }
                fullLog := strings.Join(lines, "\n")
                if len(fullLog) > 0 {
                        fullLog += "\n"
                }

                total := len(fullLog)
                if offset > total {
                        offset = total
                }
                end := offset + limitBytes
                var nextOffset *int
                if end < total {
                        nextOffset = &end
                } else {
                        end = total
                }

                content := fullLog[offset:end]
                resp := gin.H{
                        "runId":   runID,
                        "store":   "db",
                        "logRef":  runID,
                        "content": content,
                }
                if nextOffset != nil {
                        resp["nextOffset"] = *nextOffset
                }
                c.JSON(http.StatusOK, resp)
        }
}

func getRunIssues(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                runID := c.Param("runId")
                type IssueForRun struct {
                        IssueID    string  `json:"issueId"`
                        Identifier *string `json:"identifier"`
                        Title      string  `json:"title"`
                }
                var results []IssueForRun
                db.Raw(`
                        SELECT i.id as issue_id, i.identifier, i.title
                        FROM issues i
                        JOIN heartbeat_run_events hre ON hre.issue_id = i.id
                        WHERE hre.heartbeat_run_id = ?
                        GROUP BY i.id, i.identifier, i.title
                `, runID).Scan(&results)
                if results == nil {
                        results = []IssueForRun{}
                }
                c.JSON(http.StatusOK, results)
        }
}

func getRunWorkspaceOps(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var ops []models.WorkspaceOperation
                db.Where("heartbeat_run_id = ?", c.Param("runId")).
                        Order("created_at asc").Find(&ops)
                c.JSON(http.StatusOK, ops)
        }
}

func cancelRun(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                now := time.Now()
                db.Model(&models.HeartbeatRun{}).Where("id = ?", c.Param("runId")).
                        Updates(map[string]interface{}{
                                "status":       "cancelled",
                                "completed_at": now,
                                "updated_at":   now,
                        })
                var run models.HeartbeatRun
                db.First(&run, "id = ?", c.Param("runId"))
                c.JSON(http.StatusOK, run)
        }
}

func listLiveRuns(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                companyID := c.Param("companyId")
                var runs []models.HeartbeatRun
                q := db.Where("company_id = ? AND status IN ('queued','running')", companyID).
                        Order("created_at desc").Limit(50)
                if agentID := c.Query("agentId"); agentID != "" {
                        q = q.Where("agent_id = ?", agentID)
                }
                q.Find(&runs)
                c.JSON(http.StatusOK, runs)
        }
}

// HeartbeatRunEventRoutes allows agents to submit run events
func HeartbeatRunEventRoutes(rg *gin.RouterGroup, db *gorm.DB) {
        rg.POST("", createRunEvent(db))
}

func createRunEvent(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var req struct {
                        Kind     string      `json:"kind" binding:"required"`
                        Summary  string      `json:"summary"`
                        Detail   models.JSON `json:"detail"`
                        IssueID  *string     `json:"issueId"`
                }
                if err := c.ShouldBindJSON(&req); err != nil {
                        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                        return
                }

                // Get current max sequence
                var maxSeq struct{ Max int }
                db.Model(&models.HeartbeatRunEvent{}).
                        Select("COALESCE(MAX(sequence_number),0) as max").
                        Where("heartbeat_run_id = ?", c.Param("runId")).
                        Scan(&maxSeq)

                event := models.HeartbeatRunEvent{
                        ID:             uuid.NewString(),
                        HeartbeatRunID: c.Param("runId"),
                        Kind:           req.Kind,
                        Summary:        req.Summary,
                        Detail:         req.Detail,
                        IssueID:        req.IssueID,
                        SequenceNumber: maxSeq.Max + 1,
                        CreatedAt:      time.Now(),
                }
                db.Create(&event)

                // Update run status if applicable
                if req.Kind == "run_started" {
                        db.Model(&models.HeartbeatRun{}).Where("id = ?", c.Param("runId")).
                                Updates(map[string]interface{}{"status": "running", "started_at": time.Now()})
                } else if req.Kind == "run_completed" || req.Kind == "run_failed" {
                        status := "completed"
                        if req.Kind == "run_failed" {
                                status = "failed"
                        }
                        db.Model(&models.HeartbeatRun{}).Where("id = ?", c.Param("runId")).
                                Updates(map[string]interface{}{
                                        "status":       status,
                                        "completed_at": time.Now(),
                                })
                }

                c.JSON(http.StatusCreated, event)
        }
}
