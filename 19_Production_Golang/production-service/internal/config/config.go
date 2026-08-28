// Package config loads settings from environment variables, with sensible
// defaults — the guide's Configuration and Environment Variables sections,
// applied directly. No config file or flag layer here to keep this
// reference skeleton small; viper (see the guide's Libraries section)
// is the natural next step once a project needs the full precedence chain.
package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port            int
	LogLevel        string
	ShutdownTimeout int // seconds
}

func Load() Config {
	return Config{
		Port:            envInt("PORT", 8080),
		LogLevel:        envString("LOG_LEVEL", "info"),
		ShutdownTimeout: envInt("SHUTDOWN_TIMEOUT_SECONDS", 10),
	}
}

func envString(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
