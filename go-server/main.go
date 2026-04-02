package main

import (
        "bytes"
        "fmt"
        "io/fs"
        "log"
        "net/http"
        "os"
        "path/filepath"
        "strings"
        "time"

        "github.com/gin-gonic/gin"
        "github.com/joho/godotenv"
        "paperclip-go/config"
        "paperclip-go/db"
        "paperclip-go/handlers"
        "paperclip-go/middleware"
        "paperclip-go/services"
        "paperclip-go/ws"
)

func main() {
        _ = godotenv.Load("../.env")
        _ = godotenv.Load(".env")

        cfg := config.Load()
        middleware.SetJWTSecret(cfg.JWTSecret)
        middleware.SetDeploymentMode(cfg.DeploymentMode)

        database := db.Connect(cfg.DSN)
        db.AutoMigrate(database)

        hub := ws.GlobalHub
        go hub.Run()

        hb := services.NewHeartbeatService(database, hub)
        go hb.Start()

        tg := services.NewTelegramService(database, hub)
        go tg.Start()

        if os.Getenv("GIN_MODE") == "" {
                gin.SetMode(gin.ReleaseMode)
        }
        router := gin.New()
        router.Use(gin.Logger(), gin.Recovery())
        router.Use(middleware.CORS())
        router.Use(middleware.Auth(database))

        // ── API Routes ───────────────────────────────────────────────────────────
        api := router.Group("/api")

        // Health
        api.GET("/health", handlers.HealthHandler(database))

        // Auth  /api/auth/*
        handlers.AuthRoutes(api.Group("/auth"), database)

        // WebSocket live events
        api.GET("/live-events", ws.ServeWs(hub))

        // ── Companies (no company scope) ─────────────────────────────────────────
        companiesGroup := api.Group("/companies")
        companiesGroup.Use(middleware.RequireAuth())
        handlers.CompanyRoutes(companiesGroup, database)

        // ── Per-company sub-routes ────────────────────────────────────────────────
        company := api.Group("/companies/:companyId")
        company.Use(middleware.RequireAuth())

        handlers.AgentRoutes(company.Group("/agents"), database)
        handlers.AgentHireRoutes(company.Group("/agent-hires"), database)
        handlers.IssueRoutes(company.Group("/issues"), database, hub, hb)
        handlers.LabelRoutes(company.Group("/labels"), database)
        handlers.ProjectRoutes(company.Group("/projects"), database)
        handlers.GoalRoutes(company.Group("/goals"), database)
        handlers.DashboardRoutes(company.Group("/dashboard"), database)
        handlers.SidebarRoutes(company.Group("/sidebar-badges"), database)
        handlers.ApprovalRoutes(company.Group("/approvals"), database, hub)
        handlers.CostRoutes(company.Group("/costs"), database)
        handlers.ActivityRoutes(company.Group("/activity"), database)
        handlers.RoutineRoutes(company.Group("/routines"), database)
        handlers.RunRoutes(company.Group("/runs"), database, hb)
        // /heartbeat-runs is the canonical frontend path for the same resource
        handlers.RunRoutes(company.Group("/heartbeat-runs"), database, hb)

        // Members + Org
        handlers.MemberRoutes(company.Group("/members"), database)
        handlers.OrgRoutes(company.Group("/org"), database)

        handlers.GlobalLiveRunRoutes(company.Group("/live-runs"), database)

        // Secrets
        handlers.CompanySecretRoutes(company.Group("/secrets"), database)
        // Skills
        handlers.SkillRoutes(company.Group("/skills"), database)
        // Execution workspaces (company-scoped)
        handlers.CompanyWorkspaceRoutes(company.Group("/execution-workspaces"), database)
        // Inbox
        handlers.InboxRoutes(company.Group("/inbox"), database)
        // Assets
        handlers.AssetRoutes(company.Group("/assets"), database)
        // Invites
        handlers.InviteRoutes(company.Group("/invites"), database)
        // Join requests (empty in local_trusted mode)
        handlers.JoinRequestRoutes(company.Group("/join-requests"), database)
        // Database manager
        handlers.DatabaseRoutes(company.Group("/database"), database)
        // Budget policies overview
        handlers.BudgetRoutes(company.Group("/budgets"), database)
        // Agent configurations (access control)
        handlers.AccessRoutes(company.Group("/agent-configurations"), database)
        // Adapter model listing + environment tests
        handlers.AdapterRoutes(company.Group("/adapters"), database)
        // Per-company live events WebSocket
        handlers.CompanyEventsRoute(company.Group("/events"), hub)

        // ── Global routes (cross-company by resource ID) ──────────────────────────
        global := api.Group("")
        global.Use(middleware.RequireAuth())

        // Global heartbeat run routes
        handlers.GlobalRunRoutes(global.Group("/heartbeat-runs"), database)
        // Run events submission
        handlers.HeartbeatRunEventRoutes(global.Group("/heartbeat-runs/:runId/events"), database)

        // Global execution workspaces
        handlers.GlobalWorkspaceRoutes(global.Group("/execution-workspaces"), database)

        // Global agent lookup (by agentId, companyId as query param)
        handlers.GlobalAgentRoutes(global.Group("/agents"), database)

        // Global issue sub-routes
        handlers.GlobalIssueRoutes(global.Group("/issues"), database, hub, hb)

        // Global secrets
        handlers.GlobalSecretRoutes(global.Group("/secrets"), database)

        // Global labels (cross-company delete)
        handlers.GlobalLabelRoutes(global.Group("/labels"), database)

        // Global goals (by ID, no company scope required)
        handlers.GlobalGoalRoutes(global.Group("/goals"), database)

        // Plugins
        handlers.PluginRoutes(api.Group("/plugins"), database)

        // Instance settings
        handlers.InstanceRoutes(api.Group("/instance"), database)

        // ── Static UI (built React app) ──────────────────────────────────────────
        var uiFS fs.FS
        if eFS := embeddedUI(); eFS != nil {
                log.Println("[server] serving UI from embedded binary")
                uiFS = eFS
        } else if dir := findUIDistDir(); dir != "" {
                log.Printf("[server] serving UI from disk: %s", dir)
                uiFS = os.DirFS(dir)
        }

        if uiFS != nil {
                assetsFS, _ := fs.Sub(uiFS, "assets")
                router.StaticFS("/assets", http.FS(assetsFS))
                for _, name := range []string{
                        "favicon.ico", "favicon.svg",
                        "favicon-32x32.png", "favicon-16x16.png",
                        "apple-touch-icon.png", "sw.js",
                } {
                        fname := name
                        router.GET("/"+fname, func(c *gin.Context) {
                                data, err := fs.ReadFile(uiFS, fname)
                                if err != nil {
                                        c.Status(http.StatusNotFound)
                                        return
                                }
                                http.ServeContent(c.Writer, c.Request, fname, time.Time{}, bytes.NewReader(data))
                        })
                }
                router.NoRoute(func(c *gin.Context) {
                        if strings.HasPrefix(c.Request.URL.Path, "/api") {
                                c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
                                return
                        }
                        data, err := fs.ReadFile(uiFS, "index.html")
                        if err != nil {
                                c.Status(http.StatusInternalServerError)
                                return
                        }
                        c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
                        c.Header("Pragma", "no-cache")
                        c.Header("Expires", "0")
                        c.Data(http.StatusOK, "text/html; charset=utf-8", data)
                })
        } else {
                log.Println("[server] no UI dist found — running in API-only mode")
                router.NoRoute(func(c *gin.Context) {
                        c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
                })
        }

        addr := fmt.Sprintf("0.0.0.0:%s", cfg.Port)
        log.Printf("[server] NanoClip listening on %s", addr)
        if err := router.Run(addr); err != nil {
                log.Fatalf("server error: %v", err)
        }
}

func findUIDistDir() string {
        candidates := []string{
                "../ui/dist",
                "./ui-dist",
                "./ui/dist",
        }
        for _, path := range candidates {
                if _, err := os.Stat(filepath.Join(path, "index.html")); err == nil {
                        abs, _ := filepath.Abs(path)
                        return abs
                }
        }
        return ""
}
