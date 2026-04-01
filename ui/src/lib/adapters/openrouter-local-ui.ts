// Shim for @nanoclip/adapter-openrouter-local and /ui sub-path
import type { CreateConfigValues, TranscriptEntry } from "../adapter-utils";

export const DEFAULT_OPENROUTER_MODEL = "openai/gpt-4o-mini";

export function buildOpenRouterLocalConfig(values: CreateConfigValues): Record<string, unknown> {
  return {
    model: values.model || DEFAULT_OPENROUTER_MODEL,
    instructionsFilePath: values.instructionsFilePath || undefined,
    promptTemplate: values.promptTemplate || undefined,
    bootstrapPrompt: values.bootstrapPrompt || undefined,
    extraArgs: values.extraArgs || undefined,
    envVars: values.envVars || undefined,
    envBindings: Object.keys(values.envBindings ?? {}).length ? values.envBindings : undefined,
    maxTurnsPerRun: values.maxTurnsPerRun || undefined,
  };
}

export function parseOpenRouterStdoutLine(line: string, ts: string): TranscriptEntry[] {
  try {
    const obj = JSON.parse(line);
    if (obj.type === "message" && obj.content) {
      const text = Array.isArray(obj.content)
        ? obj.content.filter((b: { type: string }) => b.type === "text").map((b: { text: string }) => b.text).join("")
        : String(obj.content);
      if (text) return [{ kind: "assistant", ts, text, delta: false }];
    }
    if (obj.choices?.[0]?.delta?.content) {
      return [{ kind: "assistant", ts, text: obj.choices[0].delta.content, delta: true }];
    }
  } catch {
    // plain text
  }
  if (line.trim()) return [{ kind: "assistant", ts, text: line, delta: false }];
  return [];
}
