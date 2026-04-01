import type { UIAdapterModule } from "./types";
import { ollamaLocalUIAdapter } from "./ollama-local";
import { openRouterLocalUIAdapter } from "./openrouter-local";

const uiAdapters: UIAdapterModule[] = [
  ollamaLocalUIAdapter,
  openRouterLocalUIAdapter,
];

const adaptersByType = new Map<string, UIAdapterModule>(
  uiAdapters.map((a) => [a.type, a]),
);

export function getUIAdapter(type: string): UIAdapterModule {
  return adaptersByType.get(type) ?? ollamaLocalUIAdapter;
}

export function listUIAdapters(): UIAdapterModule[] {
  return [...uiAdapters];
}
