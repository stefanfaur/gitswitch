package askpass

import (
	"bytes"
	"testing"
)

func TestUsernamePrompt(t *testing.T) {
	env := map[string]string{"GITSWITCH_USER": "alice", "GITSWITCH_PAT": "tok"}
	var out bytes.Buffer
	code := Run([]string{"Username for 'https://gitea.example.com': "}, env, &out)
	if code != 0 || out.String() != "alice\n" {
		t.Fatalf("code=%d out=%q", code, out.String())
	}
}

func TestUsernameCaseInsensitive(t *testing.T) {
	env := map[string]string{"GITSWITCH_USER": "alice", "GITSWITCH_PAT": "tok"}
	var out bytes.Buffer
	code := Run([]string{"username for ...: "}, env, &out)
	if code != 0 || out.String() != "alice\n" {
		t.Fatalf("code=%d out=%q", code, out.String())
	}
}

func TestPasswordPrompt(t *testing.T) {
	env := map[string]string{"GITSWITCH_USER": "alice", "GITSWITCH_PAT": "tok"}
	var out bytes.Buffer
	code := Run([]string{"Password for 'https://alice@...': "}, env, &out)
	if code != 0 || out.String() != "tok\n" {
		t.Fatalf("code=%d out=%q", code, out.String())
	}
}

func TestMissingEnv(t *testing.T) {
	var out bytes.Buffer
	code := Run([]string{"Password: "}, map[string]string{}, &out)
	if code != 1 || out.Len() != 0 {
		t.Fatalf("code=%d out=%q", code, out.String())
	}
}

func TestMissingArg(t *testing.T) {
	env := map[string]string{"GITSWITCH_USER": "alice", "GITSWITCH_PAT": "tok"}
	var out bytes.Buffer
	code := Run(nil, env, &out)
	if code != 1 {
		t.Fatalf("code=%d", code)
	}
}
