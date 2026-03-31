package config

import (
	"errors"
	"os"
	"strconv"
)

const defaultDaemonBind = "127.0.0.1:7331"

type Daemon struct {
	Bind          string
	Token         string
	LocalhostOnly bool
}

func LoadDaemonFromEnv() (Daemon, error) {
	token := os.Getenv("BOBD_TOKEN")
	if token == "" {
		return Daemon{}, errors.New("BOBD_TOKEN is required; run 'bobd init' to generate one")
	}

	localhostOnly := true
	if raw := os.Getenv("BOBD_LOCALHOST_ONLY"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return Daemon{}, errors.New("BOBD_LOCALHOST_ONLY must be a boolean")
		}
		localhostOnly = parsed
	}

	return Daemon{
		Bind:          getenvDefault("BOBD_BIND", defaultDaemonBind),
		Token:         token,
		LocalhostOnly: localhostOnly,
	}, nil
}
