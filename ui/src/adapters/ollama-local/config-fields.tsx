import type { AdapterConfigFieldsProps } from "../types";
import {
  Field,
  DraftInput,
} from "../../components/agent-config-primitives";
import { ChoosePathButton } from "../../components/PathInstructionsModal";

const inputClass =
  "w-full rounded-md border border-border px-2.5 py-1.5 bg-transparent outline-none text-sm font-mono placeholder:text-muted-foreground/40";

const baseUrlHint =
  "URL of your Ollama instance. Use http://localhost:11434 for a local install, or a remote URL for cloud/hosted Ollama.";
const modelHint =
  "Model name to use (e.g. llama3.2, mistral, qwen2.5-coder:7b). Must already be pulled in your Ollama instance.";
const apiKeyHint =
  "API key for authenticated/cloud Ollama deployments. Leave blank for local instances.";
const instructionsFileHint =
  "Absolute path to a markdown file (e.g. AGENTS.md) that defines this agent's behaviour. Injected into the system prompt at runtime.";

export function OllamaLocalConfigFields({
  isCreate,
  values,
  set,
  config,
  eff,
  mark,
  hideInstructionsFile,
}: AdapterConfigFieldsProps) {
  const v = values as any;
  return (
    <>
      <Field label="Ollama base URL" hint={baseUrlHint}>
        <DraftInput
          value={
            isCreate
              ? v?.baseUrl ?? "http://localhost:11434"
              : eff("adapterConfig", "baseUrl", String(config.baseUrl ?? "http://localhost:11434"))
          }
          onCommit={(val) =>
            isCreate
              ? set!({ ...(v ?? {}), baseUrl: val || "http://localhost:11434" } as any)
              : mark("adapterConfig", "baseUrl", val || "http://localhost:11434")
          }
          immediate
          className={inputClass}
          placeholder="http://localhost:11434"
        />
      </Field>

      <Field label="Model" hint={modelHint}>
        <DraftInput
          value={
            isCreate
              ? v?.model ?? ""
              : eff("adapterConfig", "model", String(config.model ?? ""))
          }
          onCommit={(val) =>
            isCreate
              ? set!({ ...(v ?? {}), model: val } as any)
              : mark("adapterConfig", "model", val || undefined)
          }
          immediate
          className={inputClass}
          placeholder="e.g. llama3.2, mistral, qwen2.5-coder:7b"
        />
      </Field>

      <Field label="API key (optional)" hint={apiKeyHint}>
        <DraftInput
          value={
            isCreate
              ? v?.apiKey ?? ""
              : eff("adapterConfig", "apiKey", String(config.apiKey ?? ""))
          }
          onCommit={(val) =>
            isCreate
              ? set!({ ...(v ?? {}), apiKey: val } as any)
              : mark("adapterConfig", "apiKey", val || undefined)
          }
          immediate
          className={inputClass}
          placeholder="sk-... (leave blank for local)"
        />
      </Field>

      {!hideInstructionsFile && (
        <Field label="Agent instructions file" hint={instructionsFileHint}>
          <div className="flex items-center gap-2">
            <DraftInput
              value={
                isCreate
                  ? v?.instructionsFilePath ?? ""
                  : eff("adapterConfig", "instructionsFilePath", String(config.instructionsFilePath ?? ""))
              }
              onCommit={(val) =>
                isCreate
                  ? set!({ ...(v ?? {}), instructionsFilePath: val } as any)
                  : mark("adapterConfig", "instructionsFilePath", val || undefined)
              }
              immediate
              className={inputClass}
              placeholder="/absolute/path/to/AGENTS.md"
            />
            <ChoosePathButton />
          </div>
        </Field>
      )}
    </>
  );
}
