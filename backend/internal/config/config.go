package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port          int
	DBPath        string
	EncryptionKey string
	LogLevel      string
}

func Load() *Config {
	port, err := strconv.Atoi(getEnv("PORT", "8080"))
	if err != nil {
		port = 8080
	}

	return &Config{
		Port:          port,
		DBPath:        getEnv("DB_PATH", "./data/redora.db"),
		EncryptionKey: getEnv("ENCRYPTION_KEY", "default-32-byte-secret-key-change-me!!"),
		LogLevel:      getEnv("LOG_LEVEL", "info"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
