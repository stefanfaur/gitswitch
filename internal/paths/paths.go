// Package paths resolves gitswitch's XDG config directory.
package paths

import (
	"errors"
	"os"
	"path/filepath"
)

func ConfigDir() (string, error) {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "gitswitch"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", errors.New("cannot determine home directory")
	}
	return filepath.Join(home, ".config", "gitswitch"), nil
}
