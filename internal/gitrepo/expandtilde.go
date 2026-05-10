package gitrepo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// expandTilde resolves a leading "~/" or bare "~" to the user's home dir, the
// only tilde form git expands in core.excludesfile.
func expandTilde(p string) (string, error) {
	switch {
	case p == "~" || strings.HasPrefix(p, "~/"):
		// Tilde expansion needed.
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		if home == "" {
			return "", fmt.Errorf("home directory is empty")
		}

		if p == "~" {
			return home, nil
		}

		return filepath.Join(home, p[2:]), nil

	case strings.HasPrefix(p, "~"):
		// Tilde expansion needed, but resolving usernames is unsupported.
		return "", fmt.Errorf("unsupported tilde expansion %q: only ~ and ~/ are supported", p)

	default:
		// No tilde expansion needed.
		return p, nil
	}
}
