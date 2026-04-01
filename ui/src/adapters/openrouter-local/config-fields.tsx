import type { AdapterConfigFieldsProps } from "../types";
import { Field, DraftInput } from "../../components/agent-config-primitives";
import { ChoosePathButton } from "../../components/PathInstructionsModal";

const inputClass =
  "w-full rounded-md border border-border px-2.5 py-1.5 bg-transparent outline-none text-sm font-mono placeholder:text-muted-foreground/40";

const baseUrlHint =
  "Base URL for the OpenRouter API. Defaults to https://openrouter.ai/api/v1.";
const modelHint =
  "Model to use (e.g. openai/gpt-4o, anthropic/claude-3.5-sonnet, google/gemini-pro). See openrouter.ai/models for available options.";
const apiKeyHint =
  "Your OpenRouter API key. Get one at openrouter.ai/keys.";
const instructionsFileHint =
  "Absolute path to a markdown file (e.g. AGENTS.md) that defines this agent's behaviour. Injected into the system prompt at runtime.";

export function OpenRouterLocalConfigFields({
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
      <Field label="Base URL" hint={baseUrlHint}>
        <DraftInput
          value={
            isCreate
              ? v?.baseUrl ?? "https://openrouter.ai/api/v1"
              : eff("adapterConfig", "baseUrl", String(config.baseUrl ?? "https://openrouter.ai/api/v1"))
          }
          onCommit={(val) =>
            isCreate
              ? set!({ ...(v ?? {}), baseUrl: val || "https://openrouter.ai/api/v1" } as any)
              : mark("adapterConfig", "baseUrl", val || "https://openrouter.ai/api/v1")
          }
          immediate
          className={inputClass}
          placeholder="https://openrouter.ai/api/v1"
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
          placeholder="e.g. openai/gpt-4o, anthropic/claude-3.5-sonnet"
        />
      </Field>

      <Field label="API key" hint={apiKeyHint}>
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
          placeholder="sk-or-..."
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
