// Shim for @nanoclip/adapter-codex-local and /ui sub-path
import type { CreateConfigValues, TranscriptEntry } from "../adapter-utils";

export const DEFAULT_CODEX_LOCAL_MODEL = "o4-mini";
export const DEFAULT_CODEX_LOCAL_BYPASS_APPROVALS_AND_SANDBOX = false;

export function buildCodexLocalConfig(values: CreateConfigValues): Record<string, unknown> {
  return {
    model: values.model || DEFAULT_CODEX_LOCAL_MODEL,
    dangerouslyBypassSandbox: values.dangerouslyBypassSandbox,
    search: values.search || undefined,
    instructionsFilePath: values.instructionsFilePath || undefined,
    promptTemplate: values.promptTemplate || undefined,
    bootstrapPrompt: values.bootstrapPrompt || undefined,
    extraArgs: values.extraArgs || undefined,
    envVars: values.envVars || undefined,
    envBindings: Object.keys(values.envBindings ?? {}).length ? values.envBindings : undefined,
    maxTurnsPerRun: values.maxTurnsPerRun || undefined,
  };
}

export function parseCodexStdoutLine(line: string, ts: string): TranscriptEntry[] {
  try {
    const obj = JSON.parse(line);
    if (obj.type === "message" && obj.content) {
      const text = Array.isArray(obj.content)
        ? obj.content.filter((b: { type: string }) => b.type === "output_text").map((b: { text: string }) => b.text).join("")
        : String(obj.content);
      if (text) return [{ kind: "assistant", ts, text, delta: false }];
    }
    if (obj.type === "function_call") {
      return [{ kind: "tool_call", ts, toolName: obj.name, toolInput: obj.arguments }];
    }
    if (obj.type === "function_call_output") {
      return [{ kind: "tool_result", ts, toolResult: obj.output }];
    }
  } catch {
    // plain text
  }
  if (line.trim()) return [{ kind: "assistant", ts, text: line, delta: false }];
  return [];
}
