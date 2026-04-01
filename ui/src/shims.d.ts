declare module "@nanoclip/adapter-utils" {
  export interface TranscriptEntry {
    kind: string;
    text?: string;
    ts?: string;
    delta?: boolean;
    role?: string;
    toolName?: string;
    toolInput?: any;
    toolResult?: any;
    subtype?: string;
    model?: string;
    name?: string;
    input?: any;
    toolUseId?: string;
    content?: any;
    isError?: boolean;
    [key: string]: any;
  }

  export type StdoutLineParser = (line: string, ts: string) => TranscriptEntry[];

  export interface CreateConfigValues {
    name?: string;
    instructionsFile?: string;
    model?: string;
    workspacePath?: string;
    [key: string]: any;
  }

  export function redactHomePathUserSegments(path: string, opts?: unknown): string;
  export function redactHomePathUserSegmentsInValue(value: unknown, opts?: unknown): unknown;
  export function redactTranscriptEntryPaths(entry: TranscriptEntry, opts?: unknown): TranscriptEntry;
}

declare module "@nanoclip/adapter-codex-local" {
  export const DEFAULT_CODEX_LOCAL_MODEL: string;
  export const DEFAULT_CODEX_LOCAL_BYPASS_APPROVALS_AND_SANDBOX: boolean;
}

declare module "@nanoclip/adapter-cursor-local" {
  export const DEFAULT_CURSOR_LOCAL_MODEL: string;
}

declare module "@nanoclip/adapter-gemini-local" {
  export const DEFAULT_GEMINI_LOCAL_MODEL: string;
}

declare module "@nanoclip/adapter-claude-local/ui" {
  import type { TranscriptEntry } from "@nanoclip/adapter-utils";
  export function parseClaudeStdoutLine(line: string, ts: string): TranscriptEntry[];
  export function buildClaudeLocalConfig(values: Record<string, any>): Record<string, any>;
}

declare module "@nanoclip/adapter-codex-local/ui" {
  import type { TranscriptEntry } from "@nanoclip/adapter-utils";
  export function parseCodexStdoutLine(line: string, ts: string): TranscriptEntry[];
  export function buildCodexLocalConfig(values: Record<string, any>): Record<string, any>;
}

declare module "@nanoclip/adapter-cursor-local/ui" {
  import type { TranscriptEntry } from "@nanoclip/adapter-utils";
  export function parseCursorStdoutLine(line: string, ts: string): TranscriptEntry[];
  export function buildCursorLocalConfig(values: Record<string, any>): Record<string, any>;
}

declare module "@nanoclip/adapter-gemini-local/ui" {
  import type { TranscriptEntry } from "@nanoclip/adapter-utils";
  export function parseGeminiStdoutLine(line: string, ts: string): TranscriptEntry[];
  export function buildGeminiLocalConfig(values: Record<string, any>): Record<string, any>;
}

declare module "@nanoclip/adapter-openclaw-gateway/ui" {
  import type { TranscriptEntry } from "@nanoclip/adapter-utils";
  export function parseOpenClawGatewayStdoutLine(line: string, ts: string): TranscriptEntry[];
  export function buildOpenClawGatewayConfig(values: Record<string, any>): Record<string, any>;
}

declare module "@nanoclip/adapter-opencode-local/ui" {
  import type { TranscriptEntry } from "@nanoclip/adapter-utils";
  export function parseOpenCodeStdoutLine(line: string, ts: string): TranscriptEntry[];
  export function buildOpenCodeLocalConfig(values: Record<string, any>): Record<string, any>;
}

declare module "@nanoclip/adapter-pi-local/ui" {
  import type { TranscriptEntry } from "@nanoclip/adapter-utils";
  export function parsePiStdoutLine(line: string, ts: string): TranscriptEntry[];
  export function buildPiLocalConfig(values: Record<string, any>): Record<string, any>;
}

declare module "hermes-paperclip-adapter/ui" {
  import type { TranscriptEntry } from "@nanoclip/adapter-utils";
  export function parseHermesStdoutLine(line: string, ts: string): TranscriptEntry[];
  export function buildHermesConfig(values: Record<string, any>): Record<string, any>;
}
