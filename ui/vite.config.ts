import path from "path";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
      lexical: path.resolve(__dirname, "./node_modules/lexical/Lexical.mjs"),
      "@nanoclip/shared": path.resolve(__dirname, "./src/lib/paperclip-shared.ts"),
      "@nanoclip/adapter-utils": path.resolve(__dirname, "./src/lib/adapter-utils.ts"),
      "@nanoclip/adapter-claude-local/ui": path.resolve(__dirname, "./src/lib/adapters/claude-local-ui.ts"),
      "@nanoclip/adapter-claude-local": path.resolve(__dirname, "./src/lib/adapters/claude-local-ui.ts"),
      "@nanoclip/adapter-codex-local/ui": path.resolve(__dirname, "./src/lib/adapters/codex-local-ui.ts"),
      "@nanoclip/adapter-codex-local": path.resolve(__dirname, "./src/lib/adapters/codex-local-ui.ts"),
      "@nanoclip/adapter-cursor-local/ui": path.resolve(__dirname, "./src/lib/adapters/cursor-local-ui.ts"),
      "@nanoclip/adapter-cursor-local": path.resolve(__dirname, "./src/lib/adapters/cursor-local-ui.ts"),
      "@nanoclip/adapter-gemini-local/ui": path.resolve(__dirname, "./src/lib/adapters/gemini-local-ui.ts"),
      "@nanoclip/adapter-gemini-local": path.resolve(__dirname, "./src/lib/adapters/gemini-local-ui.ts"),
      "@nanoclip/adapter-openclaw-gateway/ui": path.resolve(__dirname, "./src/lib/adapters/openclaw-gateway-ui.ts"),
      "@nanoclip/adapter-openclaw-gateway": path.resolve(__dirname, "./src/lib/adapters/openclaw-gateway-ui.ts"),
      "@nanoclip/adapter-opencode-local/ui": path.resolve(__dirname, "./src/lib/adapters/opencode-local-ui.ts"),
      "@nanoclip/adapter-opencode-local": path.resolve(__dirname, "./src/lib/adapters/opencode-local-ui.ts"),
      "@nanoclip/adapter-pi-local/ui": path.resolve(__dirname, "./src/lib/adapters/pi-local-ui.ts"),
      "@nanoclip/adapter-pi-local": path.resolve(__dirname, "./src/lib/adapters/pi-local-ui.ts"),
      "@nanoclip/adapter-openrouter-local/ui": path.resolve(__dirname, "./src/lib/adapters/openrouter-local-ui.ts"),
      "@nanoclip/adapter-openrouter-local": path.resolve(__dirname, "./src/lib/adapters/openrouter-local-ui.ts"),
    },
  },
  server: {
    host: "0.0.0.0",
    port: 5000,
    allowedHosts: true,
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        ws: true,
        changeOrigin: true,
      },
    },
  },
});
