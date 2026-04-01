// Shim replacing the deleted @nanoclip/shared workspace package.
// All runtime values that the UI depends on are defined here.

// ─── Icon names ───────────────────────────────────────────────────────────────
export const AGENT_ICON_NAMES = [
  "bot", "cpu", "brain", "zap", "rocket", "code", "terminal", "shield",
  "eye", "search", "wrench", "hammer", "lightbulb", "sparkles", "star",
  "heart", "flame", "bug", "cog", "database", "globe", "lock", "mail",
  "message-square", "file-code", "git-branch", "package", "puzzle",
  "target", "wand", "atom", "circuit-board", "radar", "swords", "telescope",
  "microscope", "crown", "gem", "hexagon", "pentagon", "fingerprint",
] as const;

export type AgentIconName = (typeof AGENT_ICON_NAMES)[number];

// ─── Agent roles ──────────────────────────────────────────────────────────────
export const AGENT_ROLES = [
  "ceo", "manager", "engineer", "researcher", "analyst",
  "coordinator", "auditor", "general",
] as const;

export type AgentRole = (typeof AGENT_ROLES)[number];

export const AGENT_ROLE_LABELS: Record<string, string> = {
  ceo: "CEO",
  manager: "Manager",
  engineer: "Engineer",
  researcher: "Researcher",
  analyst: "Analyst",
  coordinator: "Coordinator",
  auditor: "Auditor",
  general: "General",
};

// ─── Adapter types ────────────────────────────────────────────────────────────
export const AGENT_ADAPTER_TYPES = [
  "claude_local", "codex_local", "gemini_local", "opencode_local",
  "pi_local", "openclaw_gateway", "cursor", "hermes_local", "ollama_local",
  "openrouter_local", "process", "http",
] as const;

export type AgentAdapterType = (typeof AGENT_ADAPTER_TYPES)[number];

// ─── Goal constants ───────────────────────────────────────────────────────────
export const GOAL_STATUSES = ["active", "achieved", "missed", "cancelled"] as const;
export type GoalStatus = (typeof GOAL_STATUSES)[number];

export const GOAL_LEVELS = ["company", "team", "individual"] as const;
export type GoalLevel = (typeof GOAL_LEVELS)[number];

// ─── Issue / Inbox constants ──────────────────────────────────────────────────
export const INBOX_MINE_ISSUE_STATUS_FILTER: string[] = ["open", "in_progress"];

// ─── Project constants ────────────────────────────────────────────────────────
export const PROJECT_COLORS = [
  "#ef4444", "#f97316", "#eab308", "#22c55e", "#14b8a6",
  "#3b82f6", "#8b5cf6", "#ec4899", "#64748b", "#0ea5e9",
] as const;

// ─── Plugin launcher ──────────────────────────────────────────────────────────
export const PLUGIN_LAUNCHER_BOUNDS = ["compact", "wide", "full", "inline", "default"] as const;
export type PluginLauncherBounds = (typeof PLUGIN_LAUNCHER_BOUNDS)[number] | string;

// ─── URL key helpers ──────────────────────────────────────────────────────────
function slugify(str: string | null | undefined): string {
  return (str ?? "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 40);
}

export function deriveAgentUrlKey(name: string | null | undefined, id: string): string {
  const slug = slugify(name);
  const shortId = id.replace(/-/g, "").slice(0, 8);
  return slug ? `${slug}-${shortId}` : shortId;
}

export function deriveProjectUrlKey(name: string | null | undefined, id: string): string {
  const slug = slugify(name);
  const shortId = id.replace(/-/g, "").slice(0, 8);
  return slug ? `${slug}-${shortId}` : shortId;
}

export function normalizeAgentUrlKey(key: string): string {
  return key.toLowerCase().replace(/[^a-z0-9-]/g, "-");
}

export function isUuidLike(str: string): boolean {
  return /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(str);
}

// ─── Mention href helpers ─────────────────────────────────────────────────────
const AGENT_MENTION_PREFIX = "mention://agent/";
const PROJECT_MENTION_PREFIX = "mention://project/";

export function buildAgentMentionHref(agentId: string, icon?: string | null): string {
  const base = `${AGENT_MENTION_PREFIX}${agentId}`;
  return icon ? `${base}?icon=${encodeURIComponent(icon)}` : base;
}

export function buildProjectMentionHref(projectId: string, color?: string | null): string {
  const base = `${PROJECT_MENTION_PREFIX}${projectId}`;
  return color ? `${base}?color=${encodeURIComponent(color)}` : base;
}

export function parseAgentMentionHref(href: string): { agentId: string; icon: string | null } | null {
  if (!href.startsWith(AGENT_MENTION_PREFIX)) return null;
  const rest = href.slice(AGENT_MENTION_PREFIX.length);
  const [agentId, query] = rest.split("?");
  const icon = query ? new URLSearchParams(query).get("icon") : null;
  return { agentId, icon };
}

export function parseProjectMentionHref(href: string): { projectId: string; color: string | null } | null {
  if (!href.startsWith(PROJECT_MENTION_PREFIX)) return null;
  const rest = href.slice(PROJECT_MENTION_PREFIX.length);
  const [projectId, query] = rest.split("?");
  const color = query ? new URLSearchParams(query).get("color") : null;
  return { projectId, color };
}

// ─── Types ────────────────────────────────────────────────────────────────────
// These are TypeScript-only shapes matching the Go API responses.

export interface User {
  id: string;
  name: string;
  email: string;
  emailVerified: boolean;
  createdAt: string;
  updatedAt: string;  [key: string]: any;

}

export interface Company {
  id: string;
  name: string;
  mission: string | null;
  status: string;
  logoUrl: string | null;
  issuePrefix: string;
  brandColor: string | null;
  leadAgentId: string | null;
  budgetMonthlyCents: number | null;
  heartbeatEnabled: boolean;
  schedulerActive: boolean;
  legacyBootstrapPromptTemplateActive: boolean;
  legacyPlanDocument: string | null;
  legacyPromptTemplateActive: boolean;
  executionWorkspacePolicy: string | null;
  executionWorkspacePreference: string | null;
  executionWorkspaceSettings: Record<string, unknown> | null;
  createdAt: string;
  updatedAt: string;
  [key: string]: any;
}

export interface Agent {
  id: string;
  companyId: string;
  name: string;
  title: string | null;
  role: string;
  capabilities: string | null;
  iconName: string | null;
  urlKey: string | null;
  adapterType: string;
  adapterConfig: Record<string, any>;
  reportsToId: string | null;
  status: string;
  runtimeState: AgentRuntimeState | null;
  createdAt: string;
  updatedAt: string;
  [key: string]: any;
}

export interface AgentRuntimeState {
  status: string;
  lastRunAt: string | null;
  currentRunId: string | null;  [key: string]: any;

}

export interface AgentSkillEntry {
  key: string;
  name: string;
  description: string | null;  [key: string]: any;

}

export interface AgentSkillSnapshot {
  skills: AgentSkillEntry[];  [key: string]: any;

}

export interface AgentInstructionsBundle {
  systemPrompt: string;
  files: AgentInstructionsFileDetail[];  [key: string]: any;

}

export interface AgentInstructionsFileDetail {
  path: string;
  content: string;  [key: string]: any;

}

export interface AgentConfigRevision {
  id: string;
  agentId: string;
  config: Record<string, unknown>;
  createdAt: string;  [key: string]: any;

}

export interface AgentDetail extends Agent {
  skills: AgentSkillEntry[];  [key: string]: any;

}

export interface AgentTaskSession {
  id: string;
  agentId: string;
  transcript: TranscriptEntry[];
  createdAt: string;  [key: string]: any;

}

export type AgentKeyCreated = {
  id: string;
  key: string;
  token?: string;
  name: string;
  createdAt: string;
  [key: string]: any;
};

export interface Project {
  id: string;
  companyId: string;
  name: string;
  description: string | null;
  color: string | null;
  urlKey: string | null;
  status: string;
  workspaces?: Array<{ id: string; isPrimary: boolean; [key: string]: any }>;
  primaryWorkspace?: ProjectWorkspace | null;
  executionWorkspacePolicy?: {
    enabled?: boolean;
    defaultMode?: string | null;
    defaultProjectWorkspaceId?: string | null;
    [key: string]: any;
  } | null;
  targetDate?: string | null;
  leadAgentId?: string | null;
  createdAt: string;
  updatedAt: string;
  [key: string]: any;
}

export interface ProjectWorkspace {
  id: string;
  projectId?: string;
  agentId?: string | null;
  name?: string;
  repoUrl?: string | null;
  basePath?: string | null;
  status?: string;
  isPrimary?: boolean;
  createdAt?: string;
  updatedAt?: string;
  [key: string]: any;
}

export interface WorkspaceOperation {
  id: string;
  workspaceId: string;
  kind: string;
  status: string;
  createdAt: string;  [key: string]: any;

}

export interface ExecutionWorkspace {
  id: string;
  agentId: string;
  issueId: string | null;
  projectId: string | null;
  projectWorkspaceId: string | null;
  repoRef: string | null;
  worktreePath: string | null;
  status: string;
  createdAt: string;
  updatedAt: string;  [key: string]: any;

}

export interface ExecutionWorkspaceCloseReadiness {
  ready: boolean;
  reason: string | null;  [key: string]: any;

}

export interface Goal {
  id: string;
  companyId: string;
  title: string;
  description: string | null;
  status: GoalStatus;
  level: GoalLevel;
  parentId: string | null;
  projectId: string | null;
  createdAt: string;
  updatedAt: string;
  [key: string]: any;
}

export interface Issue {
  id: string;
  companyId: string;
  identifier: string | null;
  title: string;
  description: string | null;
  status: string;
  priority: string | null;
  assignedToId: string | null;
  projectId: string | null;
  goalId: string | null;
  createdAt: string | Date;
  updatedAt: string | Date;
  [key: string]: any;
}

export interface IssueComment {
  id: string;
  issueId: string;
  authorId: string | null;
  body: string;
  role: string;
  delta: boolean;
  createdAt: string;
  updatedAt: string;  [key: string]: any;

}

export interface IssueAttachment {
  id: string;
  issueId: string;
  name: string;
  url: string;
  mimeType: string | null;
  createdAt: string;  [key: string]: any;

}

export interface IssueDocument {
  id: string;
  issueId: string;
  title: string;
  content: string;
  createdAt: string;  [key: string]: any;

}

export interface Approval {
  id: string;
  companyId: string;
  agentId: string;
  issueId: string | null;
  kind: string;
  status: string;
  body: string | null;
  diff: string | null;
  createdAt: string;
  updatedAt: string;  [key: string]: any;

}

export interface ApprovalComment {
  id: string;
  approvalId: string;
  authorId: string | null;
  body: string;
  role: string;
  createdAt: string;  [key: string]: any;

}

export interface HeartbeatRun {
  id: string;
  agentId: string;
  companyId: string;
  status: string;
  startedAt: string;
  finishedAt: string | null;
  exitCode: number | null;
  errorMessage: string | null;
  costUsd: number | null;
  createdAt: string;  [key: string]: any;

}

export interface InstanceSchedulerHeartbeatAgent {
  agentId: string;
  companyId: string;
  nextRunAt: string | null;  [key: string]: any;

}

export interface SidebarBadges {
  pendingApprovals: number;
  openIssues: number;  [key: string]: any;

}

export interface DashboardSummary {
  openIssues: number;
  pendingApprovals: number;
  activeAgents: number;
  recentRuns: HeartbeatRun[];  [key: string]: any;

}

export interface ActivityEvent {
  id: string;
  companyId: string;
  agentId: string | null;
  kind: string;
  payload: Record<string, any>;
  createdAt: string;
  [key: string]: any;
}

export interface LiveEvent {
  type: string;
  payload: Record<string, unknown>;  [key: string]: any;

}

export interface JoinRequest {
  id: string;
  companyId: string;
  token: string;
  usedAt: string | null;
  createdAt: string;  [key: string]: any;

}

export interface CompanySecret {
  id: string;
  name: string;
  provider: string | null;
  createdAt: string;  [key: string]: any;

}

export interface SecretProviderDescriptor {
  id: string;
  label: string;
  description: string | null;  [key: string]: any;

}

export type SecretProvider = SecretProviderDescriptor;

export interface RoutineTrigger {
  id: string;
  routineId: string;
  kind: string;
  config: Record<string, unknown>;  [key: string]: any;

}

export interface PluginRecord {
  id: string;
  name: string;
  version: string;
  enabled: boolean;  [key: string]: any;

}

export interface AssetImage {
  id: string;
  url: string;
  name: string;  [key: string]: any;

}

export type BillingType =
  | "metered_api" | "subscription_included" | "subscription_overage"
  | "credits" | "fixed" | "unknown";

export type FinanceDirection = "credit" | "debit";

export type FinanceEventKind =
  | "inference_charge" | "platform_fee" | "credit_purchase" | "credit_refund"
  | "credit_expiry" | "byok_fee" | "gateway_overhead" | "log_storage_charge"
  | "logpush_charge" | "provisioned_capacity_charge" | "training_charge"
  | "custom_model_import_charge" | "custom_model_storage_charge"
  | "manual_adjustment";

export interface FinanceEvent {
  id: string;
  kind: FinanceEventKind;
  direction: FinanceDirection;
  amountCents: number;
  billingType: BillingType;
  createdAt: string;  [key: string]: any;

}

export interface FinanceByBiller {
  billerId: string;
  billerName: string;
  amountCents: number;  [key: string]: any;

}

export interface FinanceByKind {
  kind: FinanceEventKind;
  amountCents: number;  [key: string]: any;

}

export interface CostByBiller {
  billerId: string;
  billerName: string;
  costCents: number;  [key: string]: any;

}

export interface CostByProviderModel {
  provider: string;
  model: string;
  costCents: number;  [key: string]: any;

}

export interface CostWindowSpendRow {
  windowStart: string;
  costCents: number;  [key: string]: any;

}

export interface QuotaWindow {
  source: string;
  totalTokens: number;
  usedTokens: number;
  resetAt: string | null;  [key: string]: any;

}

export interface BudgetPolicySummary {
  monthlyCents?: number;
  spentCents?: number;
  [key: string]: any;
}

export interface BudgetIncident {
  id: string;
  agentId: string;
  kind: string;
  createdAt: string;  [key: string]: any;

}

export interface CompanyPortabilitySidebarOrder {
  items?: string[];
  [key: string]: any;
}

export type CompanyPortabilityFileEntry = string | {
  path?: string;
  content?: string;
  encoding?: string;
  data?: string;
  contentType?: string;
  [key: string]: any;
};

export interface CompanyPortabilityIssueManifestEntry {
  id: string;
  identifier: string | null;
  title: string;  [key: string]: any;

}

export interface AdapterEnvironmentTestResult {
  ok: boolean;
  message: string | null;
  status?: string;
  testedAt?: string;
  checks?: Array<{
    name: string;
    ok: boolean;
    message?: string;
    code?: string;
    level?: string;
    detail?: string;
    hint?: string;
    [key: string]: any;
  }>;
  [key: string]: any;
}

// TranscriptEntry used by the live run view
export type TranscriptEntry = {
  id?: string;
  kind: string;
  role?: string;
  content?: string;
  delta?: boolean;
  thinking?: string;
  toolName?: string;
  toolInput?: Record<string, unknown>;
  toolResult?: unknown;
  error?: string;
  createdAt?: string;
};

// ─── Budget types ─────────────────────────────────────────────────────────────
export interface BudgetOverview {
  agentId: string;
  monthlyCents: number | null;
  spentCents: number;
  usedPercent: number;
  warnPercent: number;
  incidents: BudgetIncident[];
  [key: string]: any;
}

export interface BudgetPolicyUpsertInput {
  monthlyCents?: number | null;
  warnPercent?: number;
  [key: string]: any;
}

export interface BudgetIncidentResolutionInput {
  note?: string;
  [key: string]: any;
}

// ─── Company portability types ────────────────────────────────────────────────
export type CompanyPortabilityCollisionStrategy = "skip" | "overwrite" | "error" | "rename" | string;

export interface CompanyPortabilitySource {
  type: string;
  rootPath?: string;
  url?: string;
  files?: Record<string, CompanyPortabilityFileEntry> | CompanyPortabilityFileEntry[];
  [key: string]: any;
}

export interface CompanyPortabilityManifest {
  version: string;
  companyId: string;
  exportedAt: string;
  [key: string]: any;
}

export interface CompanyPortabilityExportRequest {
  includeSecrets?: boolean;
  [key: string]: any;
}

export interface CompanyPortabilityExportPreviewResult {
  manifest: CompanyPortabilityManifest;
  files: Record<string, CompanyPortabilityFileEntry>;
  rootPath?: string;
  paperclipExtensionPath?: string;
  [key: string]: any;
}

export interface CompanyPortabilityExportResult {
  manifest: CompanyPortabilityManifest;
  files: Record<string, CompanyPortabilityFileEntry>;
  rootPath?: string;
  paperclipExtensionPath?: string;
  [key: string]: any;
}

export interface CompanyPortabilityPreviewRequest {
  collisionStrategy?: CompanyPortabilityCollisionStrategy;
  [key: string]: any;
}

export interface CompanyPortabilityPreviewResult {
  warnings: string[];
  errors: string[];
  [key: string]: any;
}

export interface CompanyPortabilityImportRequest {
  collisionStrategy?: CompanyPortabilityCollisionStrategy;
  [key: string]: any;
}

export interface CompanyPortabilityImportResult {
  importedCount?: number;
  skippedCount?: number;
  errors?: string[];
  company?: { id: string; [key: string]: any };
  agents?: { slug: string; id: string; [key: string]: any }[];
  projects?: { slug: string; id: string; [key: string]: any }[];
  [key: string]: any;
}

export interface UpdateCompanyBranding {
  brandColor?: string | null;
  logoUrl?: string | null;
  [key: string]: any;
}

// ─── Company skills types ─────────────────────────────────────────────────────
export type CompanySkillSourceBadge = string;

export interface CompanySkillUpdateStatus {
  trackingRef?: string;
  supported?: boolean;
  hasUpdate?: boolean;
  reason?: string;
  status?: string;
  [key: string]: any;
}

export interface CompanySkill {
  id: string;
  companyId: string;
  name: string;
  key: string;
  description: string | null;
  status: string;
  createdAt: string;
  updatedAt: string;
  [key: string]: any;
}

export interface CompanySkillListItem extends CompanySkill {}

export interface CompanySkillFileDetail {
  id: string;
  skillId: string;
  path: string;
  content: string;
  isEntryFile: boolean;
  byteSize: number;
  createdAt: string;
  updatedAt: string;
  [key: string]: any;
}

export interface CompanySkillFileInventoryEntry {
  path: string;
  byteSize: number;
  isEntryFile: boolean;  [key: string]: any;

}

export interface CompanySkillDetail extends CompanySkill {
  files: CompanySkillFileDetail[];  [key: string]: any;

}

export interface CompanySkillCreateRequest {
  name: string;
  key?: string;
  slug?: string;
  description?: string | null;
  entryFile?: string;
  [key: string]: any;
}

export interface CompanySkillProjectScanRequest {
  projectPath?: string;
  [key: string]: any;
}

export interface CompanySkillProjectScanResult {
  skills: CompanySkillFileInventoryEntry[];
  [key: string]: any;
}

export interface CompanySkillImportResult {
  importedCount: number;
  skippedCount: number;
  [key: string]: any;
}

// ─── Cost types ───────────────────────────────────────────────────────────────
export interface CostByAgent {
  agentId: string;
  agentName: string;
  costCents: number;  [key: string]: any;

}

export interface CostByAgentModel {
  agentId: string;
  agentName: string;
  model: string;
  costCents: number;  [key: string]: any;

}

export interface CostByProject {
  projectId: string;
  projectName: string;
  costCents: number;  [key: string]: any;

}

export interface CostSummary {
  totalCostCents: number;
  byAgent: CostByAgent[];
  byProject: CostByProject[];
  byProviderModel: CostByProviderModel[];
  [key: string]: any;
}

export interface FinanceSummary {
  creditCents: number;
  debitCents: number;
  netCents: number;
  [key: string]: any;
}

// ─── Document types ───────────────────────────────────────────────────────────
export interface DocumentRevision {
  id: string;
  documentId: string;
  content: string;
  revisionNumber: number;
  createdAt: string;
  [key: string]: any;
}

// ─── Environment binding types ────────────────────────────────────────────────
export interface EnvBinding {
  id?: string;
  name?: string;
  type?: string;
  value?: string;
  secretId?: string | null;
  version?: string;
  [key: string]: any;
}

// ─── Instance settings types ──────────────────────────────────────────────────
export interface InstanceGeneralSettings {
  instanceName: string | null;
  allowSignUp: boolean;
  [key: string]: any;
}

export interface PatchInstanceGeneralSettings {
  instanceName?: string | null;
  allowSignUp?: boolean;
  [key: string]: any;
}

export interface InstanceExperimentalSettings {
  [key: string]: any;
}

export interface PatchInstanceExperimentalSettings {
  [key: string]: any;
}

// ─── Issue extended types ─────────────────────────────────────────────────────
export interface IssueLabel {
  id: string;
  companyId: string;
  name: string;
  color: string | null;
  createdAt: string;
  [key: string]: any;
}

export interface IssueWorkProduct {
  id: string;
  issueId: string;
  kind: string;
  content: string;
  createdAt: string;
  [key: string]: any;
}

// ─── Plugin types ─────────────────────────────────────────────────────────────
export type PluginBridgeErrorCode = string;
export type PluginStatus = "active" | "inactive" | "error";
export type PluginLauncherPlacementZone = string;
export type PluginUiSlotType = string;
export type PluginUiSlotEntityType = string;

export interface PluginConfig {
  id: string;
  name: string;
  version: string;
  enabled: boolean;
  config: Record<string, unknown>;
  [key: string]: any;
}

export interface PluginLauncherDeclaration {
  id: string;
  pluginId: string;
  placementZone: PluginLauncherPlacementZone;
  label: string;
  icon?: string;
  [key: string]: any;
}

export interface PluginLauncherRenderEnvironment {
  url: string;
  bounds: PluginLauncherBounds;
  [key: string]: any;
}

export interface PluginLauncherRenderContextSnapshot {
  companyId?: string;
  agentId?: string | null;
  issueId?: string | null;
  environment?: PluginLauncherRenderEnvironment;
  launcherId?: string;
  bounds?: PluginLauncherBounds;
  [key: string]: any;
}

export interface PluginUiSlotDeclaration {
  id: string;
  pluginId: string;
  slotType: PluginUiSlotType;
  entityType: PluginUiSlotEntityType;
  [key: string]: any;
}

// ─── Provider quota types ─────────────────────────────────────────────────────
export interface ProviderQuotaResult {
  provider: string;
  quotas: QuotaWindow[];
  [key: string]: any;
}

// ─── Routine types ────────────────────────────────────────────────────────────
export interface Routine {
  id: string;
  companyId: string;
  name: string;
  agentId: string | null;
  status: string;
  paused: boolean;
  createdAt: string;
  updatedAt: string;
  [key: string]: any;
}

export interface RoutineListItem extends Routine {}

export interface RoutineDetail extends Routine {
  triggers: RoutineTrigger[];  [key: string]: any;

}

export interface RoutineRun {
  id: string;
  routineId: string;
  status: string;
  startedAt: string;
  finishedAt: string | null;
  createdAt: string;
  [key: string]: any;
}

export interface RoutineRunSummary {
  total: number;
  succeeded: number;
  failed: number;
  [key: string]: any;
}

export interface RoutineTriggerSecretMaterial {
  secret: string;
  [key: string]: any;
}

// ─── Additional type aliases for API compatibility ────────────────────────────
export type HeartbeatRunEvent = HeartbeatRun;

export interface UpsertIssueDocument {
  title?: string;
  content?: string;
  [key: string]: any;
}

export interface CompanyPortabilityAdapterOverride {
  adapterType?: string;
  adapterConfig?: Record<string, any>;
  [key: string]: any;
}
