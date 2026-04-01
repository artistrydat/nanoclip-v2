// Shim for @nanoclip/adapter-pi-local/ui
import type { CreateConfigValues, TranscriptEntry } from "../adapter-utils";

export function buildPiLocalConfig(values: CreateConfigValues): Record<string, unknown> {
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

export function parsePiStdoutLine(line: string, ts: string): TranscriptEntry[] {
  if (line.trim()) return [{ kind: "assistant", ts, text: line, delta: false }];
  return [];
}
