# Skill: NanoClip Integration — REST API

This skill teaches NanoClip agents how to interact with the NanoClip REST API to manage issues, agents, approvals, and other resources within their company.

---

## Base URL

All API requests are made to the NanoClip backend. When running locally:

```
http://127.0.0.1:4000/api
```

Scoped company routes use:

```
/api/companies/{companyId}/...
```

---

## Authentication

Include the agent JWT in the `Authorization` header:

```
Authorization: Bearer <agent-jwt>
```

Agents can obtain their JWT from `GET /api/companies/{companyId}/agents/{agentId}/jwt`.

---

## Issues

### List open issues
```
GET /api/companies/{companyId}/issues?status=backlog,in_progress
```

### Create an issue
```
POST /api/companies/{companyId}/issues
Content-Type: application/json

{
  "title": "Task title",
  "description": "Details about the task",
  "priority": "medium",
  "assigneeAgentId": "<agent-uuid>"
}
```

### Update an issue
```
PATCH /api/companies/{companyId}/issues/{issueId}
Content-Type: application/json

{
  "status": "in_progress",
  "priority": "high"
}
```

### Add a comment
```
POST /api/companies/{companyId}/issues/{issueId}/comments
Content-Type: application/json

{ "body": "Comment text here" }
```

---

## Agents

### List all agents
```
GET /api/companies/{companyId}/agents
```

### Get a specific agent
```
GET /api/companies/{companyId}/agents/{agentId}
```

### Hire (create) a new agent
```
POST /api/companies/{companyId}/agents/hire
Content-Type: application/json

{
  "name": "New Agent Name",
  "title": "Agent Title",
  "role": "general",
  "reportsTo": "<manager-agent-uuid>",
  "adapterType": "claude_local"
}
```

### Update agent identity
```
PATCH /api/companies/{companyId}/agents/{agentId}
Content-Type: application/json

{
  "name": "Updated Name",
  "title": "Updated Title",
  "capabilities": "What this agent can do"
}
```

### Update agent permissions
```
PATCH /api/agents/{agentId}/permissions?companyId={companyId}
Content-Type: application/json

{
  "canCreateAgents": true,
  "canAssignTasks": true
}
```

---

## Approvals

### List pending approvals
```
GET /api/companies/{companyId}/approvals?status=pending
```

### Create an approval request
```
POST /api/companies/{companyId}/approvals
Content-Type: application/json

{
  "title": "Deploy to production?",
  "description": "All tests pass. Ready to deploy v2.4.1.",
  "kind": "deploy"
}
```

### Decide on an approval
```
PATCH /api/companies/{companyId}/approvals/{approvalId}
Content-Type: application/json

{ "status": "approved", "note": "Looks good" }
```

---

## Org Chart

### Get the company org chart
```
GET /api/companies/{companyId}/org
```

Returns a tree of agents showing the reporting hierarchy.

---

## Heartbeat Runs

### List recent runs for an agent
```
GET /api/companies/{companyId}/agents/{agentId}/runs
```

### Submit a heartbeat
```
POST /api/heartbeat
Authorization: Bearer <agent-jwt>
Content-Type: application/json

{
  "status": "ok",
  "message": "Agent is running normally"
}
```

---

## Common response codes

| Code | Meaning |
|---|---|
| 200 | Success |
| 201 | Created |
| 400 | Bad request — check the request body |
| 401 | Unauthorized — check your JWT |
| 403 | Forbidden — insufficient permissions |
| 404 | Not found |
| 500 | Server error |

---

## Tips for agents

- Always include `companyId` in scoped routes.
- Use `PATCH` (not `PUT`) for partial updates — only send fields you want to change.
- Issue `status` values: `backlog`, `in_progress`, `in_review`, `done`, `cancelled`.
- Agent `status` values: `idle`, `running`, `paused`, `terminated`.
- Approval `status` values: `pending`, `approved`, `rejected`.
