// Shim for @nanoclip/adapter-openclaw-gateway/ui
import type { CreateConfigValues, TranscriptEntry } from "../adapter-utils";

export function buildOpenClawGatewayConfig(values: CreateConfigValues): Record<string, unknown> {
  return {
    url: values.url || undefined,
    model: values.model || undefined,
    promptTemplate: values.promptTemplate || undefined,
    bootstrapPrompt: values.bootstrapPrompt || undefined,
    payloadTemplateJson: values.payloadTemplateJson || undefined,
    envVars: values.envVars || undefined,
    maxTurnsPerRun: values.maxTurnsPerRun || undefined,
  };
}

export function parseOpenClawGatewayStdoutLine(line: string, ts: string): TranscriptEntry[] {
  try {
    const obj = JSON.parse(line);
    if (obj.content) return [{ kind: "assistant", ts, text: String(obj.content), delta: false }];
  } catch { /* plain text */ }
  if (line.trim()) return [{ kind: "assistant", ts, text: line, delta: false }];
  return [];
}
