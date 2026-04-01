import type { UIAdapterModule } from "../types";
import { parseOpenRouterStdoutLine } from "./parse-stdout";
import { OpenRouterLocalConfigFields } from "./config-fields";
import { buildOpenRouterLocalConfig } from "./build-config";

export const openRouterLocalUIAdapter: UIAdapterModule = {
  type: "openrouter_local",
  label: "OpenRouter",
  parseStdoutLine: parseOpenRouterStdoutLine,
  ConfigFields: OpenRouterLocalConfigFields,
  buildAdapterConfig: buildOpenRouterLocalConfig,
};
