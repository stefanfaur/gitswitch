package session

import (
	"sort"
	"strings"
	"testing"
)

func TestBuildEnvStripsSensitive(t *testing.T) {
	parent := []string{
		"HOME=/home/shared",
		"SHELL=/bin/bash",
		"PATH=/usr/bin",
		"GITSWITCH_USER=stale",
		"GITSWITCH_PAT=stale-pat",
		"GIT_AUTHOR_NAME=stale",
		"GIT_AUTHOR_EMAIL=stale",
		"GIT_COMMITTER_NAME=stale",
		"GIT_COMMITTER_EMAIL=stale",
		"GIT_ASKPASS=/old/path",
		"SSH_ASKPASS=/old/ssh",
		"GIT_CONFIG=/evil",
		"GIT_CONFIG_GLOBAL=/evil",
		"GIT_CONFIG_SYSTEM=/evil",
		"LC_MESSAGES=de_DE.UTF-8",
	}
	got := BuildEnv(parent, Identity{
		Name: "Alice", Email: "a@x", PAT: "tok-new", User: "alice",
	}, "/opt/bin/gitswitch")

	sort.Strings(got)
	joined := strings.Join(got, "\n")

	mustContain := []string{
		"HOME=/home/shared",
		"PATH=/usr/bin",
		"GITSWITCH_USER=alice",
		"GITSWITCH_PAT=tok-new",
		"GITSWITCH_ASKPASS=1",
		"GIT_AUTHOR_NAME=Alice",
		"GIT_AUTHOR_EMAIL=a@x",
		"GIT_COMMITTER_NAME=Alice",
		"GIT_COMMITTER_EMAIL=a@x",
		"GIT_ASKPASS=/opt/bin/gitswitch",
		"LC_MESSAGES=C",
	}
	for _, want := range mustContain {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in env:\n%s", want, joined)
		}
	}
	mustNotContain := []string{
		"GITSWITCH_USER=stale",
		"GITSWITCH_PAT=stale-pat",
		"GIT_AUTHOR_NAME=stale",
		"GIT_CONFIG=/evil",
		"GIT_CONFIG_GLOBAL=/evil",
		"GIT_CONFIG_SYSTEM=/evil",
		"SSH_ASKPASS=/old/ssh",
		"LC_MESSAGES=de_DE.UTF-8",
	}
	for _, bad := range mustNotContain {
		if strings.Contains(joined, bad) {
			t.Fatalf("leaked %q in env:\n%s", bad, joined)
		}
	}
}

func TestLaunchRefusesIfAlreadyInSession(t *testing.T) {
	called := false
	exec := func(argv0 string, argv []string, env []string) error {
		called = true
		return nil
	}
	err := Launch(Options{
		ParentEnv: []string{"GITSWITCH_USER=bob", "SHELL=/bin/sh"},
		Identity:  Identity{Name: "A", Email: "a", PAT: "p", User: "alice"},
		SelfPath:  "/opt/gitswitch",
		ExecFn:    exec,
	})
	if err == nil || called {
		t.Fatalf("expected refusal, err=%v called=%v", err, called)
	}
}

func TestLaunchExecsShell(t *testing.T) {
	var gotArgv0 string
	var gotArgv []string
	exec := func(argv0 string, argv []string, env []string) error {
		gotArgv0 = argv0
		gotArgv = argv
		return nil
	}
	err := Launch(Options{
		ParentEnv: []string{"SHELL=/bin/zsh"},
		Identity:  Identity{Name: "A", Email: "a", PAT: "p", User: "alice"},
		SelfPath:  "/opt/gitswitch",
		ExecFn:    exec,
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotArgv0 != "/bin/zsh" || len(gotArgv) < 2 || gotArgv[1] != "-l" {
		t.Fatalf("argv0=%q argv=%v", gotArgv0, gotArgv)
	}
}

func TestLaunchFallbackShell(t *testing.T) {
	var gotArgv0 string
	exec := func(argv0 string, argv []string, env []string) error {
		gotArgv0 = argv0
		return nil
	}
	err := Launch(Options{
		ParentEnv: []string{},
		Identity:  Identity{Name: "A", Email: "a", PAT: "p", User: "alice"},
		SelfPath:  "/opt/gitswitch",
		ExecFn:    exec,
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotArgv0 != "/bin/sh" {
		t.Fatalf("expected fallback /bin/sh, got %q", gotArgv0)
	}
}
