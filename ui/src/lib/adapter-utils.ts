// Shim replacing @nanoclip/adapter-utils

export interface CreateConfigValues {
  adapterType: string;
  cwd: string;
  instructionsFilePath: string;
  promptTemplate: string;
  model: string;
  thinkingEffort: string;
  chrome: boolean;
  dangerouslySkipPermissions: boolean;
  search: boolean;
  dangerouslyBypassSandbox: boolean;
  command: string;
  args: string;
  extraArgs: string;
  envVars: string;
  envBindings: Record<string, string>;
  url: string;
  bootstrapPrompt: string;
  payloadTemplateJson: string;
  workspaceStrategyType: string;
  workspaceBaseRef: string;
  workspaceBranchTemplate: string;
  worktreeParentDir: string;
  runtimeServicesJson: string;
  maxTurnsPerRun: number;
  heartbeatEnabled: boolean;
  intervalSec: number;
}

export interface TranscriptEntry {
  kind: string;
  ts: string;
  text?: string;
  delta?: boolean;
  thinking?: string;
  toolName?: string;
  toolInput?: unknown;
  toolResult?: unknown;
  error?: string;
}

export type StdoutLineParser = (line: string, ts: string) => TranscriptEntry[];

interface RedactOptions {
  enabled: boolean;
}

/** Redact the home path username segments from a string. */
export function redactHomePathUserSegments(
  text: string,
  opts: RedactOptions,
): string {
  if (!opts.enabled) return text;
  return text.replace(/\/home\/[^/\s]+/g, "/home/[user]").replace(/\/Users\/[^/\s]+/g, "/Users/[user]");
}

/** Redact username from a value (string or object). */
export function redactHomePathUserSegmentsInValue(
  value: unknown,
  opts: RedactOptions,
): unknown {
  if (!opts.enabled) return value;
  if (typeof value === "string") return redactHomePathUserSegments(value, opts);
  return value;
}

/** Redact username segments from a transcript entry's text fields. */
export function redactTranscriptEntryPaths(
  entry: TranscriptEntry,
  opts: RedactOptions,
): TranscriptEntry {
  if (!opts.enabled) return entry;
  return {
    ...entry,
    text: entry.text != null ? redactHomePathUserSegments(entry.text, opts) : entry.text,
    thinking: entry.thinking != null ? redactHomePathUserSegments(entry.thinking, opts) : entry.thinking,
  };
}
