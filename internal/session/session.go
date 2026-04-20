// Package session builds the env for a `gitswitch use` subshell and execs it.
// syscall.Exec is injected so behavior can be unit-tested.
package session

import (
	"errors"
	"fmt"
	"strings"
)

type Identity struct {
	Name, Email, PAT, User string
}

type Options struct {
	ParentEnv []string
	Identity  Identity
	SelfPath  string
	ExecFn    func(argv0 string, argv []string, env []string) error
}

var stripPrefixes = []string{
	"GITSWITCH_",
	"GIT_AUTHOR_",
	"GIT_COMMITTER_",
	"GIT_CONFIG",
}

var stripExact = map[string]struct{}{
	"GIT_ASKPASS": {},
	"SSH_ASKPASS": {},
	"LC_MESSAGES": {},
}

func BuildEnv(parent []string, id Identity, selfPath string) []string {
	out := make([]string, 0, len(parent)+10)
	for _, kv := range parent {
		eq := strings.IndexByte(kv, '=')
		if eq < 0 {
			continue
		}
		k := kv[:eq]
		if _, drop := stripExact[k]; drop {
			continue
		}
		dropPrefix := false
		for _, p := range stripPrefixes {
			if strings.HasPrefix(k, p) {
				dropPrefix = true
				break
			}
		}
		if dropPrefix {
			continue
		}
		out = append(out, kv)
	}
	out = append(out,
		"GIT_AUTHOR_NAME="+id.Name,
		"GIT_AUTHOR_EMAIL="+id.Email,
		"GIT_COMMITTER_NAME="+id.Name,
		"GIT_COMMITTER_EMAIL="+id.Email,
		"GIT_ASKPASS="+selfPath,
		"GITSWITCH_ASKPASS=1",
		"GITSWITCH_USER="+id.User,
		"GITSWITCH_PAT="+id.PAT,
		"LC_MESSAGES=C",
	)
	return out
}

func Launch(o Options) error {
	for _, kv := range o.ParentEnv {
		if strings.HasPrefix(kv, "GITSWITCH_USER=") {
			v := strings.TrimPrefix(kv, "GITSWITCH_USER=")
			if v != "" {
				return fmt.Errorf("already in gitswitch session for %q; exit first", v)
			}
		}
	}
	shell := "/bin/sh"
	for _, kv := range o.ParentEnv {
		if strings.HasPrefix(kv, "SHELL=") {
			if v := strings.TrimPrefix(kv, "SHELL="); v != "" {
				shell = v
			}
			break
		}
	}
	env := BuildEnv(o.ParentEnv, o.Identity, o.SelfPath)
	if o.ExecFn == nil {
		return errors.New("session: ExecFn nil")
	}
	return o.ExecFn(shell, []string{shell, "-l"}, env)
}
