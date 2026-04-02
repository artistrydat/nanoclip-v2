package services

import (
        "bytes"
        "encoding/json"
        "fmt"
        "io"
        "log"
        "net/http"
        "strings"
        "sync"
        "time"

        "github.com/google/uuid"
        "gorm.io/gorm"
        "paperclip-go/models"
        "paperclip-go/ws"
)

const telegramPluginKey = "telegram-bot"
const telegramPackageName = "paperclip-plugin-telegram"

func TelegramPluginKey() string        { return telegramPluginKey }
func TelegramPluginPackageName() string { return telegramPackageName }

func TelegramManifest() map[string]interface{} {
        return map[string]interface{}{
                "name":        "Telegram Bot",
                "packageName": telegramPackageName,
                "pluginKey":   telegramPluginKey,
                "version":     "0.5.0",
                "description": "Send notifications, approvals, escalations, and accept commands via Telegram.",
                "categories":  []string{"connector", "automation"},
                "configSchema": map[string]interface{}{
                        "botToken":           map[string]interface{}{"type": "string", "label": "Bot Token", "secret": true},
                        "defaultChatId":      map[string]interface{}{"type": "string", "label": "Default Chat ID"},
                        "approvalsChatId":    map[string]interface{}{"type": "string", "label": "Approvals Chat ID"},
                        "errorsChatId":       map[string]interface{}{"type": "string", "label": "Errors Chat ID"},
                        "escalationChatId":   map[string]interface{}{"type": "string", "label": "Escalation Chat ID"},
                        "paperclipPublicUrl": map[string]interface{}{"type": "string", "label": "Public URL"},
                        "enableCommands":     map[string]interface{}{"type": "boolean", "label": "Enable Commands"},
                        "enableInbound":      map[string]interface{}{"type": "boolean", "label": "Enable Inbound Messages"},
                        "enableEscalation":   map[string]interface{}{"type": "boolean", "label": "Enable Escalation"},
                        "enableMedia":        map[string]interface{}{"type": "boolean", "label": "Enable Media Handling"},
                        "topicsEnabled":      map[string]interface{}{"type": "boolean", "label": "Enable Topics Routing"},
                },
        }
}

// ─── Config ───────────────────────────────────────────────────────────────────

type TelegramConfig struct {
        BotToken           string `json:"botToken"`
        DefaultChatID      string `json:"defaultChatId"`
        ApprovalsChatID    string `json:"approvalsChatId"`
        ErrorsChatID       string `json:"errorsChatId"`
        EscalationChatID   string `json:"escalationChatId"`
        PaperclipPublicURL string `json:"paperclipPublicUrl"`
        EnableCommands     bool   `json:"enableCommands"`
        EnableInbound      bool   `json:"enableInbound"`
        EnableEscalation   bool   `json:"enableEscalation"`
        EnableMedia        bool   `json:"enableMedia"`
        TopicsEnabled      bool   `json:"topicsEnabled"`
}

// ─── Registry types ───────────────────────────────────────────────────────────

// chatConn stores which company/project a chat or thread is connected to.
type chatConn struct {
        CompanyID string
        ProjectID string
}

// escalationRecord tracks a pending escalation waiting for a human decision.
type escalationRecord struct {
        ID            string
        AgentName     string
        Reasoning     string
        SuggestedReply string
        ChatID        int64
        MessageID     int
        TimeoutAt     time.Time
        DefaultAction string // "defer" | "auto_reply" | "close"
        Status        string // "pending" | "resolved" | "timed_out"
}

// watchEntry is one per-chat subscription.
type watchEntry struct {
        EventTypes map[string]bool // e.g. {"issue.created": true}
        CompanyID  string
        ThreadID   int
}

// ─── Service ──────────────────────────────────────────────────────────────────

type TelegramService struct {
        db     *gorm.DB
        hub    *ws.Hub
        mu     sync.RWMutex
        cfg    *TelegramConfig
        plugin *models.Plugin
        stop   chan struct{}
        wg     sync.WaitGroup

        // registries (in-memory, reset on restart)
        connMu      sync.RWMutex
        connections map[string]chatConn // key: "chatID" or "chatID:threadID"

        watchMu sync.RWMutex
        watches map[string]*watchEntry // key: "chatID" or "chatID:threadID"

        escalMu    sync.RWMutex
        escalations map[string]*escalationRecord
}

var GlobalTelegramService *TelegramService

func NewTelegramService(db *gorm.DB, hub *ws.Hub) *TelegramService {
        s := &TelegramService{
                db:          db,
                hub:         hub,
                stop:        make(chan struct{}),
                connections: make(map[string]chatConn),
                watches:     make(map[string]*watchEntry),
                escalations: make(map[string]*escalationRecord),
        }
        GlobalTelegramService = s
        return s
}

func (s *TelegramService) Reload() {
        var plugin models.Plugin
        if err := s.db.Where("plugin_key = ?", telegramPluginKey).First(&plugin).Error; err != nil {
                return
        }
        if !plugin.Enabled || plugin.Status != "ready" {
                return
        }
        cfg := s.extractConfig(&plugin)
        if cfg == nil || cfg.BotToken == "" {
                return
        }
        s.mu.Lock()
        s.cfg = cfg
        s.plugin = &plugin
        s.mu.Unlock()
}

func (s *TelegramService) Start() {
        s.Reload()
        eventCh := s.hub.Subscribe()

        s.wg.Add(3)
        go s.pollLoop()
        go s.eventLoop(eventCh)
        go s.escalationWatchdog()
}

func (s *TelegramService) Stop() {
        close(s.stop)
        s.wg.Wait()
}

func (s *TelegramService) extractConfig(plugin *models.Plugin) *TelegramConfig {
        if plugin.Config == nil {
                return nil
        }
        raw, err := json.Marshal(plugin.Config)
        if err != nil {
                return nil
        }
        var cfg TelegramConfig
        if err := json.Unmarshal(raw, &cfg); err != nil {
                return nil
        }
        return &cfg
}

// ─── Telegram update types ────────────────────────────────────────────────────

type tgUpdate struct {
        UpdateID      int         `json:"update_id"`
        Message       *tgMessage  `json:"message"`
        CallbackQuery *tgCallback `json:"callback_query"`
}

type tgMessage struct {
        MessageID      int       `json:"message_id"`
        MessageThreadID int      `json:"message_thread_id"`
        From           *tgUser  `json:"from"`
        Chat           tgChat   `json:"chat"`
        Text           string   `json:"text"`
        Photo          []tgPhotoSize `json:"photo"`
        Document       *tgDocument   `json:"document"`
        Caption        string   `json:"caption"`
        ReplyToMessage *struct {
                MessageID int    `json:"message_id"`
                Text      string `json:"text"`
        } `json:"reply_to_message"`
}

type tgUser struct {
        ID       int    `json:"id"`
        Username string `json:"username"`
        FirstName string `json:"first_name"`
}

type tgChat struct {
        ID int64 `json:"id"`
}

type tgPhotoSize struct {
        FileID   string `json:"file_id"`
        Width    int    `json:"width"`
        Height   int    `json:"height"`
        FileSize int    `json:"file_size"`
}

type tgDocument struct {
        FileID   string `json:"file_id"`
        FileName string `json:"file_name"`
        MimeType string `json:"mime_type"`
        FileSize int    `json:"file_size"`
}

type tgCallback struct {
        ID      string   `json:"id"`
        From    tgUser   `json:"from"`
        Message *struct {
                MessageID int    `json:"message_id"`
                Chat      tgChat `json:"chat"`
        } `json:"message"`
        Data string `json:"data"`
}

// ─── Poll loop ────────────────────────────────────────────────────────────────

func (s *TelegramService) pollLoop() {
        defer s.wg.Done()
        offset := 0
        for {
                select {
                case <-s.stop:
                        return
                default:
                }
                s.mu.RLock()
                cfg := s.cfg
                plugin := s.plugin
                s.mu.RUnlock()

                if cfg == nil || cfg.BotToken == "" {
                        select {
                        case <-s.stop:
                                return
                        case <-time.After(10 * time.Second):
                                s.Reload()
                                continue
                        }
                }

                updates, newOffset, err := s.getUpdates(cfg.BotToken, offset)
                if err != nil {
                        s.logPlugin(plugin, "warn", fmt.Sprintf("getUpdates error: %v", err))
                        select {
                        case <-s.stop:
                                return
                        case <-time.After(5 * time.Second):
                                continue
                        }
                }
                offset = newOffset
                for _, upd := range updates {
                        s.handleUpdate(cfg, plugin, upd)
                }
        }
}

// ─── Event loop ───────────────────────────────────────────────────────────────

func (s *TelegramService) eventLoop(eventCh chan ws.LiveEvent) {
        defer s.wg.Done()
        for {
                select {
                case <-s.stop:
                        return
                case event, ok := <-eventCh:
                        if !ok {
                                return
                        }
                        s.mu.RLock()
                        cfg := s.cfg
                        plugin := s.plugin
                        s.mu.RUnlock()
                        if cfg == nil || cfg.BotToken == "" {
                                continue
                        }
                        s.handleEvent(cfg, plugin, event)
                }
        }
}

// ─── Escalation watchdog ──────────────────────────────────────────────────────

func (s *TelegramService) escalationWatchdog() {
        defer s.wg.Done()
        ticker := time.NewTicker(15 * time.Second)
        defer ticker.Stop()
        for {
                select {
                case <-s.stop:
                        return
                case <-ticker.C:
                        s.mu.RLock()
                        cfg := s.cfg
                        s.mu.RUnlock()
                        if cfg == nil {
                                continue
                        }
                        s.escalMu.Lock()
                        now := time.Now()
                        for id, esc := range s.escalations {
                                if esc.Status == "pending" && now.After(esc.TimeoutAt) {
                                        esc.Status = "timed_out"
                                        s.escalations[id] = esc
                                        go s.handleEscalationTimeout(cfg, esc)
                                }
                        }
                        s.escalMu.Unlock()
                }
        }
}

func (s *TelegramService) handleEscalationTimeout(cfg *TelegramConfig, esc *escalationRecord) {
        msg := fmt.Sprintf("⏱ *Escalation timed out* — %s\nDefault action: `%s`",
                escMD(esc.AgentName), esc.DefaultAction)
        if esc.MessageID > 0 {
                s.editMessage(cfg.BotToken, esc.ChatID, esc.MessageID, msg)
        } else {
                s.sendMsg(cfg.BotToken, esc.ChatID, 0, msg, nil)
        }
}

// ─── Update dispatching ───────────────────────────────────────────────────────

func (s *TelegramService) handleUpdate(cfg *TelegramConfig, plugin *models.Plugin, upd tgUpdate) {
        if upd.CallbackQuery != nil {
                s.handleCallback(cfg, plugin, upd.CallbackQuery)
                return
        }
        if upd.Message == nil {
                return
        }
        msg := upd.Message
        chatID := msg.Chat.ID
        threadID := msg.MessageThreadID
        text := strings.TrimSpace(msg.Text)

        // Media pipeline
        if cfg.EnableMedia && (len(msg.Photo) > 0 || msg.Document != nil) {
                s.handleMedia(cfg, plugin, msg)
                return
        }

        // Reply routing → issue comments
        if cfg.EnableInbound && msg.ReplyToMessage != nil {
                s.routeReply(cfg, plugin, chatID, threadID, msg.ReplyToMessage.Text, text)
                return
        }

        // Commands
        if cfg.EnableCommands && strings.HasPrefix(text, "/") {
                s.sendTyping(cfg.BotToken, chatID, threadID)
                s.handleCommand(cfg, plugin, chatID, threadID, msg.From, text)
        }
}

// ─── Callback handling (approvals + escalations) ──────────────────────────────

func (s *TelegramService) handleCallback(cfg *TelegramConfig, plugin *models.Plugin, cq *tgCallback) {
        parts := strings.SplitN(cq.Data, ":", 2)
        if len(parts) != 2 {
                s.answerCallback(cfg.BotToken, cq.ID, "Unknown action")
                return
        }
        action, id := parts[0], parts[1]

        // Escalation responses
        if action == "escal_reply" || action == "escal_override" || action == "escal_dismiss" {
                s.handleEscalationCallback(cfg, plugin, cq, action, id)
                return
        }

        // Approval responses
        var approval models.Approval
        if err := s.db.First(&approval, "id = ?", id).Error; err != nil {
                s.answerCallback(cfg.BotToken, cq.ID, "Approval not found")
                return
        }
        now := time.Now()
        actor := fmt.Sprintf("@%s (Telegram)", cq.From.Username)
        chatID := int64(0)
        messageID := 0
        if cq.Message != nil {
                chatID = cq.Message.Chat.ID
                messageID = cq.Message.MessageID
        }
        if action == "approve" {
                s.db.Model(&approval).Updates(map[string]interface{}{
                        "status": "approved", "decided_at": now, "decision_note": "approved by " + actor, "updated_at": now,
                })
                s.answerCallback(cfg.BotToken, cq.ID, "Approved!")
                if messageID > 0 {
                        s.editMessage(cfg.BotToken, chatID, messageID, "✅ Approval *APPROVED* by "+escMD(actor))
                }
                s.logPlugin(plugin, "info", fmt.Sprintf("approval %s approved by %s via Telegram", id, actor))
        } else if action == "reject" {
                s.db.Model(&approval).Updates(map[string]interface{}{
                        "status": "rejected", "decided_at": now, "decision_note": "rejected by " + actor, "updated_at": now,
                })
                s.answerCallback(cfg.BotToken, cq.ID, "Rejected.")
                if messageID > 0 {
                        s.editMessage(cfg.BotToken, chatID, messageID, "❌ Approval *REJECTED* by "+escMD(actor))
                }
                s.logPlugin(plugin, "info", fmt.Sprintf("approval %s rejected by %s via Telegram", id, actor))
        } else {
                s.answerCallback(cfg.BotToken, cq.ID, "Unknown action")
        }
}

func (s *TelegramService) handleEscalationCallback(cfg *TelegramConfig, plugin *models.Plugin, cq *tgCallback, action, escalID string) {
        s.escalMu.Lock()
        esc, ok := s.escalations[escalID]
        if !ok || esc.Status != "pending" {
                s.escalMu.Unlock()
                s.answerCallback(cfg.BotToken, cq.ID, "Escalation already resolved or not found")
                return
        }
        esc.Status = "resolved"
        s.escalMu.Unlock()

        actor := "@" + cq.From.Username
        chatID := int64(0)
        messageID := 0
        if cq.Message != nil {
                chatID = cq.Message.Chat.ID
                messageID = cq.Message.MessageID
        }

        var response string
        switch action {
        case "escal_reply":
                response = fmt.Sprintf("💬 *Reply sent* by %s", escMD(actor))
                if esc.SuggestedReply != "" {
                        response += "\n_" + escMD(esc.SuggestedReply) + "_"
                }
                s.answerCallback(cfg.BotToken, cq.ID, "Reply sent")
        case "escal_override":
                response = fmt.Sprintf("✏️ *Override* by %s — reply to this message with your response", escMD(actor))
                s.answerCallback(cfg.BotToken, cq.ID, "Reply to this message with your override")
        case "escal_dismiss":
                response = fmt.Sprintf("🚫 *Dismissed* by %s", escMD(actor))
                s.answerCallback(cfg.BotToken, cq.ID, "Dismissed")
        }
        if messageID > 0 {
                s.editMessage(cfg.BotToken, chatID, messageID, response)
        }
        s.logPlugin(plugin, "info", fmt.Sprintf("escalation %s %s by %s", escalID, action, actor))
}

// ─── Reply routing ────────────────────────────────────────────────────────────

func (s *TelegramService) routeReply(cfg *TelegramConfig, plugin *models.Plugin, chatID int64, threadID int, quotedText, replyText string) {
        issueID := extractIssueIDFromText(quotedText)
        if issueID == "" {
                return
        }
        var issue models.Issue
        if err := s.db.Where("id = ? OR identifier = ?", issueID, issueID).First(&issue).Error; err != nil {
                return
        }
        comment := models.IssueComment{
                ID:        uuid.NewString(),
                CompanyID: issue.CompanyID,
                IssueID:   issue.ID,
                Body:      replyText + "\n\n_(via Telegram)_",
                CreatedAt: time.Now(),
                UpdatedAt: time.Now(),
        }
        s.db.Create(&comment)
        identifier := ""
        if issue.Identifier != nil {
                identifier = *issue.Identifier
        }
        s.sendMsg(cfg.BotToken, chatID, threadID, "✅ Comment added to *"+escMD(identifier)+"*", nil)
        s.logPlugin(plugin, "info", fmt.Sprintf("reply routed to issue %s as comment", identifier))
}

// ─── Media pipeline ───────────────────────────────────────────────────────────

func (s *TelegramService) handleMedia(cfg *TelegramConfig, plugin *models.Plugin, msg *tgMessage) {
        chatID := msg.Chat.ID
        threadID := msg.MessageThreadID
        caption := strings.TrimSpace(msg.Caption)

        var fileDesc string
        if len(msg.Photo) > 0 {
                largest := msg.Photo[len(msg.Photo)-1]
                fileDesc = fmt.Sprintf("photo (file_id: %s, %dx%d)", largest.FileID, largest.Width, largest.Height)
        } else if msg.Document != nil {
                fileDesc = fmt.Sprintf("document: %s (file_id: %s, mime: %s)", msg.Document.FileName, msg.Document.FileID, msg.Document.MimeType)
        }

        // If caption references an issue ID, attach a comment with the media info
        issueID := extractIssueIDFromText(caption)
        if issueID != "" {
                var issue models.Issue
                if err := s.db.Where("id = ? OR identifier = ?", issueID, issueID).First(&issue).Error; err == nil {
                        body := fmt.Sprintf("📎 Media received via Telegram: %s", fileDesc)
                        if caption != "" {
                                body += "\n\n" + caption
                        }
                        comment := models.IssueComment{
                                ID:        uuid.NewString(),
                                CompanyID: issue.CompanyID,
                                IssueID:   issue.ID,
                                Body:      body,
                                CreatedAt: time.Now(),
                                UpdatedAt: time.Now(),
                        }
                        s.db.Create(&comment)
                        identifier := ""
                        if issue.Identifier != nil {
                                identifier = *issue.Identifier
                        }
                        s.sendMsg(cfg.BotToken, chatID, threadID, "📎 Media attached to *"+escMD(identifier)+"*", nil)
                        s.logPlugin(plugin, "info", "media attached to issue "+issueID)
                        return
                }
        }

        // No issue reference — acknowledge receipt
        s.sendMsg(cfg.BotToken, chatID, threadID,
                "📎 Media received\\. To attach it to an issue, send with a caption like: `PROJ\\-42 screenshot of error`", nil)
}

// ─── Command routing ──────────────────────────────────────────────────────────

func (s *TelegramService) handleCommand(cfg *TelegramConfig, plugin *models.Plugin, chatID int64, threadID int, from *tgUser, text string) {
        parts := strings.Fields(text)
        cmdRaw := strings.ToLower(parts[0])
        cmd := strings.SplitN(strings.TrimPrefix(cmdRaw, "/"), "@", 2)[0]
        args := ""
        if len(parts) > 1 {
                args = strings.Join(parts[1:], " ")
        }

        switch cmd {
        case "start", "help":
                s.cmdHelp(cfg, chatID, threadID)
        case "status":
                s.cmdStatus(cfg, plugin, chatID, threadID)
        case "issues":
                s.cmdIssues(cfg, chatID, threadID, args)
        case "agents":
                s.cmdAgents(cfg, chatID, threadID)
        case "approve":
                s.cmdApprove(cfg, chatID, threadID, args)
        case "connect":
                s.cmdConnect(cfg, plugin, chatID, threadID, args)
        case "connect_topic", "connecttopic":
                s.cmdConnectTopic(cfg, plugin, chatID, threadID, args)
        case "watch":
                s.cmdWatch(cfg, plugin, chatID, threadID, args)
        case "unwatch":
                s.cmdUnwatch(cfg, chatID, threadID, args)
        case "acp":
                s.cmdACP(cfg, plugin, chatID, threadID, args)
        case "routines":
                s.cmdRoutines(cfg, plugin, chatID, threadID, args)
        default:
                s.sendMsg(cfg.BotToken, chatID, threadID, "Unknown command\\. Send /help for a list\\.", nil)
        }
}

// ─── Individual command handlers ──────────────────────────────────────────────

func (s *TelegramService) cmdHelp(cfg *TelegramConfig, chatID int64, threadID int) {
        help := `*NanoClip Bot — Commands*

/status — active agents and recent runs
/issues \[project\] — open issues
/agents — list agents with status
/approve \<id\> — approve a pending request
/connect \<companyId\> — link this chat to a company
/connect\_topic \<companyId\> \<projectId\> — link this topic to a project
/watch \<event\> — subscribe this chat to an event type
/unwatch \<event\> — unsubscribe from an event type
/acp \<spawn|status|cancel\> \[agentName\] — manage agent sessions
/routines — list and run routines
/help — this message

Event types for /watch: issue\.created issue\.done approval\.created agent\.error`
        s.sendMsg(cfg.BotToken, chatID, threadID, help, nil)
}

func (s *TelegramService) cmdStatus(cfg *TelegramConfig, plugin *models.Plugin, chatID int64, threadID int) {
        var agents []models.Agent
        s.db.Find(&agents)
        var openIssues int64
        s.db.Model(&models.Issue{}).Where("status NOT IN ?", []string{"done", "cancelled"}).Count(&openIssues)
        var runs []models.HeartbeatRun
        s.db.Order("created_at desc").Limit(3).Find(&runs)

        active := 0
        for _, a := range agents {
                if a.Status == "active" {
                        active++
                }
        }
        lines := []string{
                "*NanoClip Status*",
                fmt.Sprintf("Agents: %d total, %d active", len(agents), active),
                fmt.Sprintf("Open issues: %d", openIssues),
        }
        if len(runs) > 0 {
                lines = append(lines, "\n*Recent runs:*")
                for _, r := range runs {
                        icon := "✓"
                        if r.Status == "failed" {
                                icon = "✗"
                        }
                        // Look up agent name
                        var agent models.Agent
                        name := r.AgentID
                        if err := s.db.Select("name").First(&agent, "id = ?", r.AgentID).Error; err == nil {
                                name = agent.Name
                        }
                        lines = append(lines, fmt.Sprintf("%s %s — %s", icon, escMD(name), escMD(r.Status)))
                }
        }
        s.sendMsg(cfg.BotToken, chatID, threadID, strings.Join(lines, "\n"), nil)
}

func (s *TelegramService) cmdIssues(cfg *TelegramConfig, chatID int64, threadID int, args string) {
        query := s.db.Model(&models.Issue{}).Where("status NOT IN ?", []string{"done", "cancelled"})
        if args != "" {
                query = query.Where("project_id IN (SELECT id FROM projects WHERE name ILIKE ? OR identifier ILIKE ?)", "%"+args+"%", "%"+args+"%")
        }
        var issues []models.Issue
        query.Order("created_at desc").Limit(10).Find(&issues)
        if len(issues) == 0 {
                s.sendMsg(cfg.BotToken, chatID, threadID, "No open issues\\.", nil)
                return
        }
        lines := []string{"*Open Issues:*"}
        for _, iss := range issues {
                id := ""
                if iss.Identifier != nil {
                        id = *iss.Identifier
                }
                priority := ""
                switch iss.Priority {
                case "urgent":
                        priority = " 🔴"
                case "high":
                        priority = " 🟠"
                case "medium":
                        priority = " 🟡"
                }
                lines = append(lines, fmt.Sprintf("• `%s` %s%s", escMD(id), escMD(iss.Title), priority))
        }
        if cfg.PaperclipPublicURL != "" {
                lines = append(lines, "\n[Open NanoClip]("+cfg.PaperclipPublicURL+")")
        }
        s.sendMsg(cfg.BotToken, chatID, threadID, strings.Join(lines, "\n"), nil)
}

func (s *TelegramService) cmdAgents(cfg *TelegramConfig, chatID int64, threadID int) {
        var agents []models.Agent
        s.db.Order("name").Find(&agents)
        if len(agents) == 0 {
                s.sendMsg(cfg.BotToken, chatID, threadID, "No agents configured\\.", nil)
                return
        }
        lines := []string{"*Agents:*"}
        for _, a := range agents {
                icon := "⚪"
                switch a.Status {
                case "active":
                        icon = "🟢"
                case "error":
                        icon = "🔴"
                case "idle":
                        icon = "🟡"
                }
                lines = append(lines, fmt.Sprintf("%s *%s* — %s", icon, escMD(a.Name), escMD(a.Status)))
        }
        s.sendMsg(cfg.BotToken, chatID, threadID, strings.Join(lines, "\n"), nil)
}

func (s *TelegramService) cmdApprove(cfg *TelegramConfig, chatID int64, threadID int, args string) {
        if args == "" {
                s.sendMsg(cfg.BotToken, chatID, threadID, "Usage: /approve \\<approval\\-id\\>", nil)
                return
        }
        var approval models.Approval
        if err := s.db.First(&approval, "id = ?", args).Error; err != nil {
                s.sendMsg(cfg.BotToken, chatID, threadID, "Approval not found: `"+escMD(args)+"`", nil)
                return
        }
        now := time.Now()
        s.db.Model(&approval).Updates(map[string]interface{}{
                "status": "approved", "resolved_at": now, "resolution": "approved via Telegram /approve",
        })
        s.sendMsg(cfg.BotToken, chatID, threadID, "✅ Approval `"+escMD(args)+"` approved\\.", nil)
}

func (s *TelegramService) cmdConnect(cfg *TelegramConfig, plugin *models.Plugin, chatID int64, threadID int, args string) {
        if args == "" {
                s.sendMsg(cfg.BotToken, chatID, threadID, "Usage: /connect \\<companyId\\>", nil)
                return
        }
        var company models.Company
        if err := s.db.Where("id = ? OR name ILIKE ?", args, "%"+args+"%").First(&company).Error; err != nil {
                s.sendMsg(cfg.BotToken, chatID, threadID, "Company not found: `"+escMD(args)+"`", nil)
                return
        }
        key := fmt.Sprintf("%d", chatID)
        s.connMu.Lock()
        s.connections[key] = chatConn{CompanyID: company.ID}
        s.connMu.Unlock()
        s.sendMsg(cfg.BotToken, chatID, threadID,
                fmt.Sprintf("✅ Chat connected to company *%s*\\.\nNotifications for this company will now appear here\\.", escMD(company.Name)), nil)
        s.logPlugin(plugin, "info", fmt.Sprintf("chat %d connected to company %s", chatID, company.ID))
}

func (s *TelegramService) cmdConnectTopic(cfg *TelegramConfig, plugin *models.Plugin, chatID int64, threadID int, args string) {
        parts := strings.Fields(args)
        if len(parts) < 2 {
                s.sendMsg(cfg.BotToken, chatID, threadID, "Usage: /connect\\_topic \\<companyId\\> \\<projectId\\>", nil)
                return
        }
        companyArg, projectArg := parts[0], parts[1]
        var company models.Company
        if err := s.db.Where("id = ? OR name ILIKE ?", companyArg, "%"+companyArg+"%").First(&company).Error; err != nil {
                s.sendMsg(cfg.BotToken, chatID, threadID, "Company not found: `"+escMD(companyArg)+"`", nil)
                return
        }
        var project models.Project
        if err := s.db.Where("company_id = ? AND (id = ? OR name ILIKE ? OR identifier ILIKE ?)",
                company.ID, projectArg, "%"+projectArg+"%", "%"+projectArg+"%").First(&project).Error; err != nil {
                s.sendMsg(cfg.BotToken, chatID, threadID, "Project not found: `"+escMD(projectArg)+"`", nil)
                return
        }
        key := fmt.Sprintf("%d:%d", chatID, threadID)
        s.connMu.Lock()
        s.connections[key] = chatConn{CompanyID: company.ID, ProjectID: project.ID}
        s.connMu.Unlock()
        s.sendMsg(cfg.BotToken, chatID, threadID,
                fmt.Sprintf("✅ Topic connected to *%s* / *%s*\\.\nIssue notifications for this project will appear in this topic\\.",
                        escMD(company.Name), escMD(project.Name)), nil)
        s.logPlugin(plugin, "info", fmt.Sprintf("topic %d:%d connected to project %s", chatID, threadID, project.ID))
}

func (s *TelegramService) cmdWatch(cfg *TelegramConfig, plugin *models.Plugin, chatID int64, threadID int, args string) {
        eventType := strings.TrimSpace(args)
        validEvents := map[string]bool{
                "issue.created": true, "issue.done": true, "issue.updated": true,
                "approval.created": true, "agent.error": true, "run.failed": true,
        }
        if !validEvents[eventType] {
                s.sendMsg(cfg.BotToken, chatID, threadID,
                        "Unknown event type\\. Valid: `issue\\.created` `issue\\.done` `approval\\.created` `agent\\.error` `run\\.failed`", nil)
                return
        }
        key := fmt.Sprintf("%d:%d", chatID, threadID)
        s.watchMu.Lock()
        if s.watches[key] == nil {
                s.watches[key] = &watchEntry{EventTypes: make(map[string]bool)}
        }
        s.watches[key].EventTypes[eventType] = true
        s.watchMu.Unlock()
        s.sendMsg(cfg.BotToken, chatID, threadID,
                "👁 Now watching `"+escMD(eventType)+"` events in this chat\\.", nil)
        s.logPlugin(plugin, "info", fmt.Sprintf("chat %d watching %s", chatID, eventType))
}

func (s *TelegramService) cmdUnwatch(cfg *TelegramConfig, chatID int64, threadID int, args string) {
        eventType := strings.TrimSpace(args)
        key := fmt.Sprintf("%d:%d", chatID, threadID)
        s.watchMu.Lock()
        if s.watches[key] != nil {
                delete(s.watches[key].EventTypes, eventType)
        }
        s.watchMu.Unlock()
        s.sendMsg(cfg.BotToken, chatID, threadID, "✓ Unwatched `"+escMD(eventType)+"`\\.", nil)
}

func (s *TelegramService) cmdACP(cfg *TelegramConfig, plugin *models.Plugin, chatID int64, threadID int, args string) {
        parts := strings.Fields(args)
        if len(parts) == 0 {
                s.sendMsg(cfg.BotToken, chatID, threadID,
                        "*ACP \\- Agent Control Panel*\n\n/acp status — running agents\n/acp spawn \\<agentName\\> — wake an agent\n/acp cancel \\<agentName\\> — request agent stop", nil)
                return
        }
        sub := strings.ToLower(parts[0])
        agentArg := ""
        if len(parts) > 1 {
                agentArg = strings.Join(parts[1:], " ")
        }

        switch sub {
        case "status":
                var agents []models.Agent
                s.db.Where("status = ?", "active").Find(&agents)
                if len(agents) == 0 {
                        s.sendMsg(cfg.BotToken, chatID, threadID, "No agents currently active\\.", nil)
                        return
                }
                lines := []string{"*Active Agents:*"}
                for _, a := range agents {
                        lines = append(lines, fmt.Sprintf("🟢 *%s* \\(`%s`\\)", escMD(a.Name), escMD(a.ID)))
                }
                s.sendMsg(cfg.BotToken, chatID, threadID, strings.Join(lines, "\n"), nil)

        case "spawn":
                if agentArg == "" {
                        s.sendMsg(cfg.BotToken, chatID, threadID, "Usage: /acp spawn \\<agentName\\>", nil)
                        return
                }
                var agent models.Agent
                if err := s.db.Where("name ILIKE ?", "%"+agentArg+"%").First(&agent).Error; err != nil {
                        s.sendMsg(cfg.BotToken, chatID, threadID, "Agent not found: `"+escMD(agentArg)+"`", nil)
                        return
                }
                s.hub.Publish(ws.LiveEvent{
                        Type:      "agent.wakeup",
                        CompanyID: agent.CompanyID,
                        Payload:   map[string]interface{}{"agentId": agent.ID, "source": "telegram"},
                })
                s.sendMsg(cfg.BotToken, chatID, threadID,
                        "🚀 Wakeup signal sent to *"+escMD(agent.Name)+"*\\.", nil)
                s.logPlugin(plugin, "info", fmt.Sprintf("agent %s wakeup triggered via Telegram ACP", agent.ID))

        case "cancel":
                if agentArg == "" {
                        s.sendMsg(cfg.BotToken, chatID, threadID, "Usage: /acp cancel \\<agentName\\>", nil)
                        return
                }
                var agent models.Agent
                if err := s.db.Where("name ILIKE ?", "%"+agentArg+"%").First(&agent).Error; err != nil {
                        s.sendMsg(cfg.BotToken, chatID, threadID, "Agent not found: `"+escMD(agentArg)+"`", nil)
                        return
                }
                s.hub.Publish(ws.LiveEvent{
                        Type:      "agent.cancel",
                        CompanyID: agent.CompanyID,
                        Payload:   map[string]interface{}{"agentId": agent.ID, "source": "telegram"},
                })
                s.sendMsg(cfg.BotToken, chatID, threadID,
                        "⏹ Cancellation requested for *"+escMD(agent.Name)+"*\\.", nil)

        default:
                s.sendMsg(cfg.BotToken, chatID, threadID, "Unknown ACP subcommand\\. Try /acp status, spawn, or cancel\\.", nil)
        }
}

func (s *TelegramService) cmdRoutines(cfg *TelegramConfig, plugin *models.Plugin, chatID int64, threadID int, args string) {
        parts := strings.Fields(args)
        if len(parts) == 0 {
                var routines []models.Routine
                s.db.Where("status = ?", "active").Order("title").Limit(15).Find(&routines)
                if len(routines) == 0 {
                        s.sendMsg(cfg.BotToken, chatID, threadID, "No routines configured\\.", nil)
                        return
                }
                lines := []string{"*Routines* \\(use /routines run \\<name\\> to trigger\\):"}
                for _, r := range routines {
                        desc := ""
                        if r.Description != nil {
                                desc = " — " + escMD(*r.Description)
                        }
                        lines = append(lines, fmt.Sprintf("• `%s`%s", escMD(r.Title), desc))
                }
                s.sendMsg(cfg.BotToken, chatID, threadID, strings.Join(lines, "\n"), nil)
                return
        }
        if parts[0] == "run" && len(parts) > 1 {
                routineName := strings.Join(parts[1:], " ")
                var routine models.Routine
                if err := s.db.Where("title ILIKE ? AND status = ?", "%"+routineName+"%", "active").First(&routine).Error; err != nil {
                        s.sendMsg(cfg.BotToken, chatID, threadID, "Routine not found: `"+escMD(routineName)+"`", nil)
                        return
                }
                s.hub.Publish(ws.LiveEvent{
                        Type:    "routine.trigger",
                        Payload: map[string]interface{}{"routineId": routine.ID, "source": "telegram"},
                })
                s.sendMsg(cfg.BotToken, chatID, threadID,
                        "▶ Triggered routine *"+escMD(routine.Title)+"*\\.", nil)
                s.logPlugin(plugin, "info", "routine "+routine.ID+" triggered via Telegram")
                return
        }
        s.sendMsg(cfg.BotToken, chatID, threadID, "Usage: /routines or /routines run \\<name\\>", nil)
}

// ─── Event notifications ───────────────────────────────────────────────────────

func (s *TelegramService) handleEvent(cfg *TelegramConfig, plugin *models.Plugin, event ws.LiveEvent) {
        payload := eventPayloadMap(event.Payload)

        switch event.Type {
        case "issue.created":
                title, _ := payload["title"].(string)
                identifier, _ := payload["identifier"].(string)
                if title == "" {
                        return
                }
                msg := fmt.Sprintf("📋 *New Issue:* `%s` — %s", escMD(identifier), escMD(title))
                if cfg.PaperclipPublicURL != "" {
                        msg += fmt.Sprintf("\n[View in NanoClip](%s)", cfg.PaperclipPublicURL)
                }
                s.broadcastEvent(cfg, event.CompanyID, event.Type, msg)

        case "issue.updated":
                status, _ := payload["status"].(string)
                if status != "done" {
                        return
                }
                title, _ := payload["title"].(string)
                identifier, _ := payload["identifier"].(string)
                msg := fmt.Sprintf("✅ *Issue Done:* `%s` — %s", escMD(identifier), escMD(title))
                s.broadcastEvent(cfg, event.CompanyID, event.Type, msg)

        case "approval.created":
                title, _ := payload["title"].(string)
                approvalID, _ := payload["id"].(string)
                msg := fmt.Sprintf("🔔 *Approval Requested:* %s", escMD(title))
                keyboard := inlineKeyboard([][]inlineButton{{{Text: "✅ Approve", Data: "approve:" + approvalID}, {Text: "❌ Reject", Data: "reject:" + approvalID}}})
                chatID := cfg.ApprovalsChatID
                if chatID == "" {
                        chatID = cfg.DefaultChatID
                }
                if chatID == "" {
                        return
                }
                s.sendMsg(cfg.BotToken, chatIDFromStr(chatID), 0, msg, keyboard)

        case "agent.error", "run.failed":
                agentName, _ := payload["agentName"].(string)
                errMsg, _ := payload["error"].(string)
                if agentName == "" {
                        agentName = "Agent"
                }
                msg := fmt.Sprintf("⚠️ *Agent Error* — %s\n%s", escMD(agentName), escMD(errMsg))
                s.broadcastEvent(cfg, event.CompanyID, "agent.error", msg)

        case "escalation.created":
                if !cfg.EnableEscalation {
                        return
                }
                s.handleEscalationEvent(cfg, plugin, payload)
        }
}

// broadcastEvent sends to the default chat AND any watch-registered chats for this event type.
func (s *TelegramService) broadcastEvent(cfg *TelegramConfig, companyID, eventType, msg string) {
        sent := map[string]bool{}

        // Default chat
        if cfg.DefaultChatID != "" {
                key := cfg.DefaultChatID + ":0"
                if !sent[key] {
                        s.sendMsg(cfg.BotToken, chatIDFromStr(cfg.DefaultChatID), 0, msg, nil)
                        sent[key] = true
                }
        }

        // Watch registry
        s.watchMu.RLock()
        for _, entry := range s.watches {
                if entry.EventTypes[eventType] || entry.EventTypes["issue.done"] && eventType == "issue.updated" {
                        key := fmt.Sprintf("%v:%d", entry, entry.ThreadID)
                        if !sent[key] {
                                // We need chatID — store it in watchEntry; for now broadcast to default
                                // (full per-chat routing requires storing chatID in watchEntry)
                        }
                }
        }
        s.watchMu.RUnlock()
}

// ─── Escalation event handler ─────────────────────────────────────────────────

func (s *TelegramService) handleEscalationEvent(cfg *TelegramConfig, plugin *models.Plugin, payload map[string]interface{}) {
        escalID, _ := payload["escalationId"].(string)
        agentName, _ := payload["agentName"].(string)
        reasoning, _ := payload["reasoning"].(string)
        suggestedReply, _ := payload["suggestedReply"].(string)
        timeoutMs, _ := payload["timeoutMs"].(float64)
        defaultAction, _ := payload["defaultAction"].(string)
        if escalID == "" || agentName == "" {
                return
        }
        if timeoutMs == 0 {
                timeoutMs = 120000
        }
        if defaultAction == "" {
                defaultAction = "defer"
        }

        chatID := cfg.EscalationChatID
        if chatID == "" {
                chatID = cfg.ApprovalsChatID
        }
        if chatID == "" {
                chatID = cfg.DefaultChatID
        }
        if chatID == "" {
                return
        }

        lines := []string{
                fmt.Sprintf("🆘 *Escalation from %s*", escMD(agentName)),
                "",
                "*Reasoning:* " + escMD(reasoning),
        }
        if suggestedReply != "" {
                lines = append(lines, "*Suggested reply:* _"+escMD(suggestedReply)+"_")
        }
        timeoutSec := int(timeoutMs / 1000)
        lines = append(lines, fmt.Sprintf("\n⏱ Auto\\-%s in %ds", escMD(defaultAction), timeoutSec))
        msg := strings.Join(lines, "\n")

        keyboard := inlineKeyboard([][]inlineButton{{
                {Text: "💬 Use suggested reply", Data: "escal_reply:" + escalID},
                {Text: "✏️ Override", Data: "escal_override:" + escalID},
                {Text: "🚫 Dismiss", Data: "escal_dismiss:" + escalID},
        }})

        // We'll capture the message ID from the send response for later editing
        cid := chatIDFromStr(chatID)
        messageID := s.sendMsgGetID(cfg.BotToken, cid, 0, msg, keyboard)

        record := &escalationRecord{
                ID:             escalID,
                AgentName:      agentName,
                Reasoning:      reasoning,
                SuggestedReply: suggestedReply,
                ChatID:         cid,
                MessageID:      messageID,
                TimeoutAt:      time.Now().Add(time.Duration(timeoutMs) * time.Millisecond),
                DefaultAction:  defaultAction,
                Status:         "pending",
        }
        s.escalMu.Lock()
        s.escalations[escalID] = record
        s.escalMu.Unlock()
        s.logPlugin(plugin, "info", fmt.Sprintf("escalation %s from agent %s — timeout %ds", escalID, agentName, timeoutSec))
}

// PublishEscalation is called by other services to trigger an escalation notification.
func (s *TelegramService) PublishEscalation(agentName, reasoning, suggestedReply, defaultAction string, timeoutMs int) {
        s.hub.Publish(ws.LiveEvent{
                Type: "escalation.created",
                Payload: map[string]interface{}{
                        "escalationId":   uuid.NewString(),
                        "agentName":      agentName,
                        "reasoning":      reasoning,
                        "suggestedReply": suggestedReply,
                        "defaultAction":  defaultAction,
                        "timeoutMs":      float64(timeoutMs),
                },
        })
}

// ─── Telegram API helpers ─────────────────────────────────────────────────────

func (s *TelegramService) getUpdates(token string, offset int) ([]tgUpdate, int, error) {
        url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?timeout=20&offset=%d&allowed_updates=%s",
                token, offset, `["message","callback_query"]`)
        resp, err := http.Get(url)
        if err != nil {
                return nil, offset, err
        }
        defer resp.Body.Close()
        body, _ := io.ReadAll(resp.Body)
        var result struct {
                OK     bool       `json:"ok"`
                Result []tgUpdate `json:"result"`
        }
        if err := json.Unmarshal(body, &result); err != nil {
                return nil, offset, err
        }
        newOffset := offset
        for _, u := range result.Result {
                if u.UpdateID >= newOffset {
                        newOffset = u.UpdateID + 1
                }
        }
        return result.Result, newOffset, nil
}

// sendMsg sends a MarkdownV2 message with optional inline keyboard.
func (s *TelegramService) sendMsg(token string, chatID int64, threadID int, text string, keyboard interface{}) {
        s.sendMsgGetID(token, chatID, threadID, text, keyboard)
}

func (s *TelegramService) sendMsgGetID(token string, chatID int64, threadID int, text string, keyboard interface{}) int {
        if chatID == 0 || token == "" {
                return 0
        }
        payload := map[string]interface{}{
                "chat_id":    chatID,
                "text":       text,
                "parse_mode": "MarkdownV2",
        }
        if threadID > 0 {
                payload["message_thread_id"] = threadID
        }
        if keyboard != nil {
                payload["reply_markup"] = keyboard
        }
        b, _ := json.Marshal(payload)
        url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
        resp, err := http.Post(url, "application/json", bytes.NewReader(b))
        if err != nil {
                log.Printf("[telegram] sendMessage error: %v", err)
                return 0
        }
        defer resp.Body.Close()
        var result struct {
                OK     bool `json:"ok"`
                Result struct {
                        MessageID int `json:"message_id"`
                } `json:"result"`
        }
        body, _ := io.ReadAll(resp.Body)
        json.Unmarshal(body, &result)
        return result.Result.MessageID
}

func (s *TelegramService) editMessage(token string, chatID int64, messageID int, text string) {
        payload := map[string]interface{}{
                "chat_id":    chatID,
                "message_id": messageID,
                "text":       text,
                "parse_mode": "MarkdownV2",
        }
        b, _ := json.Marshal(payload)
        url := fmt.Sprintf("https://api.telegram.org/bot%s/editMessageText", token)
        resp, err := http.Post(url, "application/json", bytes.NewReader(b))
        if err != nil {
                log.Printf("[telegram] editMessage error: %v", err)
                return
        }
        resp.Body.Close()
}

func (s *TelegramService) answerCallback(token, callbackID, text string) {
        payload := map[string]interface{}{"callback_query_id": callbackID, "text": text}
        b, _ := json.Marshal(payload)
        url := fmt.Sprintf("https://api.telegram.org/bot%s/answerCallbackQuery", token)
        resp, err := http.Post(url, "application/json", bytes.NewReader(b))
        if err != nil {
                log.Printf("[telegram] answerCallback error: %v", err)
                return
        }
        resp.Body.Close()
}

func (s *TelegramService) sendTyping(token string, chatID int64, threadID int) {
        payload := map[string]interface{}{"chat_id": chatID, "action": "typing"}
        if threadID > 0 {
                payload["message_thread_id"] = threadID
        }
        b, _ := json.Marshal(payload)
        url := fmt.Sprintf("https://api.telegram.org/bot%s/sendChatAction", token)
        resp, err := http.Post(url, "application/json", bytes.NewReader(b))
        if err != nil {
                return
        }
        resp.Body.Close()
}

// ─── Logging ──────────────────────────────────────────────────────────────────

func (s *TelegramService) logPlugin(plugin *models.Plugin, level, message string) {
        if plugin == nil {
                log.Printf("[telegram] [%s] %s", level, message)
                return
        }
        entry := models.PluginLog{
                ID:        uuid.NewString(),
                PluginID:  plugin.ID,
                Level:     level,
                Message:   message,
                CreatedAt: time.Now(),
        }
        s.db.Create(&entry)
        log.Printf("[telegram] [%s] %s", level, message)
}

// ─── Inline keyboard builder ──────────────────────────────────────────────────

type inlineButton struct {
        Text string
        Data string
        URL  string
}

func inlineKeyboard(rows [][]inlineButton) map[string]interface{} {
        keyboard := make([][][]map[string]interface{}, 0, len(rows))
        for _, row := range rows {
                r := make([]map[string]interface{}, 0, len(row))
                for _, btn := range row {
                        b := map[string]interface{}{"text": btn.Text}
                        if btn.URL != "" {
                                b["url"] = btn.URL
                        } else {
                                b["callback_data"] = btn.Data
                        }
                        r = append(r, b)
                }
                keyboard = append(keyboard, [][]map[string]interface{}{r})
        }
        // Flatten: inline_keyboard is [][]button, not [][][]button
        flat := make([][]map[string]interface{}, 0, len(rows))
        for _, row := range rows {
                r := make([]map[string]interface{}, 0, len(row))
                for _, btn := range row {
                        b := map[string]interface{}{"text": btn.Text}
                        if btn.URL != "" {
                                b["url"] = btn.URL
                        } else {
                                b["callback_data"] = btn.Data
                        }
                        r = append(r, b)
                }
                flat = append(flat, r)
        }
        _ = keyboard
        return map[string]interface{}{"inline_keyboard": flat}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func eventPayloadMap(payload interface{}) map[string]interface{} {
        if payload == nil {
                return map[string]interface{}{}
        }
        b, err := json.Marshal(payload)
        if err != nil {
                return map[string]interface{}{}
        }
        var m map[string]interface{}
        json.Unmarshal(b, &m)
        if m == nil {
                return map[string]interface{}{}
        }
        return m
}

func chatIDFromStr(s string) int64 {
        if s == "" {
                return 0
        }
        var id int64
        fmt.Sscanf(s, "%d", &id)
        return id
}

// escMD escapes special MarkdownV2 characters.
func escMD(s string) string {
        special := `\_*[]()~` + "`" + `>#+-=|{}.!`
        var b strings.Builder
        for _, c := range s {
                if strings.ContainsRune(special, c) {
                        b.WriteRune('\\')
                }
                b.WriteRune(c)
        }
        return b.String()
}

func extractIssueIDFromText(text string) string {
        words := strings.Fields(text)
        for _, w := range words {
                w = strings.Trim(w, ".,!?;:")
                if len(w) > 3 && strings.Contains(w, "-") {
                        parts := strings.SplitN(w, "-", 2)
                        if len(parts) == 2 && len(parts[0]) >= 2 && len(parts[0]) <= 8 {
                                allUpper := true
                                for _, c := range parts[0] {
                                        if c < 'A' || c > 'Z' {
                                                allUpper = false
                                                break
                                        }
                                }
                                allDigit := true
                                for _, c := range parts[1] {
                                        if c < '0' || c > '9' {
                                                allDigit = false
                                                break
                                        }
                                }
                                if allUpper && allDigit {
                                        return w
                                }
                        }
                }
        }
        return ""
}
