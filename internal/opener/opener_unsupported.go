//go:build !linux && !darwin && !windows

package opener

import "errors"

func platformCommand(rawURL string) (string, []string, error) {
	return "", nil, errors.New("unsupported platform")
}
