package bobcli

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"bob/internal/config"
)

func runCodeServer(args []string, stderr io.Writer) int {
	parsed, err := parseCodeServerArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	cfg, err := config.LoadCLI()
	if err != nil {
		fmt.Fprintf(stderr, "config error: %v\n", err)
		return 1
	}

	port := cfg.CodeServer.Port
	if parsed.portSet {
		port = parsed.port
	} else if raw := os.Getenv("BOB_CODE_SERVER_PORT"); raw != "" {
		parsedPort, err := strconv.Atoi(raw)
		if err != nil {
			fmt.Fprintf(stderr, "invalid code-server port: BOB_CODE_SERVER_PORT must be a number\n")
			return 1
		}
		port = parsedPort
	}
	if err := config.ValidatePort(port); err != nil {
		fmt.Fprintf(stderr, "invalid code-server port: %v\n", err)
		return 1
	}

	absPath, err := resolveCodeServerPath(parsed.path)
	if err != nil {
		fmt.Fprintf(stderr, "resolve code-server path: %v\n", err)
		return 1
	}

	rawURL := fmt.Sprintf("http://127.0.0.1:%d/?folder=%s", port, url.QueryEscape(absPath))
	return openURLWithFailureStatuses(rawURL, stderr, map[string]bool{
		"SESSION_REQUIRED":  true,
		"SESSION_NOT_FOUND": true,
		"MIRROR_FAILED":     true,
	})
}

type codeServerArgs struct {
	path    string
	port    int
	portSet bool
}

func parseCodeServerArgs(args []string) (codeServerArgs, error) {
	parsed := codeServerArgs{}
	paths := make([]string, 0, 1)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--port":
			if i+1 >= len(args) {
				return codeServerArgs{}, fmt.Errorf("usage: bob code-server [--port <port>] [path]")
			}
			port, err := strconv.Atoi(args[i+1])
			if err != nil {
				return codeServerArgs{}, fmt.Errorf("invalid code-server port: must be a number")
			}
			parsed.port = port
			parsed.portSet = true
			i++
		case strings.HasPrefix(arg, "--port="):
			port, err := strconv.Atoi(strings.TrimPrefix(arg, "--port="))
			if err != nil {
				return codeServerArgs{}, fmt.Errorf("invalid code-server port: must be a number")
			}
			parsed.port = port
			parsed.portSet = true
		case strings.HasPrefix(arg, "-"):
			return codeServerArgs{}, fmt.Errorf("unknown code-server flag: %s", arg)
		default:
			paths = append(paths, arg)
		}
	}
	if len(paths) > 1 {
		return codeServerArgs{}, fmt.Errorf("usage: bob code-server [--port <port>] [path]")
	}
	if len(paths) == 1 {
		parsed.path = paths[0]
	}
	return parsed, nil
}

func resolveCodeServerPath(path string) (string, error) {
	if path == "" {
		path = "."
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			path = home
		} else {
			path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return filepath.Abs(path)
}
