// Shim for @nanoclip/adapter-gemini-local and /ui sub-path
import type { CreateConfigValues, TranscriptEntry } from "../adapter-utils";

export const DEFAULT_GEMINI_LOCAL_MODEL = "gemini-2.5-pro";

export function buildGeminiLocalConfig(values: CreateConfigValues): Record<string, unknown> {
  return {
    model: values.model || DEFAULT_GEMINI_LOCAL_MODEL,
    instructionsFilePath: values.instructionsFilePath || undefined,
    promptTemplate: values.promptTemplate || undefined,
    bootstrapPrompt: values.bootstrapPrompt || undefined,
    extraArgs: values.extraArgs || undefined,
    envVars: values.envVars || undefined,
    maxTurnsPerRun: values.maxTurnsPerRun || undefined,
  };
}

export function parseGeminiStdoutLine(line: string, ts: string): TranscriptEntry[] {
  if (line.trim()) return [{ kind: "assistant", ts, text: line, delta: false }];
  return [];
}
