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

// TelegramConfig holds config values stored in the Plugin's Config JSON.
type TelegramConfig struct {
        BotToken       string `json:"botToken"`
        DefaultChatID  string `json:"defaultChatId"`
        ApprovalsChatID string `json:"approvalsChatId"`
        ErrorsChatID   string `json:"errorsChatId"`
        PaperclipPublicURL string `json:"paperclipPublicUrl"`
        EnableCommands bool   `json:"enableCommands"`
        EnableInbound  bool   `json:"enableInbound"`
}

// TelegramService runs the bidirectional Telegram bot integration.
type TelegramService struct {
        db     *gorm.DB
        hub    *ws.Hub
        mu     sync.RWMutex
        cfg    *TelegramConfig
        plugin *models.Plugin
        stop   chan struct{}
        wg     sync.WaitGroup
}

var GlobalTelegramService *TelegramService

func NewTelegramService(db *gorm.DB, hub *ws.Hub) *TelegramService {
        s := &TelegramService{
                db:   db,
                hub:  hub,
                stop: make(chan struct{}),
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

        s.wg.Add(2)
        go s.pollLoop()
        go s.eventLoop(eventCh)
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

// ─── Telegram polling ─────────────────────────────────────────────────────────

type tgUpdate struct {
        UpdateID int `json:"update_id"`
        Message  *struct {
                MessageID int `json:"message_id"`
                From      *struct {
                        ID       int    `json:"id"`
                        Username string `json:"username"`
                } `json:"from"`
                Chat struct {
                        ID int64 `json:"id"`
                } `json:"chat"`
                Text          string `json:"text"`
                ReplyToMessage *struct {
                        MessageID int    `json:"message_id"`
                        Text      string `json:"text"`
                } `json:"reply_to_message"`
        } `json:"message"`
        CallbackQuery *struct {
                ID   string `json:"id"`
                From struct {
                        ID       int    `json:"id"`
                        Username string `json:"username"`
                } `json:"from"`
                Message *struct {
                        MessageID int `json:"message_id"`
                        Chat      struct {
                                ID int64 `json:"id"`
                        } `json:"chat"`
                } `json:"message"`
                Data string `json:"data"`
        } `json:"callback_query"`
}

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
                        // Plugin not configured — re-check every 10s
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

// ─── Update handling ──────────────────────────────────────────────────────────

func (s *TelegramService) handleUpdate(cfg *TelegramConfig, plugin *models.Plugin, upd tgUpdate) {
        if upd.CallbackQuery != nil {
                s.handleCallbackQuery(cfg, plugin, upd.CallbackQuery)
                return
        }
        if upd.Message == nil {
                return
        }
        msg := upd.Message
        chatID := msg.Chat.ID
        text := strings.TrimSpace(msg.Text)

        // Reply routing: replies to bot messages create issue comments
        if cfg.EnableInbound && msg.ReplyToMessage != nil {
                s.routeReply(cfg, plugin, chatID, msg.ReplyToMessage.Text, text)
                return
        }

        // Bot commands
        if cfg.EnableCommands && strings.HasPrefix(text, "/") {
                s.handleCommand(cfg, plugin, chatID, text)
        }
}

func (s *TelegramService) handleCallbackQuery(cfg *TelegramConfig, plugin *models.Plugin, cq *struct {
        ID   string `json:"id"`
        From struct {
                ID       int    `json:"id"`
                Username string `json:"username"`
        } `json:"from"`
        Message *struct {
                MessageID int `json:"message_id"`
                Chat      struct {
                        ID int64 `json:"id"`
                } `json:"chat"`
        } `json:"message"`
        Data string `json:"data"`
}) {
        // Data format: "approve:<approvalId>" or "reject:<approvalId>"
        parts := strings.SplitN(cq.Data, ":", 2)
        if len(parts) != 2 {
                s.answerCallback(cfg.BotToken, cq.ID, "Unknown action")
                return
        }
        action, approvalID := parts[0], parts[1]

        var approval models.Approval
        if err := s.db.First(&approval, "id = ?", approvalID).Error; err != nil {
                s.answerCallback(cfg.BotToken, cq.ID, "Approval not found")
                return
        }

        now := time.Now()
        actor := fmt.Sprintf("@%s (Telegram)", cq.From.Username)
        if action == "approve" {
                s.db.Model(&approval).Updates(map[string]interface{}{
                        "status":            "approved",
                        "decided_at":        now,
                        "decision_note":     "approved by " + actor,
                        "updated_at":        now,
                })
                s.answerCallback(cfg.BotToken, cq.ID, "Approved!")
                if cq.Message != nil {
                        s.editMessage(cfg.BotToken, cq.Message.Chat.ID, cq.Message.MessageID,
                                "Approval *APPROVED* by "+escapeMarkdown(actor))
                }
                s.logPlugin(plugin, "info", fmt.Sprintf("approval %s approved by %s via Telegram", approvalID, actor))
        } else if action == "reject" {
                s.db.Model(&approval).Updates(map[string]interface{}{
                        "status":        "rejected",
                        "decided_at":    now,
                        "decision_note": "rejected by " + actor,
                        "updated_at":    now,
                })
                s.answerCallback(cfg.BotToken, cq.ID, "Rejected.")
                if cq.Message != nil {
                        s.editMessage(cfg.BotToken, cq.Message.Chat.ID, cq.Message.MessageID,
                                "Approval *REJECTED* by "+escapeMarkdown(actor))
                }
                s.logPlugin(plugin, "info", fmt.Sprintf("approval %s rejected by %s via Telegram", approvalID, actor))
        } else {
                s.answerCallback(cfg.BotToken, cq.ID, "Unknown action")
        }
}

func (s *TelegramService) routeReply(cfg *TelegramConfig, plugin *models.Plugin, chatID int64, quotedText, replyText string) {
        // Look for an issue identifier in the quoted message (e.g. "ACME-42")
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
        s.sendMessage(cfg.BotToken, chatID, "Comment added to "+identifier, nil)
        s.logPlugin(plugin, "info", fmt.Sprintf("reply routed to issue %s as comment", identifier))
}

func (s *TelegramService) handleCommand(cfg *TelegramConfig, plugin *models.Plugin, chatID int64, text string) {
        parts := strings.Fields(text)
        cmd := strings.ToLower(parts[0])

        switch {
        case cmd == "/help" || cmd == "/start":
                help := `*NanoClip Bot Commands*

/status — active agents and recent runs
/issues — open issues
/agents — list agents
/approve <id> — approve a pending request
/help — this message`
                s.sendMessage(cfg.BotToken, chatID, help, nil)

        case cmd == "/status":
                var agents []models.Agent
                s.db.Limit(10).Find(&agents)
                var runs []models.HeartbeatRun
                s.db.Order("created_at desc").Limit(5).Find(&runs)
                msg := fmt.Sprintf("*Agents*: %d total\n*Recent runs*: %d", len(agents), len(runs))
                s.sendMessage(cfg.BotToken, chatID, msg, nil)

        case cmd == "/issues":
                var issues []models.Issue
                s.db.Where("status NOT IN (?)", []string{"done", "cancelled"}).Limit(10).Order("created_at desc").Find(&issues)
                if len(issues) == 0 {
                        s.sendMessage(cfg.BotToken, chatID, "No open issues.", nil)
                        return
                }
                lines := []string{"*Open Issues:*"}
                for _, iss := range issues {
                        lines = append(lines, fmt.Sprintf("• [%s] %s", iss.Identifier, escapeMarkdown(iss.Title)))
                }
                s.sendMessage(cfg.BotToken, chatID, strings.Join(lines, "\n"), nil)

        case cmd == "/agents":
                var agents []models.Agent
                s.db.Limit(20).Find(&agents)
                if len(agents) == 0 {
                        s.sendMessage(cfg.BotToken, chatID, "No agents configured.", nil)
                        return
                }
                lines := []string{"*Agents:*"}
                for _, a := range agents {
                        status := "●"
                        if a.Status == "active" {
                                status = "🟢"
                        }
                        lines = append(lines, fmt.Sprintf("%s %s", status, escapeMarkdown(a.Name)))
                }
                s.sendMessage(cfg.BotToken, chatID, strings.Join(lines, "\n"), nil)

        case cmd == "/approve" && len(parts) >= 2:
                approvalID := parts[1]
                var approval models.Approval
                if err := s.db.First(&approval, "id = ?", approvalID).Error; err != nil {
                        s.sendMessage(cfg.BotToken, chatID, "Approval not found: "+approvalID, nil)
                        return
                }
                now := time.Now()
                s.db.Model(&approval).Updates(map[string]interface{}{
                        "status":      "approved",
                        "resolved_at": now,
                        "resolution":  "approved via Telegram /approve command",
                })
                s.sendMessage(cfg.BotToken, chatID, "✓ Approval "+approvalID+" approved.", nil)

        default:
                s.sendMessage(cfg.BotToken, chatID, "Unknown command. Send /help for a list of commands.", nil)
        }
}

// ─── Event notifications ──────────────────────────────────────────────────────

func (s *TelegramService) handleEvent(cfg *TelegramConfig, plugin *models.Plugin, event ws.LiveEvent) {
        chatID := cfg.DefaultChatID
        if chatID == "" {
                return
        }

        switch event.Type {
        case "issue.created":
                payload := eventPayloadMap(event.Payload)
                title, _ := payload["title"].(string)
                identifier, _ := payload["identifier"].(string)
                if title == "" {
                        return
                }
                msg := fmt.Sprintf("📋 *New Issue*: %s — %s", identifier, escapeMarkdown(title))
                if cfg.PaperclipPublicURL != "" && identifier != "" {
                        msg += fmt.Sprintf("\n[View issue](%s)", cfg.PaperclipPublicURL)
                }
                s.sendMessage(cfg.BotToken, chatIDFromStr(chatID), msg, nil)

        case "issue.updated":
                payload := eventPayloadMap(event.Payload)
                status, _ := payload["status"].(string)
                if status != "done" {
                        return
                }
                title, _ := payload["title"].(string)
                identifier, _ := payload["identifier"].(string)
                msg := fmt.Sprintf("✅ *Issue Done*: %s — %s", identifier, escapeMarkdown(title))
                s.sendMessage(cfg.BotToken, chatIDFromStr(chatID), msg, nil)

        case "approval.created":
                approvalChatID := cfg.ApprovalsChatID
                if approvalChatID == "" {
                        approvalChatID = chatID
                }
                payload := eventPayloadMap(event.Payload)
                title, _ := payload["title"].(string)
                approvalID, _ := payload["id"].(string)
                msg := fmt.Sprintf("🔔 *Approval Requested*: %s", escapeMarkdown(title))
                keyboard := map[string]interface{}{
                        "inline_keyboard": [][]map[string]interface{}{
                                {
                                        {"text": "✅ Approve", "callback_data": "approve:" + approvalID},
                                        {"text": "❌ Reject", "callback_data": "reject:" + approvalID},
                                },
                        },
                }
                s.sendMessage(cfg.BotToken, chatIDFromStr(approvalChatID), msg, keyboard)

        case "agent.error", "run.failed":
                errorChatID := cfg.ErrorsChatID
                if errorChatID == "" {
                        errorChatID = chatID
                }
                payload := eventPayloadMap(event.Payload)
                agentName, _ := payload["agentName"].(string)
                errMsg, _ := payload["error"].(string)
                if agentName == "" {
                        agentName = "Agent"
                }
                msg := fmt.Sprintf("⚠️ *Agent Error* — %s\n%s", escapeMarkdown(agentName), escapeMarkdown(errMsg))
                s.sendMessage(cfg.BotToken, chatIDFromStr(errorChatID), msg, nil)
        }
}

// ─── Telegram API helpers ─────────────────────────────────────────────────────

func (s *TelegramService) getUpdates(token string, offset int) ([]tgUpdate, int, error) {
        url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?timeout=20&offset=%d", token, offset)
        resp, err := http.Get(url)
        if err != nil {
                return nil, offset, err
        }
        defer resp.Body.Close()
        body, _ := io.ReadAll(resp.Body)
        var result struct {
                OK     bool        `json:"ok"`
                Result []tgUpdate  `json:"result"`
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

func (s *TelegramService) sendMessage(token string, chatID int64, text string, replyMarkup interface{}) {
        if chatID == 0 || token == "" {
                return
        }
        payload := map[string]interface{}{
                "chat_id":    chatID,
                "text":       text,
                "parse_mode": "Markdown",
        }
        if replyMarkup != nil {
                payload["reply_markup"] = replyMarkup
        }
        b, _ := json.Marshal(payload)
        url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
        resp, err := http.Post(url, "application/json", bytes.NewReader(b))
        if err != nil {
                log.Printf("[telegram] sendMessage error: %v", err)
                return
        }
        resp.Body.Close()
}

func (s *TelegramService) editMessage(token string, chatID int64, messageID int, text string) {
        payload := map[string]interface{}{
                "chat_id":    chatID,
                "message_id": messageID,
                "text":       text,
                "parse_mode": "Markdown",
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
        payload := map[string]interface{}{
                "callback_query_id": callbackID,
                "text":              text,
        }
        b, _ := json.Marshal(payload)
        url := fmt.Sprintf("https://api.telegram.org/bot%s/answerCallbackQuery", token)
        resp, err := http.Post(url, "application/json", bytes.NewReader(b))
        if err != nil {
                log.Printf("[telegram] answerCallback error: %v", err)
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

func escapeMarkdown(s string) string {
        replacer := strings.NewReplacer(
                "_", `\_`,
                "*", `\*`,
                "[", `\[`,
                "`", "\\`",
        )
        return replacer.Replace(s)
}

func extractIssueIDFromText(text string) string {
        // Look for patterns like "ACME-42" or a UUID
        for _, word := range strings.Fields(text) {
                // Simple identifier pattern: LETTERS-DIGITS
                if len(word) > 2 && strings.Contains(word, "-") {
                        parts := strings.SplitN(word, "-", 2)
                        if len(parts[0]) >= 1 && len(parts[1]) >= 1 {
                                allAlpha := true
                                for _, c := range parts[0] {
                                        if c < 'A' || c > 'Z' {
                                                allAlpha = false
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
                                if allAlpha && allDigit {
                                        return word
                                }
                        }
                }
        }
        return ""
}

// ─── Manifest for plugin registration ────────────────────────────────────────

func TelegramManifest() map[string]interface{} {
        return map[string]interface{}{
                "id":          "paperclip-plugin-telegram",
                "apiVersion":  1,
                "version":     "0.2.4",
                "displayName": "Telegram Bot",
                "description": "Bidirectional Telegram integration: push notifications, bot commands, inline approve/reject buttons, and reply routing back to issues.",
                "author":      "mvanhorn",
                "categories":  []string{"connector", "automation"},
                "capabilities": []string{
                        "issues.read", "issues.create", "agents.read", "events.subscribe",
                        "http.outbound", "secrets.read-ref", "activity.log.write",
                },
                "instanceConfigSchema": map[string]interface{}{
                        "type": "object",
                        "properties": map[string]interface{}{
                                "botToken": map[string]interface{}{
                                        "type":        "string",
                                        "title":       "Telegram Bot Token",
                                        "description": "Get a token from @BotFather on Telegram.",
                                        "default":     "",
                                },
                                "defaultChatId": map[string]interface{}{
                                        "type":        "string",
                                        "title":       "Default Chat ID",
                                        "description": "Telegram chat ID to send notifications to. Run getUpdates and find chat.id.",
                                        "default":     "",
                                },
                                "approvalsChatId": map[string]interface{}{
                                        "type":        "string",
                                        "title":       "Approvals Chat ID",
                                        "description": "Separate chat for approval notifications. Falls back to default chat.",
                                        "default":     "",
                                },
                                "errorsChatId": map[string]interface{}{
                                        "type":        "string",
                                        "title":       "Errors Chat ID",
                                        "description": "Separate chat for agent error notifications. Falls back to default chat.",
                                        "default":     "",
                                },
                                "paperclipPublicUrl": map[string]interface{}{
                                        "type":        "string",
                                        "title":       "NanoClip Public URL",
                                        "description": "Public URL of your NanoClip instance, used for deep links in notifications.",
                                        "default":     "",
                                },
                                "enableCommands": map[string]interface{}{
                                        "type":    "boolean",
                                        "title":   "Enable bot commands",
                                        "description": "Allow /status, /issues, /agents, /approve commands.",
                                        "default": true,
                                },
                                "enableInbound": map[string]interface{}{
                                        "type":    "boolean",
                                        "title":   "Enable inbound reply routing",
                                        "description": "Route Telegram replies to issue comments.",
                                        "default": true,
                                },
                        },
                        "required": []string{"botToken"},
                },
        }
}
