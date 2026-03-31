//go:build darwin

package opener

func platformCommand(rawURL string) (string, []string, error) {
	return "open", []string{rawURL}, nil
}
