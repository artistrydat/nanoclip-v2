# Telegram Plugin — Setup & Usage Guide

The NanoClip Telegram plugin gives you a two-way bridge between your agents and Telegram. You get push notifications when things happen, inline Approve/Reject buttons for agent approvals, bot commands to check status, and the ability to reply to bot messages to create issue comments — all without leaving Telegram.

---

## What it does

| Feature | Description |
|---|---|
| **New issue notification** | Bot sends a message when a new issue is created |
| **Issue done notification** | Bot sends a message when an issue moves to "done" |
| **Approval request** | Bot sends an inline keyboard with ✅ Approve and ❌ Reject buttons |
| **Error alert** | Bot sends a message to a dedicated chat when an agent run fails |
| **Bot commands** | `/status`, `/issues`, `/agents`, `/approve`, `/help` |
| **Inbound replies** | Reply to any bot message to add a comment to the linked issue |

---

## Part 1 — Create a Telegram bot

### Step 1 — Talk to BotFather

1. Open Telegram and search for **@BotFather**
2. Tap **Start** (or send `/start`)
3. Send `/newbot`
4. When asked for a name, type anything — for example: `NanoClip Bot`
5. When asked for a username, it must end in `bot` — for example: `nanoclip_assistant_bot`
6. BotFather will reply with your **Bot Token** — it looks like:
   ```
   7312345678:AAHdqTcvCH1vGWJxfSeofSs3mCQuidGi-EI
   ```
   **Copy and save this token — you will need it in Step 5.**

> Keep your bot token secret. Anyone who has it can control your bot.

---

### Step 2 — Get your Chat ID

You need a **Chat ID** — the number that identifies which Telegram chat the bot should send messages to. This can be a personal chat, a group, or a channel.

**Easiest method — use your personal chat:**

1. Search for your new bot in Telegram and tap **Start**
2. Send any message to the bot (e.g. `hello`)
3. Open this URL in your browser (replace `YOUR_TOKEN` with your actual token):
   ```
   https://api.telegram.org/botYOUR_TOKEN/getUpdates
   ```
4. In the response, look for `"chat":{"id":` — the number after it is your Chat ID:
   ```json
   "chat": { "id": 123456789, "first_name": "Your Name", ... }
   ```
5. Copy that number — for example: `123456789`

**For a group or channel:**

1. Add your bot to the group or channel as an administrator
2. Send a message in the group
3. Open `https://api.telegram.org/botYOUR_TOKEN/getUpdates`
4. Look for `"chat":{"id":` — group IDs are negative numbers, e.g. `-1001234567890`

> **Tip:** You can use the same Chat ID for all notifications, or use different chats for approvals and errors by setting separate IDs in the plugin config.

---

## Part 2 — Install the plugin in NanoClip

### Step 3 — Open Plugin Manager

1. In NanoClip, click **Settings** in the left sidebar
2. Click the **Plugins** tab
3. You will see two sections: **Installed Plugins** and **Available Plugins**

---

### Step 4 — Install the Telegram plugin

1. Under **Available Plugins**, find **Telegram Bot**
2. Click **Install Example**
3. The plugin will appear in **Installed Plugins** with status `disabled`

---

### Step 5 — Configure the plugin

1. Click the **⚙ Settings** button next to the Telegram Bot plugin
2. Fill in the fields:

   | Field | What to enter |
   |---|---|
   | **Bot Token** | The token from BotFather (Step 1) |
   | **Default Chat ID** | Your Chat ID (Step 2) — receives all general notifications |
   | **Approvals Chat ID** | Chat ID for approval requests (can be the same as Default) |
   | **Errors Chat ID** | Chat ID for error alerts (can be the same as Default) |
   | **NanoClip Public URL** | The URL where NanoClip is accessible, e.g. `http://localhost:8080` |
   | **Enable bot commands** | Toggle on — allows `/status`, `/issues`, etc. |
   | **Enable inbound replies** | Toggle on — allows replying to bot messages to create comments |

3. Click **Save**
4. Click **Enable** — the plugin status changes to `ready`

> If you leave Approvals Chat ID or Errors Chat ID blank, those notifications will go to the Default Chat ID.

---

## Part 3 — Test that it works

### Step 6 — Verify the bot responds

Send `/help` to your bot in Telegram. You should receive:

```
NanoClip Bot Commands

/status — system status and recent runs
/issues — list open issues
/agents — list agents and their status
/approve <id> — approve a pending approval by ID
/help — show this help message
```

If the bot does not respond within a few seconds:
- Check that the Bot Token in the plugin config is correct
- Check that you clicked **Enable** and the status shows `ready`
- Try clicking **Health Check** in the plugin settings to see the error

---

### Step 7 — Create a test issue

1. In NanoClip, create a new issue in any project
2. Within a few seconds, you should receive a Telegram notification like:

   ```
   📋 New issue created
   #42 · Fix the login page
   ```

If you don't receive the notification:
- Make sure your Chat ID is correct (check Step 2)
- Make sure the bot has permission to message you (start the bot first)

---

## Bot commands reference

All commands work in any chat where the bot is a member.

| Command | What it does |
|---|---|
| `/help` | Shows all available commands |
| `/status` | Shows the number of agents and the last 3 run results |
| `/issues` | Lists up to 10 open issues with their IDs and titles |
| `/agents` | Lists all agents with their current status (idle / running / paused) |
| `/approve <id>` | Approves a pending approval by its ID |

**Example:**
```
/approve a1b2c3d4
```

---

## Approval flow

When an agent requests approval, the bot sends a message like:

```
⏳ Approval Required

Agent "Research Bot" is requesting approval.
Task: Summarize quarterly report

ID: a1b2c3d4
```

With two inline buttons: **✅ Approve** and **❌ Reject**.

Tap either button directly in Telegram — no need to open NanoClip. The bot will update the message to confirm your choice.

You can also approve from the command line:
```
/approve a1b2c3d4
```

---

## Replying to bot messages

When inbound replies are enabled, you can reply to any bot notification to add a comment to the linked issue.

1. The bot sends a notification about an issue
2. In Telegram, **long-press the message** and tap **Reply**
3. Type your comment and send it
4. NanoClip receives the reply and adds it as a comment on the issue
5. The assigned agent will see the comment and respond on the next run

This is useful for giving feedback to agents without opening NanoClip.

---

## Notifications reference

| Event | When it's sent | Where |
|---|---|---|
| New issue | When any issue is created in NanoClip | Default Chat ID |
| Issue done | When an issue moves to the "done" column | Default Chat ID |
| Approval request | When an agent requests a human approval | Approvals Chat ID |
| Agent error | When an agent run fails | Errors Chat ID |

---

## Troubleshooting

### Bot does not respond to commands

- Make sure **Enable bot commands** is toggled on in the plugin settings
- Make sure you tapped Start in the bot's chat before sending commands
- In a group, make sure the bot is an admin and that group privacy mode is disabled (in BotFather: `/mybots → your bot → Bot Settings → Group Privacy → Turn off`)

---

### Notifications are not arriving

- Check that the Chat ID is correct — copy it fresh from `getUpdates` (Step 2)
- For groups, the ID is a negative number — make sure you copied the full number including the minus sign
- Click **Health Check** in the plugin settings to see if there's a connection error

---

### "Unauthorized" error in Health Check

Your Bot Token is wrong. Copy it again directly from BotFather — there should be no spaces and it should look like `7312345678:AAHd...`.

---

### Inbound replies are not creating comments

- Make sure **Enable inbound replies** is on
- You must **Reply** to a specific bot message (not just send a new message)
- The issue linked to that notification must still be open

---

### The plugin shows "disabled" after enabling

Click **Health Check** — the error message will tell you what is wrong. Common causes:
- Invalid bot token
- Bot has not been started by the user (open the bot in Telegram and tap Start)

---

## Removing the plugin

1. Go to **Settings → Plugins**
2. Click **Disable** next to Telegram Bot
3. Click **Uninstall**

This removes the plugin and stops all notifications. Your bot still exists in Telegram — delete it in BotFather with `/deletebot` if you no longer need it.

---

## Security notes

- The bot token gives full control of your bot. Do not share it publicly.
- NanoClip stores the token encrypted in your local database.
- The plugin uses long-polling (not webhooks) so no public URL is required for the bot to receive commands.
- Inbound messages are only accepted from Telegram's servers — no external access is possible.
