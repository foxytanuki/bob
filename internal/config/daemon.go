package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

const defaultDaemonBind = "127.0.0.1:7331"

const daemonConfigFilename = "bobd.json"

type Daemon struct {
	Bind          string
	Token         string
	LocalhostOnly bool
}

type daemonFileConfig struct {
	Bind          string `json:"bind"`
	Token         string `json:"token"`
	LocalhostOnly *bool  `json:"localhost_only"`
}

func LoadDaemon() (Daemon, error) {
	fileCfg, _, err := loadJSON[daemonFileConfig](daemonConfigFilename)
	if err != nil {
		return Daemon{}, err
	}

	cfg := Daemon{
		Bind:          defaultDaemonBind,
		Token:         fileCfg.Token,
		LocalhostOnly: true,
	}
	if fileCfg.Bind != "" {
		cfg.Bind = fileCfg.Bind
	}
	if fileCfg.LocalhostOnly != nil {
		cfg.LocalhostOnly = *fileCfg.LocalhostOnly
	}

	if value := os.Getenv("BOBD_BIND"); value != "" {
		cfg.Bind = value
	}
	if value := os.Getenv("BOBD_TOKEN"); value != "" {
		cfg.Token = value
	}
	if raw := os.Getenv("BOBD_LOCALHOST_ONLY"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return Daemon{}, errors.New("BOBD_LOCALHOST_ONLY must be a boolean")
		}
		cfg.LocalhostOnly = parsed
	}

	if cfg.Token == "" {
		return Daemon{}, errors.New("BOBD_TOKEN is required or token must be set in bobd.json; run 'bobd init' to generate one")
	}

	return cfg, nil
}

func WriteDaemonConfig(cfg Daemon, force bool) (string, error) {
	if cfg.Token == "" {
		return "", errors.New("token is required")
	}
	path, err := writeJSON(daemonConfigFilename, daemonFileConfig{
		Bind:          cfg.Bind,
		Token:         cfg.Token,
		LocalhostOnly: &cfg.LocalhostOnly,
	}, force)
	if err != nil {
		return "", fmt.Errorf("write daemon config: %w", err)
	}
	return path, nil
}
