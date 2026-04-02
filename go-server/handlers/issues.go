package handlers

import (
        "fmt"
        "net/http"
        "os"
        "path/filepath"
        "strconv"
        "time"

        "github.com/gin-gonic/gin"
        "github.com/google/uuid"
        "gorm.io/gorm"
        "paperclip-go/middleware"
        "paperclip-go/models"
        "paperclip-go/services"
        "paperclip-go/ws"
)

func IssueRoutes(rg *gin.RouterGroup, db *gorm.DB, hub *ws.Hub, hb *services.HeartbeatService) {
        rg.GET("", listIssues(db))
        rg.POST("", createIssue(db, hub))
        rg.GET("/:issueId", getIssue(db))
        rg.PATCH("/:issueId", updateIssue(db, hub))
        rg.DELETE("/:issueId", deleteIssue(db))
        rg.POST("/:issueId/comments", addComment(db, hb))
        rg.GET("/:issueId/comments", listComments(db))
        rg.POST("/:issueId/cancel", cancelIssue(db, hub))
        rg.POST("/:issueId/attachments", uploadAttachment(db))
}

func listIssues(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                companyID := c.Param("companyId")
                q := db.Where("company_id = ? AND hidden_at IS NULL", companyID).Order("created_at desc")

                if status := c.Query("status"); status != "" {
                        q = q.Where("status = ?", status)
                }
                if projectID := c.Query("projectId"); projectID != "" {
                        q = q.Where("project_id = ?", projectID)
                }
                if assignee := c.Query("assigneeAgentId"); assignee != "" {
                        q = q.Where("assignee_agent_id = ?", assignee)
                }
                if participant := c.Query("participantAgentId"); participant != "" {
                        q = q.Where("assignee_agent_id = ? OR created_by_agent_id = ?", participant, participant)
                }
                // touchedByUserId=me  → issues that have any non-archived InboxItem for this company
                if touched := c.Query("touchedByUserId"); touched != "" {
                        q = q.Where("id IN (SELECT issue_id FROM inbox_items WHERE company_id = ? AND issue_id IS NOT NULL AND status != 'archived')", companyID)
                }
                // inboxArchivedByUserId=me → further exclude issues whose InboxItems are all archived
                if archived := c.Query("inboxArchivedByUserId"); archived != "" {
                        q = q.Where("id NOT IN (SELECT DISTINCT issue_id FROM inbox_items WHERE company_id = ? AND issue_id IS NOT NULL AND status = 'archived')", companyID)
                }
                // unreadForUserId=me → only issues with unread InboxItems (used for badge count)
                if unread := c.Query("unreadForUserId"); unread != "" {
                        q = q.Where("id IN (SELECT issue_id FROM inbox_items WHERE company_id = ? AND issue_id IS NOT NULL AND status = 'unread')", companyID)
                }
                if parent := c.Query("parentId"); parent != "" {
                        q = q.Where("parent_id = ?", parent)
                } else if c.Query("includeSubtasks") != "true" {
                        q = q.Where("parent_id IS NULL")
                }

                limitStr := c.DefaultQuery("limit", "100")
                limit, _ := strconv.Atoi(limitStr)
                if limit <= 0 || limit > 500 {
                        limit = 100
                }

                var issues []models.Issue
                q.Limit(limit).Find(&issues)
                c.JSON(http.StatusOK, issues)
        }
}

func getIssue(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var issue models.Issue
                if err := db.First(&issue, "id = ? AND company_id = ?",
                        c.Param("issueId"), c.Param("companyId")).Error; err != nil {
                        c.JSON(http.StatusNotFound, gin.H{"error": "issue not found"})
                        return
                }
                c.JSON(http.StatusOK, issue)
        }
}

type createIssueRequest struct {
        Title           string  `json:"title" binding:"required"`
        Description     *string `json:"description"`
        Status          *string `json:"status"`
        Priority        *string `json:"priority"`
        ProjectID       *string `json:"projectId"`
        GoalID          *string `json:"goalId"`
        ParentID        *string `json:"parentId"`
        AssigneeAgentID *string `json:"assigneeAgentId"`
        AssigneeUserID  *string `json:"assigneeUserId"`
        BillingCode     *string `json:"billingCode"`
}

func createIssue(db *gorm.DB, hub *ws.Hub) gin.HandlerFunc {
        return func(c *gin.Context) {
                companyID := c.Param("companyId")
                var req createIssueRequest
                if err := c.ShouldBindJSON(&req); err != nil {
                        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                        return
                }

                // Auto-increment issue number
                var company models.Company
                db.First(&company, "id = ?", companyID)
                db.Model(&company).Update("issue_counter", gorm.Expr("issue_counter + 1"))
                db.First(&company, "id = ?", companyID)
                issueNumber := company.IssueCounter
                identifier := fmt.Sprintf("%s-%d", company.IssuePrefix, issueNumber)

                status := "backlog"
                if req.Status != nil {
                        status = *req.Status
                }
                priority := "medium"
                if req.Priority != nil {
                        priority = *req.Priority
                }

                actor := middleware.GetActor(c)
                issue := models.Issue{
                        ID:              uuid.NewString(),
                        CompanyID:       companyID,
                        ProjectID:       req.ProjectID,
                        GoalID:          req.GoalID,
                        ParentID:        req.ParentID,
                        Title:           req.Title,
                        Description:     req.Description,
                        Status:          status,
                        Priority:        priority,
                        AssigneeAgentID: req.AssigneeAgentID,
                        AssigneeUserID:  req.AssigneeUserID,
                        IssueNumber:     &issueNumber,
                        Identifier:      &identifier,
                        BillingCode:     req.BillingCode,
                        OriginKind:      "manual",
                        CreatedAt:       time.Now(),
                        UpdatedAt:       time.Now(),
                }
                if actor != nil {
                        if actor.Type == "user" {
                                issue.CreatedByUserID = &actor.UserID
                        } else if actor.Type == "agent" {
                                issue.CreatedByAgentID = &actor.AgentID
                        }
                }

                if err := db.Create(&issue).Error; err != nil {
                        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                        return
                }

                hub.Publish(ws.LiveEvent{Type: "issue.created", Payload: issue})
                logActivity(db, companyID, actor, "created", "issue", issue.ID, nil)
                c.JSON(http.StatusCreated, issue)
        }
}

type updateIssueRequest struct {
        Title           *string `json:"title"`
        Description     *string `json:"description"`
        Status          *string `json:"status"`
        Priority        *string `json:"priority"`
        AssigneeAgentID *string `json:"assigneeAgentId"`
        AssigneeUserID  *string `json:"assigneeUserId"`
        ProjectID       *string `json:"projectId"`
        GoalID          *string `json:"goalId"`
}

func updateIssue(db *gorm.DB, hub *ws.Hub) gin.HandlerFunc {
        return func(c *gin.Context) {
                var issue models.Issue
                if err := db.First(&issue, "id = ? AND company_id = ?",
                        c.Param("issueId"), c.Param("companyId")).Error; err != nil {
                        c.JSON(http.StatusNotFound, gin.H{"error": "issue not found"})
                        return
                }
                var req updateIssueRequest
                c.ShouldBindJSON(&req)

                updates := map[string]interface{}{"updated_at": time.Now()}
                if req.Title != nil {
                        updates["title"] = *req.Title
                }
                if req.Description != nil {
                        updates["description"] = req.Description
                }
                if req.Status != nil {
                        updates["status"] = *req.Status
                        now := time.Now()
                        switch *req.Status {
                        case "in_progress":
                                if issue.StartedAt == nil {
                                        updates["started_at"] = now
                                }
                        case "done":
                                updates["completed_at"] = now
                        case "cancelled":
                                updates["cancelled_at"] = now
                        }
                }
                if req.Priority != nil {
                        updates["priority"] = *req.Priority
                }
                if req.AssigneeAgentID != nil {
                        updates["assignee_agent_id"] = req.AssigneeAgentID
                }
                if req.AssigneeUserID != nil {
                        updates["assignee_user_id"] = req.AssigneeUserID
                }
                if req.ProjectID != nil {
                        updates["project_id"] = req.ProjectID
                }
                if req.GoalID != nil {
                        updates["goal_id"] = req.GoalID
                }

                db.Model(&issue).Updates(updates)
                db.First(&issue, "id = ?", issue.ID)

                hub.Publish(ws.LiveEvent{Type: "issue.updated", Payload: issue})
                actor := middleware.GetActor(c)
                logActivity(db, issue.CompanyID, actor, "updated", "issue", issue.ID, nil)
                c.JSON(http.StatusOK, issue)
        }
}

func deleteIssue(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                now := time.Now()
                db.Model(&models.Issue{}).Where("id = ? AND company_id = ?",
                        c.Param("issueId"), c.Param("companyId")).
                        Updates(map[string]interface{}{"hidden_at": now, "updated_at": now})
                c.JSON(http.StatusOK, gin.H{"success": true})
        }
}

func cancelIssue(db *gorm.DB, hub *ws.Hub) gin.HandlerFunc {
        return func(c *gin.Context) {
                now := time.Now()
                db.Model(&models.Issue{}).Where("id = ? AND company_id = ?",
                        c.Param("issueId"), c.Param("companyId")).
                        Updates(map[string]interface{}{
                                "status":       "cancelled",
                                "cancelled_at": now,
                                "updated_at":   now,
                        })
                var issue models.Issue
                db.First(&issue, "id = ?", c.Param("issueId"))
                hub.Publish(ws.LiveEvent{Type: "issue.updated", Payload: issue})
                c.JSON(http.StatusOK, issue)
        }
}

func addComment(db *gorm.DB, hb *services.HeartbeatService) gin.HandlerFunc {
        return func(c *gin.Context) {
                var req struct {
                        Body string `json:"body" binding:"required"`
                }
                if err := c.ShouldBindJSON(&req); err != nil {
                        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                        return
                }
                actor := middleware.GetActor(c)
                companyID := c.Param("companyId")
                issueID := c.Param("issueId")
                comment := models.IssueComment{
                        ID:        uuid.NewString(),
                        CompanyID: companyID,
                        IssueID:   issueID,
                        Body:      req.Body,
                        CreatedAt: time.Now(),
                        UpdatedAt: time.Now(),
                }
                fromUser := false
                if actor != nil {
                        if actor.Type == "user" {
                                comment.AuthorUserID = &actor.UserID
                                fromUser = true
                        } else if actor.Type == "agent" {
                                comment.AuthorAgentID = &actor.AgentID
                        }
                }
                db.Create(&comment)
                c.JSON(http.StatusCreated, comment)

                // Trigger an agent run when a user (not agent) posts a comment
                // and the issue has an assigned agent
                if fromUser && hb != nil {
                        var issue models.Issue
                        if err := db.First(&issue, "id = ? AND company_id = ?", issueID, companyID).Error; err == nil {
                                if issue.AssigneeAgentID != nil && *issue.AssigneeAgentID != "" {
                                        go hb.TriggerRun(*issue.AssigneeAgentID, companyID, &issue)
                                }
                        }
                }
        }
}

func listComments(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                var comments []models.IssueComment
                db.Where("issue_id = ? AND company_id = ?",
                        c.Param("issueId"), c.Param("companyId")).
                        Order("created_at asc").Find(&comments)
                c.JSON(http.StatusOK, comments)
        }
}

func uploadAttachment(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                companyID := c.Param("companyId")
                issueID := c.Param("issueId")

                var issue models.Issue
                if err := db.Where("id = ? AND company_id = ?", issueID, companyID).First(&issue).Error; err != nil {
                        c.JSON(http.StatusNotFound, gin.H{"error": "issue not found"})
                        return
                }

                fh, err := c.FormFile("file")
                if err != nil {
                        c.JSON(http.StatusBadRequest, gin.H{"error": "file field required"})
                        return
                }

                storageDir := os.ExpandEnv("$HOME/.paperclip-go/attachments")
                if err := os.MkdirAll(storageDir, 0755); err != nil {
                        c.JSON(http.StatusInternalServerError, gin.H{"error": "storage unavailable"})
                        return
                }

                attachmentID := uuid.NewString()
                ext := filepath.Ext(fh.Filename)
                storePath := filepath.Join(storageDir, attachmentID+ext)

                if err := c.SaveUploadedFile(fh, storePath); err != nil {
                        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
                        return
                }

                mimeType := fh.Header.Get("Content-Type")
                if mimeType == "" {
                        mimeType = "application/octet-stream"
                }

                attachment := models.IssueAttachment{
                        ID:        attachmentID,
                        CompanyID: companyID,
                        IssueID:   issueID,
                        Filename:  fh.Filename,
                        MimeType:  mimeType,
                        SizeBytes: fh.Size,
                        StorePath: storePath,
                }
                db.Create(&attachment)

                c.JSON(http.StatusCreated, attachment)
        }
}
