package config

import (
	"fmt"
	"os"
	"time"
)

const defaultCLIEndpoint = "http://127.0.0.1:17331"

const defaultCLITimeout = 5 * time.Second

const cliConfigFilename = "bob.json"

type CLI struct {
	Endpoint string
	Token    string
	Session  string
	Timeout  time.Duration
}

type cliFileConfig struct {
	Endpoint string `json:"endpoint"`
	Token    string `json:"token"`
	Session  string `json:"session"`
	Timeout  string `json:"timeout"`
}

func LoadCLI() (CLI, error) {
	fileCfg, _, err := loadJSON[cliFileConfig](cliConfigFilename)
	if err != nil {
		return CLI{}, err
	}

	cfg := CLI{
		Endpoint: defaultCLIEndpoint,
		Token:    fileCfg.Token,
		Session:  fileCfg.Session,
		Timeout:  defaultCLITimeout,
	}
	if fileCfg.Endpoint != "" {
		cfg.Endpoint = fileCfg.Endpoint
	}
	if fileCfg.Timeout != "" {
		parsed, err := time.ParseDuration(fileCfg.Timeout)
		if err != nil {
			return CLI{}, fmt.Errorf("BOB_TIMEOUT in config file must be a duration: %w", err)
		}
		cfg.Timeout = parsed
	}

	if value := os.Getenv("BOB_ENDPOINT"); value != "" {
		cfg.Endpoint = value
	}
	if value := os.Getenv("BOB_TOKEN"); value != "" {
		cfg.Token = value
	}
	if value := os.Getenv("BOB_SESSION"); value != "" {
		cfg.Session = value
	}

	if raw := os.Getenv("BOB_TIMEOUT"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return CLI{}, fmt.Errorf("BOB_TIMEOUT must be a duration: %w", err)
		}
		cfg.Timeout = parsed
	}

	return cfg, nil
}

func WriteCLIConfig(cfg CLI, force bool) (string, error) {
	if cfg.Token == "" {
		return "", fmt.Errorf("token is required")
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = defaultCLIEndpoint
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = defaultCLITimeout
	}
	path, err := writeJSON(cliConfigFilename, cliFileConfig{
		Endpoint: cfg.Endpoint,
		Token:    cfg.Token,
		Session:  cfg.Session,
		Timeout:  cfg.Timeout.String(),
	}, force)
	if err != nil {
		return "", fmt.Errorf("write CLI config: %w", err)
	}
	return path, nil
}

func getenvDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
