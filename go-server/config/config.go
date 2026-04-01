package config

import (
        "os"
        "strconv"
)

type Config struct {
        Port          string
        DSN           string
        JWTSecret     string
        DeploymentMode string
}

func Load() *Config {
        return &Config{
                Port:          getEnv("GO_PORT", "8080"),
                DSN:           buildDSN(),
                JWTSecret:     getEnv("JWT_SECRET", "paperclip-dev-secret-change-in-production"),
                DeploymentMode: getEnv("DEPLOYMENT_MODE", "local_trusted"),
        }
}

func buildDSN() string {
        // Explicit DSN takes priority
        if dsn := os.Getenv("MARIADB_DSN"); dsn != "" {
                return dsn
        }
        // SQLite fallback for dev — no MariaDB required
        if os.Getenv("MARIADB_HOST") == "" && os.Getenv("MARIADB_DB") == "" {
                return ""
        }
        // Assemble from individual vars
        host := getEnv("MARIADB_HOST", "127.0.0.1")
        port := getEnv("MARIADB_PORT", "3306")
        user := getEnv("MARIADB_USER", "paperclip")
        pass := getEnv("MARIADB_PASS", "paperclip")
        name := getEnv("MARIADB_DB", "paperclip")
        return user + ":" + pass + "@tcp(" + host + ":" + port + ")/" + name +
                "?charset=utf8mb4&parseTime=True&loc=UTC&multiStatements=true"
}

func getEnv(key, fallback string) string {
        if v := os.Getenv(key); v != "" {
                return v
        }
        return fallback
}

func GetInt(key string, fallback int) int {
        if v := os.Getenv(key); v != "" {
                if i, err := strconv.Atoi(v); err == nil {
                        return i
                }
        }
        return fallback
}
