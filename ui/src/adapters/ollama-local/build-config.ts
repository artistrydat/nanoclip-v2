import type { CreateConfigValues } from "../../components/AgentConfigForm";

export function buildOllamaLocalConfig(v: CreateConfigValues): Record<string, unknown> {
  const ac: Record<string, unknown> = {};
  ac.baseUrl = (v as any).baseUrl || "http://localhost:11434";
  if (v.model) ac.model = v.model;
  if ((v as any).apiKey) ac.apiKey = (v as any).apiKey;
  if (v.instructionsFile) ac.instructionsFilePath = v.instructionsFile;
  if ((v as any).instructionsFilePath) ac.instructionsFilePath = (v as any).instructionsFilePath;
  ac.timeoutSec = 120;
  return ac;
}
