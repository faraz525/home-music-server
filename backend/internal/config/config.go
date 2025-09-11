package config

import (
    "os"
)

type Config struct {
    Port     string
    DataDir  string
    JWTSecret string
    RefreshSecret string
    BaseURL  string
    Env      string
}

func FromEnv() *Config {
    cfg := &Config{}
    cfg.Port = getEnv("PORT", "8080")
    cfg.DataDir = getEnv("DATA_DIR", "/mnt/music/cratedrop")
    cfg.JWTSecret = getEnv("JWT_SECRET", "")
    cfg.RefreshSecret = getEnv("REFRESH_SECRET", "")
    cfg.BaseURL = getEnv("BASE_URL", "http://localhost")
    cfg.Env = getEnv("APP_ENV", "development")
    return cfg
}

func getEnv(key, def string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return def
}

