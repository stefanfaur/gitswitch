//go:build linux || darwin

package main_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var bin string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "gs-bin-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)
	bin = filepath.Join(tmp, "gitswitch")
	build := exec.Command("go", "build", "-o", bin, "./")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func run(t *testing.T, home string, stdin string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Env = append(os.Environ(),
		"XDG_CONFIG_HOME="+home,
		"HOME="+home,
	)
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("run: %v", err)
	}
	return stdout.String(), stderr.String(), code
}

func TestAddListWhoamiRm(t *testing.T) {
	home := t.TempDir()
	stdin := "Alice Example\nalice@example.com\ngtea_testtoken\nhunter2\nhunter2\n"
	if out, errOut, code := run(t, home, stdin, "add", "alice"); code != 0 {
		t.Fatalf("add: code=%d out=%s err=%s", code, out, errOut)
	}

	if out, _, code := run(t, home, "", "list"); code != 0 || !strings.Contains(out, "alice") {
		t.Fatalf("list: code=%d out=%q", code, out)
	}

	if out, _, code := run(t, home, "", "whoami"); code != 0 || strings.TrimSpace(out) != "none" {
		t.Fatalf("whoami: code=%d out=%q", code, out)
	}

	if _, _, code := run(t, home, "wrongpass\n", "rm", "alice"); code == 0 {
		t.Fatalf("expected rm with wrong password to fail")
	}
	if _, errOut, code := run(t, home, "hunter2\n", "rm", "alice"); code != 0 {
		t.Fatalf("rm: code=%d err=%s", code, errOut)
	}
	if out, _, _ := run(t, home, "", "list"); strings.Contains(out, "alice") {
		t.Fatalf("alice still listed after rm: %q", out)
	}
}
