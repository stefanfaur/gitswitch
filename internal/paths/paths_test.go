package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigDirXDG(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	got, err := ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "gitswitch")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestConfigDirFallback(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	home, _ := os.UserHomeDir()
	got, err := ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".config", "gitswitch")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
