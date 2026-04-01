// Shim for @nanoclip/adapter-opencode-local/ui
import type { CreateConfigValues, TranscriptEntry } from "../adapter-utils";

export function buildOpenCodeLocalConfig(values: CreateConfigValues): Record<string, unknown> {
  return {
    model: values.model || undefined,
    instructionsFilePath: values.instructionsFilePath || undefined,
    promptTemplate: values.promptTemplate || undefined,
    bootstrapPrompt: values.bootstrapPrompt || undefined,
    extraArgs: values.extraArgs || undefined,
    envVars: values.envVars || undefined,
    maxTurnsPerRun: values.maxTurnsPerRun || undefined,
  };
}

export function parseOpenCodeStdoutLine(line: string, ts: string): TranscriptEntry[] {
  if (line.trim()) return [{ kind: "assistant", ts, text: line, delta: false }];
  return [];
}
