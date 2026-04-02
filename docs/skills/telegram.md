# Skill: Telegram Notifications & Approvals

This skill teaches NanoClip agents how to send notifications, push approvals, and trigger escalations to a Telegram bot. All Telegram communication goes through the built-in `TelegramService` that runs alongside the NanoClip server — agents never call the Telegram API directly.

---

## How it works

The Telegram plugin runs a background service that:
1. Polls Telegram for incoming commands (users chatting with the bot)
2. Listens to internal NanoClip events (issues created, approvals needed, etc.)
3. Sends notifications and inline-button messages to configured chats

Agents trigger notifications by **publishing events to the NanoClip event hub**. The Telegram service picks them up automatically.

---

## Sending a notification when an issue is created

The `issue.created` event fires automatically when a new issue is created through the API. Agents do not need to do anything extra — the Telegram service sends the notification if `defaultChatId` is configured.

To verify it is configured, check the Plugin Manager settings:
- `botToken` — your bot token from @BotFather
- `defaultChatId` — numeric ID of the Telegram chat to receive notifications
- `enableCommands: true` — enables bot commands

**Finding your chat ID:**
1. Send any message to your bot
2. Open: `https://api.telegram.org/bot<YOUR_TOKEN>/getUpdates`
3. Look for `"chat": {"id": 123456789}` — that number is your chat ID

---

## Sending an approval request to Telegram

When an agent needs human approval before taking an action, it creates an `Approval` record. The Telegram service sends a message with Approve / Reject buttons automatically.

**What triggers the approval notification:**
The event `approval.created` is published to the hub whenever a new approval is created via the standard approvals API endpoint.

**Approval chat routing:**
- If `approvalsChatId` is set — approvals go there
- Otherwise — approvals go to `defaultChatId`

**Inline buttons sent:**
```
✅ Approve    ❌ Reject
```
The human taps a button → the approval record is updated in the database → the agent's next run sees the decision.

---

## Triggering an escalation from an agent

When an agent has low confidence or needs human guidance, it can trigger an escalation. This sends a time-limited decision request to Telegram with three choices.

**How to trigger an escalation:**

Publish an `escalation.created` event to the hub with this payload:

```json
{
  "escalationId": "<unique-id>",
  "agentName": "My Agent",
  "reasoning": "I'm not sure whether to delete the record or archive it. The user's intent is ambiguous.",
  "suggestedReply": "I'll archive the record as a safe default.",
  "defaultAction": "auto_reply",
  "timeoutMs": 120000
}
```

| Field | Required | Description |
|---|---|---|
| `escalationId` | Yes | A UUID for this escalation |
| `agentName` | Yes | Human-readable agent name |
| `reasoning` | Yes | Why the agent is escalating |
| `suggestedReply` | No | What the agent would do by default |
| `defaultAction` | No | `auto_reply`, `defer`, or `close` — fires if no human responds in time |
| `timeoutMs` | No | Milliseconds before default action fires (default: 120000) |

**What the human sees in Telegram:**
```
🆘 Escalation from My Agent

Reasoning: I'm not sure whether to delete the record...
Suggested reply: I'll archive the record as a safe default.

⏱ Auto-auto_reply in 120s

[💬 Use suggested reply]  [✏️ Override]  [🚫 Dismiss]
```

**Escalation chat routing:**
- `escalationChatId` → `approvalsChatId` → `defaultChatId` (first non-empty wins)

---

## Subscribing a chat to specific events (/watch)

Users can subscribe their chat or Telegram topic to specific event types using bot commands:

```
/watch issue.created
/watch approval.created
/watch agent.error
/unwatch issue.created
```

Supported event types:
| Event | When it fires |
|---|---|
| `issue.created` | A new issue is created |
| `issue.done` | An issue is marked done |
| `issue.updated` | Any issue update |
| `approval.created` | An agent needs approval |
| `agent.error` | An agent run fails |
| `run.failed` | A heartbeat run fails |

---

## Linking a chat to a company or project (/connect)

By default, notifications go to `defaultChatId`. Users can override this per-chat or per-Telegram-topic:

```
/connect <companyId>              — route all company events here
/connect_topic <companyId> <projectId>  — route project events to this topic
```

This is useful for Telegram Groups with Topics — each project gets its own topic thread.

---

## Bot commands available to users

| Command | Description |
|---|---|
| `/status` | Active agents, open issue count, recent runs |
| `/issues [project]` | List open issues, optionally filtered by project |
| `/agents` | List all agents with status indicators |
| `/approve <id>` | Approve a pending approval by ID |
| `/connect <companyId>` | Link this chat to a company |
| `/connect_topic <companyId> <projectId>` | Link this topic to a project |
| `/watch <event>` | Subscribe to an event type |
| `/unwatch <event>` | Unsubscribe from an event type |
| `/acp spawn <agentName>` | Send a wakeup signal to an agent |
| `/acp status` | Show currently active agents |
| `/acp cancel <agentName>` | Request an agent to stop |
| `/routines` | List active routines |
| `/routines run <name>` | Trigger a routine by name |
| `/help` | Show all commands |

---

## Sending media to an issue

Users can send photos or documents to the bot with a caption referencing an issue identifier:

```
PROJ-42 screenshot of the error
```

The media info (file_id, type, dimensions) is saved as a comment on that issue automatically.

---

## Plugin config reference

Set these in NanoClip → Plugin Manager → Telegram → Edit config:

| Field | Type | Description |
|---|---|---|
| `botToken` | string | Bot token from @BotFather |
| `defaultChatId` | string | Numeric chat ID for general notifications |
| `approvalsChatId` | string | Chat ID for approval requests (optional) |
| `errorsChatId` | string | Chat ID for agent error alerts (optional) |
| `escalationChatId` | string | Chat ID for escalation messages (optional) |
| `paperclipPublicUrl` | string | Public URL of NanoClip (adds deep links in messages) |
| `enableCommands` | bool | Enable bot command handling |
| `enableInbound` | bool | Enable reply-to-message → issue comment routing |
| `enableEscalation` | bool | Enable the escalation system |
| `enableMedia` | bool | Enable photo/document handling |
| `topicsEnabled` | bool | Enable Telegram Topics/thread routing |

---

## Full setup guide

See [`docs/telegram-plugin.md`](../telegram-plugin.md) for a complete beginner walkthrough.
