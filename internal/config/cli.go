package config

import (
	"fmt"
	"os"
	"time"
)

const defaultCLIEndpoint = "http://127.0.0.1:17331"

const defaultCLITimeout = 5 * time.Second

const DefaultCodeServerPort = 8080

const cliConfigFilename = "bob.json"

type CLI struct {
	Endpoint   string
	Token      string
	Session    string
	Timeout    time.Duration
	CodeServer CodeServer
}

type CodeServer struct {
	Port int
}

type cliFileConfig struct {
	Endpoint   string              `json:"endpoint"`
	Token      string              `json:"token"`
	Session    string              `json:"session"`
	Timeout    string              `json:"timeout"`
	CodeServer cliCodeServerConfig `json:"codeServer,omitempty"`
}

type cliCodeServerConfig struct {
	Port *int `json:"port,omitempty"`
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
		CodeServer: CodeServer{
			Port: DefaultCodeServerPort,
		},
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
	if fileCfg.CodeServer.Port != nil {
		cfg.CodeServer.Port = *fileCfg.CodeServer.Port
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

func ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return nil
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
	fileCfg := cliFileConfig{
		Endpoint: cfg.Endpoint,
		Token:    cfg.Token,
		Session:  cfg.Session,
		Timeout:  cfg.Timeout.String(),
	}
	if cfg.CodeServer.Port != 0 {
		if err := ValidatePort(cfg.CodeServer.Port); err != nil {
			return "", fmt.Errorf("codeServer.port is invalid: %w", err)
		}
		fileCfg.CodeServer.Port = &cfg.CodeServer.Port
	}
	path, err := writeJSON(cliConfigFilename, fileCfg, force)
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
