package config

import (
	"os"
	"time"
)

const defaultCLIEndpoint = "http://127.0.0.1:17331"

type CLI struct {
	Endpoint string
	Token    string
	Timeout  time.Duration
}

func LoadCLIFromEnv() CLI {
	endpoint := getenvDefault("BOB_ENDPOINT", defaultCLIEndpoint)
	token := os.Getenv("BOB_TOKEN")
	timeout := 5 * time.Second

	if raw := os.Getenv("BOB_TIMEOUT"); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil {
			timeout = parsed
		}
	}

	return CLI{
		Endpoint: endpoint,
		Token:    token,
		Timeout:  timeout,
	}
}

func getenvDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
