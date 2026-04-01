package handlers

import (
        "context"
        "encoding/json"
        "fmt"
        "io"
        "net/http"
        "strings"
        "time"

        "github.com/gin-gonic/gin"
        "paperclip-go/ws"
        "gorm.io/gorm"
)

// AdapterRoutes registers adapter-scoped routes under /companies/:companyId/adapters/:type
func AdapterRoutes(rg *gin.RouterGroup, _ *gorm.DB) {
        rg.GET("/:adapterType/models", listAdapterModels())
        rg.GET("/:adapterType/detect-model", detectAdapterModel())
        rg.POST("/:adapterType/test-environment", testAdapterEnvironment())
}

// CompanyEventsRoute registers GET /companies/:companyId/events/ws
func CompanyEventsRoute(rg *gin.RouterGroup, hub *ws.Hub) {
        rg.GET("/ws", ws.ServeWs(hub))
}

type adapterModel struct {
        ID    string `json:"id"`
        Label string `json:"label"`
}

// ollamaLocalModels lists popular local Ollama models (fallback when server unreachable).
var ollamaLocalModels = []adapterModel{
        {ID: "llama3.2", Label: "Llama 3.2 (3B)"},
        {ID: "llama3.2:1b", Label: "Llama 3.2 (1B)"},
        {ID: "llama3.1", Label: "Llama 3.1 (8B)"},
        {ID: "llama3.1:70b", Label: "Llama 3.1 (70B)"},
        {ID: "mistral", Label: "Mistral 7B"},
        {ID: "mistral-nemo", Label: "Mistral Nemo"},
        {ID: "qwen2.5-coder:7b", Label: "Qwen 2.5 Coder (7B)"},
        {ID: "qwen2.5-coder:14b", Label: "Qwen 2.5 Coder (14B)"},
        {ID: "qwen2.5-coder:32b", Label: "Qwen 2.5 Coder (32B)"},
        {ID: "deepseek-r1:7b", Label: "DeepSeek R1 (7B)"},
        {ID: "deepseek-r1:14b", Label: "DeepSeek R1 (14B)"},
        {ID: "deepseek-r1:32b", Label: "DeepSeek R1 (32B)"},
        {ID: "phi4", Label: "Phi-4"},
        {ID: "gemma3:4b", Label: "Gemma 3 (4B)"},
        {ID: "gemma3:12b", Label: "Gemma 3 (12B)"},
        {ID: "codellama", Label: "Code Llama"},
        {ID: "command-r", Label: "Command R"},
}

// ollamaCloudModels lists known Ollama Cloud models (base URL = https://ollama.com).
var ollamaCloudModels = []adapterModel{
        {ID: "gpt-oss:120b", Label: "GPT-OSS 120B (Cloud)"},
        {ID: "qwen3.5:122b", Label: "Qwen 3.5 (122B Cloud)"},
        {ID: "qwen3-coder-next", Label: "Qwen3 Coder Next (Cloud)"},
        {ID: "minimax-m2.7", Label: "MiniMax M2.7 (Cloud)"},
        {ID: "minimax-m2.5", Label: "MiniMax M2.5 (Cloud)"},
        {ID: "nemotron-3-super:120b", Label: "Nemotron 3 Super 120B (Cloud)"},
        {ID: "glm-5", Label: "GLM-5 744B (Cloud)"},
        {ID: "kimi-k2.5", Label: "Kimi K2.5 (Cloud)"},
        {ID: "devstral-small-2:24b", Label: "Devstral Small 2 (24B Cloud)"},
        {ID: "qwen3-next:80b", Label: "Qwen3 Next (80B Cloud)"},
        {ID: "ministral-3:8b", Label: "Ministral 3 (8B Cloud)"},
        {ID: "ministral-3:14b", Label: "Ministral 3 (14B Cloud)"},
}

var knownModels = map[string][]adapterModel{
        "claude_local": {
                {ID: "claude-opus-4-5", Label: "Claude Opus 4.5"},
                {ID: "claude-sonnet-4-5", Label: "Claude Sonnet 4.5"},
                {ID: "claude-3-7-sonnet-20250219", Label: "Claude Sonnet 3.7"},
                {ID: "claude-3-5-sonnet-20241022", Label: "Claude Sonnet 3.5"},
                {ID: "claude-3-5-haiku-20241022", Label: "Claude Haiku 3.5"},
        },
        "codex_local": {
                {ID: "o4-mini", Label: "o4-mini"},
                {ID: "o3", Label: "o3"},
                {ID: "gpt-4.1", Label: "GPT-4.1"},
                {ID: "gpt-4o", Label: "GPT-4o"},
                {ID: "gpt-4o-mini", Label: "GPT-4o Mini"},
        },
        "gemini_local": {
                {ID: "gemini-2.5-pro", Label: "Gemini 2.5 Pro"},
                {ID: "gemini-2.5-flash", Label: "Gemini 2.5 Flash"},
                {ID: "gemini-2.0-flash", Label: "Gemini 2.0 Flash"},
                {ID: "gemini-1.5-pro", Label: "Gemini 1.5 Pro"},
        },
        "hermes_local": {
                {ID: "hermes-3", Label: "Hermes 3"},
                {ID: "hermes-2-pro", Label: "Hermes 2 Pro"},
        },
        "opencode_local": {
                {ID: "auto", Label: "Auto (provider default)"},
        },
        "pi_local": {
                {ID: "pi", Label: "Pi"},
        },
}

// isOllamaCloud reports whether baseURL looks like the Ollama cloud endpoint.
func isOllamaCloud(baseURL string) bool {
        return strings.Contains(baseURL, "ollama.com")
}

func listAdapterModels() gin.HandlerFunc {
        return func(c *gin.Context) {
                adapterType := c.Param("adapterType")

                if adapterType == "ollama_local" {
                        baseURL := c.Query("baseUrl")
                        apiKey := c.Query("apiKey")
                        if baseURL == "" {
                                baseURL = "http://localhost:11434"
                        }
                        if live := fetchOllamaModels(baseURL, apiKey); live != nil {
                                c.JSON(http.StatusOK, live)
                                return
                        }
                        if isOllamaCloud(baseURL) {
                                c.JSON(http.StatusOK, ollamaCloudModels)
                        } else {
                                c.JSON(http.StatusOK, ollamaLocalModels)
                        }
                        return
                }

                models, ok := knownModels[adapterType]
                if !ok {
                        models = []adapterModel{}
                }
                c.JSON(http.StatusOK, models)
        }
}

// fetchOllamaModels calls /api/tags on the given Ollama instance.
func fetchOllamaModels(baseURL, apiKey string) []adapterModel {
        baseURL = strings.TrimRight(baseURL, "/")
        ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
        defer cancel()

        req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/tags", nil)
        if err != nil {
                return nil
        }
        if apiKey != "" {
                req.Header.Set("Authorization", "Bearer "+apiKey)
        }
        resp, err := http.DefaultClient.Do(req)
        if err != nil || resp.StatusCode != http.StatusOK {
                return nil
        }
        defer resp.Body.Close()

        var payload struct {
                Models []struct {
                        Name string `json:"name"`
                } `json:"models"`
        }
        if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil || len(payload.Models) == 0 {
                return nil
        }
        out := make([]adapterModel, 0, len(payload.Models))
        for _, m := range payload.Models {
                out = append(out, adapterModel{ID: m.Name, Label: m.Name})
        }
        return out
}

func detectAdapterModel() gin.HandlerFunc {
        return func(c *gin.Context) {
                c.JSON(http.StatusOK, nil)
        }
}

// extractEnvKey pulls a plain-text env var value from an adapterConfig env map.
// Supports both { "type": "plain", "value": "..." } and bare string values.
func extractEnvKey(adapterConfig map[string]interface{}, envVarName string) string {
        envRaw, ok := adapterConfig["env"]
        if !ok {
                return ""
        }
        envMap, ok := envRaw.(map[string]interface{})
        if !ok {
                return ""
        }
        entry, ok := envMap[envVarName]
        if !ok {
                return ""
        }
        // { "type": "plain", "value": "..." }
        if m, ok := entry.(map[string]interface{}); ok {
                if v, ok := m["value"].(string); ok {
                        return v
                }
        }
        // bare string
        if s, ok := entry.(string); ok {
                return s
        }
        return ""
}

// cloudTestSpec describes how to test a cloud API endpoint.
type cloudTestSpec struct {
        name       string // adapter display name
        url        string // endpoint to GET
        authHeader func(key string) (string, string) // returns header name + value
        envVarName string // env var holding the API key
        keyHint    string // where to get the key
}

var cloudSpecs = map[string]cloudTestSpec{
        "openrouter_local": {
                name: "OpenRouter",
                url:  "https://openrouter.ai/api/v1/models",
                authHeader: func(key string) (string, string) {
                        return "Authorization", "Bearer " + key
                },
                envVarName: "OPENROUTER_API_KEY",
                keyHint:    "openrouter.ai/keys",
        },
        "claude_local": {
                name: "Claude (Anthropic)",
                url:  "https://api.anthropic.com/v1/models",
                authHeader: func(key string) (string, string) {
                        return "x-api-key", key
                },
                envVarName: "ANTHROPIC_API_KEY",
                keyHint:    "console.anthropic.com → API Keys",
        },
        "codex_local": {
                name: "Codex (OpenAI)",
                url:  "https://api.openai.com/v1/models",
                authHeader: func(key string) (string, string) {
                        return "Authorization", "Bearer " + key
                },
                envVarName: "OPENAI_API_KEY",
                keyHint:    "platform.openai.com → API Keys",
        },
        "gemini_local": {
                name:       "Gemini (Google)",
                url:        "https://generativelanguage.googleapis.com/v1beta/models",
                envVarName: "GEMINI_API_KEY",
                authHeader: func(key string) (string, string) {
                        return "x-goog-api-key", key
                },
                keyHint: "aistudio.google.com → Get API key",
        },
        "opencode_local": {
                name: "OpenCode (OpenAI-compatible)",
                url:  "https://api.openai.com/v1/models",
                authHeader: func(key string) (string, string) {
                        return "Authorization", "Bearer " + key
                },
                envVarName: "OPENAI_API_KEY",
                keyHint:    "platform.openai.com → API Keys",
        },
        "hermes_local": {
                name:       "Hermes",
                url:        "https://hermes.nous.sh/v1/models",
                envVarName: "HERMES_API_KEY",
                authHeader: func(key string) (string, string) {
                        return "Authorization", "Bearer " + key
                },
                keyHint: "hermes.nous.sh → API Keys",
        },
        "pi_local": {
                name:       "Pi (Inflection)",
                url:        "https://api.inflection.ai/v1/models",
                envVarName: "PI_API_KEY",
                authHeader: func(key string) (string, string) {
                        return "Authorization", "Bearer " + key
                },
                keyHint: "developers.inflection.ai → API Keys",
        },
        "cursor": {
                name:       "Cursor",
                url:        "https://api.cursor.sh/v1/models",
                envVarName: "CURSOR_API_KEY",
                authHeader: func(key string) (string, string) {
                        return "Authorization", "Bearer " + key
                },
                keyHint: "cursor.sh/settings → API Keys",
        },
}

func testAdapterEnvironment() gin.HandlerFunc {
        return func(c *gin.Context) {
                adapterType := c.Param("adapterType")

                // Ollama: HTTP probe (local or cloud)
                if adapterType == "ollama_local" {
                        testOllamaEnvironment(c)
                        return
                }

                // All other adapters: cloud API test
                spec, known := cloudSpecs[adapterType]
                if !known {
                        c.JSON(http.StatusOK, gin.H{
                                "ok":      false,
                                "message": "Unknown adapter type: " + adapterType,
                                "status":  "error",
                                "checks":  []interface{}{},
                        })
                        return
                }
                testCloudAPI(c, spec)
        }
}

// testCloudAPI tests a cloud API endpoint using the API key from adapterConfig.
func testCloudAPI(c *gin.Context, spec cloudTestSpec) {
        var body struct {
                AdapterConfig map[string]interface{} `json:"adapterConfig"`
        }
        _ = c.ShouldBindJSON(&body)

        apiKey := extractEnvKey(body.AdapterConfig, spec.envVarName)

        if apiKey == "" {
                c.JSON(http.StatusOK, gin.H{
                        "ok":       false,
                        "message":  "No API key provided",
                        "status":   "fail",
                        "testedAt": time.Now().UTC().Format(time.RFC3339),
                        "checks": []gin.H{
                                {
                                        "name":    "API key",
                                        "ok":      false,
                                        "message": spec.envVarName + " is not set",
                                        "hint":    "Enter your API key in the field above. Get one at: " + spec.keyHint,
                                },
                        },
                })
                return
        }

        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

        req, err := http.NewRequestWithContext(ctx, http.MethodGet, spec.url, nil)
        if err != nil {
                c.JSON(http.StatusOK, gin.H{
                        "ok": false, "message": "Failed to build request: " + err.Error(),
                        "status": "fail", "testedAt": time.Now().UTC().Format(time.RFC3339),
                        "checks": []interface{}{},
                })
                return
        }

        headerName, headerValue := spec.authHeader(apiKey)
        req.Header.Set(headerName, headerValue)
        req.Header.Set("Content-Type", "application/json")
        // Anthropic requires this header
        if spec.envVarName == "ANTHROPIC_API_KEY" {
                req.Header.Set("anthropic-version", "2023-06-01")
        }

        resp, err := http.DefaultClient.Do(req)
        ts := time.Now().UTC().Format(time.RFC3339)
        if err != nil {
                c.JSON(http.StatusOK, gin.H{
                        "ok":       false,
                        "message":  fmt.Sprintf("Cannot reach %s API: %s", spec.name, err.Error()),
                        "status":   "fail",
                        "testedAt": ts,
                        "checks": []gin.H{
                                {
                                        "name":    "Cloud API reachable",
                                        "ok":      false,
                                        "message": err.Error(),
                                        "hint":    "Check your network connection.",
                                },
                        },
                })
                return
        }
        defer resp.Body.Close()

        if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
                body2, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
                c.JSON(http.StatusOK, gin.H{
                        "ok":       false,
                        "message":  fmt.Sprintf("Authentication failed (%d) for %s", resp.StatusCode, spec.name),
                        "status":   "fail",
                        "testedAt": ts,
                        "checks": []gin.H{
                                {
                                        "name":    "API key valid",
                                        "ok":      false,
                                        "message": strings.TrimSpace(string(body2)),
                                        "hint":    "Your API key appears invalid. Get a valid key at: " + spec.keyHint,
                                },
                        },
                })
                return
        }

        if resp.StatusCode >= 400 {
                body2, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
                c.JSON(http.StatusOK, gin.H{
                        "ok":       false,
                        "message":  fmt.Sprintf("%s API returned HTTP %d", spec.name, resp.StatusCode),
                        "status":   "fail",
                        "testedAt": ts,
                        "checks": []gin.H{
                                {
                                        "name":    "API response",
                                        "ok":      false,
                                        "message": strings.TrimSpace(string(body2)),
                                        "hint":    "Unexpected error from " + spec.name + " API.",
                                },
                        },
                })
                return
        }

        c.JSON(http.StatusOK, gin.H{
                "ok":       true,
                "message":  fmt.Sprintf("Connected to %s API successfully", spec.name),
                "status":   "pass",
                "testedAt": ts,
                "checks": []gin.H{
                        {
                                "name":    "Cloud API connection",
                                "ok":      true,
                                "message": fmt.Sprintf("%s API key is valid and reachable", spec.name),
                        },
                },
        })
}

// testOllamaEnvironment probes the Ollama HTTP API.
func testOllamaEnvironment(c *gin.Context) {
        var body struct {
                AdapterConfig map[string]interface{} `json:"adapterConfig"`
        }
        _ = c.ShouldBindJSON(&body)

        baseURL := "http://localhost:11434"
        if v, ok := body.AdapterConfig["baseUrl"].(string); ok && v != "" {
                baseURL = strings.TrimRight(v, "/")
        }
        apiKey := ""
        if v, ok := body.AdapterConfig["apiKey"].(string); ok {
                apiKey = v
        }

        cloud := isOllamaCloud(baseURL)
        probeURL := baseURL + "/api/version"

        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

        req, err := http.NewRequestWithContext(ctx, http.MethodGet, probeURL, nil)
        if err != nil {
                c.JSON(http.StatusOK, gin.H{
                        "ok": false, "message": "Failed to build request: " + err.Error(),
                        "status": "error", "checks": []interface{}{},
                })
                return
        }
        if apiKey != "" {
                req.Header.Set("Authorization", "Bearer "+apiKey)
        }

        isLocalhost := strings.HasPrefix(baseURL, "http://localhost") ||
                strings.HasPrefix(baseURL, "http://127.0.0.1")
        isPrivateIP := strings.HasPrefix(baseURL, "http://10.") ||
                strings.HasPrefix(baseURL, "http://192.168.") ||
                strings.HasPrefix(baseURL, "http://172.")

        if isLocalhost {
                c.JSON(http.StatusOK, gin.H{
                        "ok":       false,
                        "message":  "Cannot test localhost Ollama from the cloud server",
                        "status":   "fail",
                        "testedAt": time.Now().UTC().Format(time.RFC3339),
                        "checks": []gin.H{
                                {
                                        "name":    "Ollama reachable",
                                        "ok":      false,
                                        "message": "localhost is this cloud server, not your device",
                                        "hint":    "Expose your local Ollama with a public tunnel (e.g. ngrok http 11434) and paste the tunnel URL here, or use https://ollama.com with an API key.",
                                },
                        },
                })
                return
        }

        resp, err := http.DefaultClient.Do(req)
        ts := time.Now().UTC().Format(time.RFC3339)
        if err != nil {
                hint := "Start Ollama with `ollama serve`, or set the correct base URL."
                if cloud {
                        hint = "Check your API key at ollama.com/settings/keys, or verify the URL is https://ollama.com."
                } else if isPrivateIP {
                        hint = "This is a private network IP — not reachable from this cloud environment. Use a public tunnel (e.g. ngrok http 11434) or switch to https://ollama.com with an API key."
                }
                c.JSON(http.StatusOK, gin.H{
                        "ok":       false,
                        "message":  fmt.Sprintf("Cannot reach Ollama at %s", baseURL),
                        "status":   "fail",
                        "testedAt": ts,
                        "checks": []gin.H{
                                {"name": "Ollama reachable", "ok": false, "message": err.Error(), "hint": hint},
                        },
                })
                return
        }
        defer resp.Body.Close()

        if resp.StatusCode == http.StatusUnauthorized {
                c.JSON(http.StatusOK, gin.H{
                        "ok": false, "message": "Authentication failed (401 Unauthorized)", "status": "fail",
                        "testedAt": ts,
                        "checks": []gin.H{
                                {
                                        "name": "Authentication", "ok": false,
                                        "message": "Invalid or missing API key",
                                        "hint":    "Generate an API key at ollama.com/settings/keys and paste it in the API key field.",
                                },
                        },
                })
                return
        }

        var versionPayload struct {
                Version string `json:"version"`
        }
        _ = json.NewDecoder(resp.Body).Decode(&versionPayload)

        label := "Ollama"
        if cloud {
                label = "Ollama Cloud"
        }
        msg := fmt.Sprintf("%s is reachable at %s", label, baseURL)
        if versionPayload.Version != "" {
                msg = fmt.Sprintf("%s %s is reachable at %s", label, versionPayload.Version, baseURL)
        }

        c.JSON(http.StatusOK, gin.H{
                "ok": true, "message": msg, "status": "pass", "testedAt": ts,
                "checks": []gin.H{{"name": "Ollama reachable", "ok": true, "message": msg}},
        })
}
