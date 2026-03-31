//go:build linux

package opener

func platformCommand(rawURL string) (string, []string, error) {
	return "xdg-open", []string{rawURL}, nil
}
