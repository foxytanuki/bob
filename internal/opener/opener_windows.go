//go:build windows

package opener

func platformCommand(rawURL string) (string, []string, error) {
	return "cmd", []string{"/c", "start", "", rawURL}, nil
}
