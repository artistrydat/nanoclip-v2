// Shim for @nanoclip/adapter-cursor-local and /ui sub-path
import type { CreateConfigValues, TranscriptEntry } from "../adapter-utils";

export const DEFAULT_CURSOR_LOCAL_MODEL = "claude-4-5-sonnet";

export function buildCursorLocalConfig(values: CreateConfigValues): Record<string, unknown> {
  return {
    model: values.model || DEFAULT_CURSOR_LOCAL_MODEL,
    instructionsFilePath: values.instructionsFilePath || undefined,
    promptTemplate: values.promptTemplate || undefined,
    bootstrapPrompt: values.bootstrapPrompt || undefined,
    extraArgs: values.extraArgs || undefined,
    envVars: values.envVars || undefined,
    maxTurnsPerRun: values.maxTurnsPerRun || undefined,
  };
}

export function parseCursorStdoutLine(line: string, ts: string): TranscriptEntry[] {
  if (line.trim()) return [{ kind: "assistant", ts, text: line, delta: false }];
  return [];
}
