# Skill: MariaDB — Local Database Operations

This skill teaches NanoClip agents how to interact with the local MariaDB database that runs on the same device (phone, server, or desktop). All data stored by NanoClip — issues, agents, approvals, comments, plugins, logs — lives here.

---

## Connection details

NanoClip connects to MariaDB using the DSN from `.env`:

```
MARIADB_DSN=nanoclip:nanoclip123@tcp(127.0.0.1:3306)/nanoclip?charset=utf8mb4&parseTime=True&loc=UTC
```

Key connection parameters:
| Parameter | Value |
|---|---|
| Host | `127.0.0.1` (always use IP, not `localhost`) |
| Port | `3306` |
| Database | `nanoclip` |
| User | `nanoclip` |
| Password | Set during setup (default: `nanoclip123`) |
| Charset | `utf8mb4` (full Unicode including emoji) |

> **Why `127.0.0.1` not `localhost`?** On modern MariaDB, `localhost` uses UNIX socket authentication. Using `127.0.0.1` forces TCP which accepts password authentication.

---

## Core tables

| Table | What it stores |
|---|---|
| `users` | NanoClip user accounts |
| `sessions` | Active login sessions |
| `companies` | Agent companies / teams |
| `agents` | Individual AI agents |
| `issues` | Tasks assigned to agents |
| `issue_comments` | Comments on issues |
| `issue_attachments` | File attachments on issues |
| `heartbeat_runs` | Agent execution runs |
| `approvals` | Human-approval requests |
| `plugins` | Installed plugins and their config |
| `plugin_logs` | Plugin activity logs |
| `routines` | Scheduled/triggered routines |
| `projects` | Projects grouping issues |
| `labels` | Issue labels |
| `activity_logs` | Audit trail |
| `company_secrets` | Encrypted secrets per company |
| `company_skills` | Skill definitions per company |

---

## Connecting from the command line (Termux / Linux)

```bash
mariadb -h 127.0.0.1 -P 3306 -u nanoclip -pnanoclip123 nanoclip
```

---

## Reading data

### List all open issues
```sql
SELECT identifier, title, status, priority
FROM issues
WHERE status NOT IN ('done', 'cancelled')
ORDER BY created_at DESC
LIMIT 20;
```

### Get issues for a specific company
```sql
SELECT i.identifier, i.title, i.status, a.name AS agent
FROM issues i
LEFT JOIN agents a ON a.id = i.assignee_agent_id
WHERE i.company_id = '<company-uuid>'
  AND i.status NOT IN ('done', 'cancelled')
ORDER BY i.created_at DESC;
```

### Get all agents and their status
```sql
SELECT name, status, adapter_type, created_at
FROM agents
ORDER BY name;
```

### Get pending approvals
```sql
SELECT id, title, status, created_at
FROM approvals
WHERE status = 'pending'
ORDER BY created_at ASC;
```

### Get recent agent runs (last 10)
```sql
SELECT ar.id, a.name AS agent, ar.status, ar.started_at, ar.completed_at
FROM heartbeat_runs ar
JOIN agents a ON a.id = ar.agent_id
ORDER BY ar.created_at DESC
LIMIT 10;
```

### Read plugin config
```sql
SELECT plugin_key, enabled, status, config
FROM plugins
WHERE plugin_key = 'telegram-bot';
```

### Get recent plugin logs
```sql
SELECT pl.level, pl.message, pl.created_at
FROM plugin_logs pl
JOIN plugins p ON p.id = pl.plugin_id
WHERE p.plugin_key = 'telegram-bot'
ORDER BY pl.created_at DESC
LIMIT 20;
```

---

## Creating records

### Create an issue
```sql
INSERT INTO issues (
  id, company_id, project_id, title, description,
  status, priority, origin_kind, created_at, updated_at
) VALUES (
  UUID(),
  '<company-uuid>',
  '<project-uuid>',
  'Fix login page crash',
  'Users report a crash when submitting the login form on Android.',
  'backlog',
  'high',
  'manual',
  NOW(),
  NOW()
);
```

### Add a comment to an issue
```sql
INSERT INTO issue_comments (
  id, company_id, issue_id, body, created_at, updated_at
) VALUES (
  UUID(),
  '<company-uuid>',
  '<issue-uuid>',
  'Reproduced the crash. Stack trace points to null pointer in AuthController.',
  NOW(),
  NOW()
);
```

### Create an approval request
```sql
INSERT INTO approvals (
  id, company_id, agent_id, title, description,
  status, created_at, updated_at
) VALUES (
  UUID(),
  '<company-uuid>',
  '<agent-uuid>',
  'Deploy to production?',
  'All tests pass. Ready to deploy version 2.4.1 to production servers.',
  'pending',
  NOW(),
  NOW()
);
```

### Write a plugin log entry
```sql
INSERT INTO plugin_logs (
  id, plugin_id, level, message, created_at
) VALUES (
  UUID(),
  (SELECT id FROM plugins WHERE plugin_key = 'telegram-bot' LIMIT 1),
  'info',
  'Notification sent for issue PROJ-42',
  NOW()
);
```

---

## Updating records

### Mark an issue as done
```sql
UPDATE issues
SET status = 'done',
    completed_at = NOW(),
    updated_at = NOW()
WHERE identifier = 'PROJ-42';
```

### Update issue priority
```sql
UPDATE issues
SET priority = 'urgent',
    updated_at = NOW()
WHERE id = '<issue-uuid>';
```

### Approve a pending approval
```sql
UPDATE approvals
SET status = 'approved',
    decided_at = NOW(),
    decision_note = 'Approved via automated review',
    updated_at = NOW()
WHERE id = '<approval-uuid>';
```

### Change agent status
```sql
UPDATE agents
SET status = 'idle',
    updated_at = NOW()
WHERE id = '<agent-uuid>';
```

### Update plugin config field
```sql
UPDATE plugins
SET config = JSON_SET(config, '$.defaultChatId', '123456789'),
    updated_at = NOW()
WHERE plugin_key = 'telegram-bot';
```

---

## Deleting records

### Delete a comment
```sql
DELETE FROM issue_comments
WHERE id = '<comment-uuid>';
```

### Delete old plugin logs (keep last 1000)
```sql
DELETE FROM plugin_logs
WHERE id NOT IN (
  SELECT id FROM (
    SELECT id FROM plugin_logs ORDER BY created_at DESC LIMIT 1000
  ) AS keep
);
```

### Soft-delete an issue (set cancelled)
```sql
UPDATE issues
SET status = 'cancelled',
    cancelled_at = NOW(),
    updated_at = NOW()
WHERE id = '<issue-uuid>';
```

---

## Table management

### Create a custom table for agent state
```sql
CREATE TABLE IF NOT EXISTS agent_state (
  id          CHAR(36) PRIMARY KEY DEFAULT (UUID()),
  agent_id    CHAR(36) NOT NULL,
  state_key   VARCHAR(255) NOT NULL,
  state_value LONGTEXT,
  updated_at  DATETIME NOT NULL DEFAULT NOW(),
  UNIQUE KEY agent_state_key (agent_id, state_key),
  INDEX idx_agent_state_agent (agent_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### Add a column to an existing table
```sql
ALTER TABLE issues
ADD COLUMN external_ticket_id VARCHAR(255) NULL AFTER identifier;
```

### Check table structure
```sql
DESCRIBE issues;
```

### Show all tables in the database
```sql
SHOW TABLES;
```

### Check database size
```sql
SELECT
  table_name,
  ROUND(((data_length + index_length) / 1024 / 1024), 2) AS size_mb
FROM information_schema.tables
WHERE table_schema = 'nanoclip'
ORDER BY size_mb DESC;
```

---

## Useful queries for agents

### Find which agent is assigned to an issue
```sql
SELECT a.name, a.status, a.adapter_type
FROM agents a
JOIN issues i ON i.assignee_agent_id = a.id
WHERE i.identifier = 'PROJ-42';
```

### Count issues by status for a company
```sql
SELECT status, COUNT(*) AS count
FROM issues
WHERE company_id = '<company-uuid>'
GROUP BY status;
```

### Find issues created in the last 24 hours
```sql
SELECT identifier, title, status, created_at
FROM issues
WHERE created_at > NOW() - INTERVAL 24 HOUR
ORDER BY created_at DESC;
```

### Get company secrets (for agent use)
```sql
SELECT secret_key, secret_value
FROM company_secrets
WHERE company_id = '<company-uuid>'
  AND secret_key = 'OPENROUTER_API_KEY';
```

### Search issues by keyword
```sql
SELECT identifier, title, status
FROM issues
WHERE (title LIKE '%payment%' OR description LIKE '%payment%')
  AND status NOT IN ('done', 'cancelled')
ORDER BY created_at DESC;
```

---

## JSON column operations

Many NanoClip tables store JSON in TEXT columns. MariaDB supports JSON functions:

### Read a JSON field
```sql
SELECT JSON_EXTRACT(config, '$.botToken') AS bot_token
FROM plugins
WHERE plugin_key = 'telegram-bot';
```

### Update one field inside a JSON column
```sql
UPDATE plugins
SET config = JSON_SET(config, '$.enableCommands', TRUE)
WHERE plugin_key = 'telegram-bot';
```

### Check if a JSON key exists
```sql
SELECT plugin_key
FROM plugins
WHERE JSON_CONTAINS_PATH(config, 'one', '$.botToken') = 1;
```

---

## Running queries from inside NanoClip

The Go server uses GORM with a MariaDB driver. All database access goes through GORM models. Raw SQL can be executed via:

```go
db.Raw("SELECT * FROM issues WHERE status = ?", "backlog").Scan(&results)
db.Exec("UPDATE issues SET status = ? WHERE id = ?", "done", issueID)
```

---

## Backup and restore

### Backup the full database
```bash
mariadb-dump -h 127.0.0.1 -u nanoclip -pnanoclip123 nanoclip > nanoclip-backup.sql
```

### Restore from backup
```bash
mariadb -h 127.0.0.1 -u nanoclip -pnanoclip123 nanoclip < nanoclip-backup.sql
```

### Export just the issues table
```bash
mariadb-dump -h 127.0.0.1 -u nanoclip -pnanoclip123 nanoclip issues > issues-backup.sql
```

---

## Common error fixes

| Error | Cause | Fix |
|---|---|---|
| `Access denied for 'nanoclip'@'localhost'` | UNIX socket auth | Use `-h 127.0.0.1` |
| `Table doesn't exist` | Migration not run | Start the NanoClip server — it runs migrations on startup |
| `Data too long for column` | String exceeds varchar limit | Use `LONGTEXT` for large fields |
| `Duplicate entry for key` | UUID collision or unique constraint | Generate a new UUID |
| `Can't connect to MySQL server` | MariaDB not running | Run `mariadbd-safe --datadir="$PREFIX/var/lib/mysql" &` |
