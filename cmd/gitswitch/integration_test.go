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

func TestUseExecsShellWithEnv(t *testing.T) {
	home := t.TempDir()
	stdin := "Alice Example\nalice@example.com\ngtea_xyz\nhunter2\nhunter2\n"
	if _, _, code := run(t, home, stdin, "add", "alice"); code != 0 {
		t.Fatal("setup add failed")
	}
	shell := filepath.Join(t.TempDir(), "fakeshell.sh")
	script := "#!/bin/sh\nenv | grep -E '^(GIT_|GITSWITCH_|LC_MESSAGES=)' | sort\n"
	if err := os.WriteFile(shell, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "use", "alice")
	cmd.Env = append(os.Environ(),
		"XDG_CONFIG_HOME="+home,
		"HOME="+home,
		"SHELL="+shell,
	)
	cmd.Stdin = strings.NewReader("hunter2\n")
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		t.Fatalf("use: %v err=%s", err, errOut.String())
	}
	s := out.String()
	for _, want := range []string{
		"GIT_AUTHOR_NAME=Alice Example",
		"GIT_AUTHOR_EMAIL=alice@example.com",
		"GIT_COMMITTER_NAME=Alice Example",
		"GIT_COMMITTER_EMAIL=alice@example.com",
		"GITSWITCH_USER=alice",
		"GITSWITCH_PAT=gtea_xyz",
		"GITSWITCH_ASKPASS=1",
		"LC_MESSAGES=C",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in:\n%s", want, s)
		}
	}
	if !strings.Contains(s, "GIT_ASKPASS=") {
		t.Fatalf("GIT_ASKPASS missing:\n%s", s)
	}
}

func TestUseRefusesNested(t *testing.T) {
	home := t.TempDir()
	stdin := "A\na@x\np\nhunter2\nhunter2\n"
	if _, _, code := run(t, home, stdin, "add", "alice"); code != 0 {
		t.Fatal("setup")
	}
	cmd := exec.Command(bin, "use", "alice")
	cmd.Env = append(os.Environ(),
		"XDG_CONFIG_HOME="+home, "HOME="+home, "SHELL=/bin/sh",
		"GITSWITCH_USER=bob",
	)
	cmd.Stdin = strings.NewReader("hunter2\n")
	var errOut bytes.Buffer
	cmd.Stderr = &errOut
	if err := cmd.Run(); err == nil {
		t.Fatalf("expected nesting refusal, stderr=%s", errOut.String())
	}
}

func TestAskpassSubprocess(t *testing.T) {
	cmd := exec.Command(bin, "Username for 'https://gitea': ")
	cmd.Env = []string{
		"GITSWITCH_ASKPASS=1",
		"GITSWITCH_USER=alice",
		"GITSWITCH_PAT=tok",
	}
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != "alice" {
		t.Fatalf("got %q", string(out))
	}

	cmd = exec.Command(bin, "Password for ...: ")
	cmd.Env = []string{
		"GITSWITCH_ASKPASS=1",
		"GITSWITCH_USER=alice",
		"GITSWITCH_PAT=tok",
	}
	out, err = cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != "tok" {
		t.Fatalf("got %q", string(out))
	}
}
