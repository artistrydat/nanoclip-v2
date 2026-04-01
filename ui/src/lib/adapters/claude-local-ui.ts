// Shim for @nanoclip/adapter-claude-local/ui
import type { CreateConfigValues, TranscriptEntry } from "../adapter-utils";

export function buildClaudeLocalConfig(values: CreateConfigValues): Record<string, unknown> {
  return {
    model: values.model || undefined,
    thinkingEffort: values.thinkingEffort || undefined,
    chrome: values.chrome || undefined,
    dangerouslySkipPermissions: values.dangerouslySkipPermissions || undefined,
    instructionsFilePath: values.instructionsFilePath || undefined,
    promptTemplate: values.promptTemplate || undefined,
    bootstrapPrompt: values.bootstrapPrompt || undefined,
    extraArgs: values.extraArgs || undefined,
    envVars: values.envVars || undefined,
    envBindings: Object.keys(values.envBindings ?? {}).length ? values.envBindings : undefined,
    localCommand: undefined,
    maxTurnsPerRun: values.maxTurnsPerRun || undefined,
  };
}

export function parseClaudeStdoutLine(line: string, ts: string): TranscriptEntry[] {
  try {
    const obj = JSON.parse(line);
    if (obj.type === "assistant" && obj.message?.content) {
      const parts: TranscriptEntry[] = [];
      for (const block of Array.isArray(obj.message.content) ? obj.message.content : [obj.message.content]) {
        if (block.type === "thinking" && block.thinking) {
          parts.push({ kind: "thinking", ts, text: block.thinking, delta: false });
        } else if (block.type === "text" && block.text) {
          parts.push({ kind: "assistant", ts, text: block.text, delta: false });
        } else if (block.type === "tool_use") {
          parts.push({ kind: "tool_call", ts, toolName: block.name, toolInput: block.input });
        }
      }
      return parts.length ? parts : [];
    }
    if (obj.type === "tool_result") {
      return [{ kind: "tool_result", ts, toolResult: obj.content }];
    }
    if (obj.type === "result" && obj.result) {
      return [{ kind: "result", ts, text: String(obj.result) }];
    }
  } catch {
    // Not JSON — treat as plain text
  }
  if (line.trim()) {
    return [{ kind: "assistant", ts, text: line, delta: false }];
  }
  return [];
}
