//go:build linux || darwin

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"syscall"

	"github.com/stefanfaur/gitswitch/internal/askpass"
	"github.com/stefanfaur/gitswitch/internal/harden"
	"github.com/stefanfaur/gitswitch/internal/paths"
	"github.com/stefanfaur/gitswitch/internal/store"
)

func main() {
	harden.Apply()

	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]
	switch cmd {
	case "add":
		fatal(cmdAdd(args))
	case "list":
		fatal(cmdList(args))
	case "rm":
		fatal(cmdRm(args))
	case "use":
		fatal(cmdUse(args))
	case "rotate":
		fatal(cmdRotate(args))
	case "passwd":
		fatal(cmdPasswd(args))
	case "whoami":
		fatal(cmdWhoami(args))
	case "-h", "--help", "help":
		usage()
	default:
		// Unknown first arg. If git invoked us as GIT_ASKPASS, the arg is a
		// prompt string (e.g. "Username for 'https://...': "), not a subcommand.
		if os.Getenv("GITSWITCH_ASKPASS") == "1" {
			os.Exit(askpass.Run(os.Args[1:], envMap(), os.Stdout))
		}
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n", cmd)
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `Usage: gitswitch <command> [args]
  add <name>      configure a new user
  list            list configured users
  rm <name>       remove a user
  use <name>      launch subshell with user's git creds
  rotate <name>   replace PAT for an existing user
  passwd <name>   change password for an existing user
  whoami          print active gitswitch user or "none"`)
}

func fatal(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, "gitswitch:", err)
	os.Exit(1)
}

func envMap() map[string]string {
	m := make(map[string]string, len(os.Environ()))
	for _, kv := range os.Environ() {
		for i := 0; i < len(kv); i++ {
			if kv[i] == '=' {
				m[kv[:i]] = kv[i+1:]
				break
			}
		}
	}
	return m
}

func openStore() (*store.Store, error) {
	base, err := paths.ConfigDir()
	if err != nil {
		return nil, err
	}
	if err := store.EnsureBase(base); err != nil {
		return nil, err
	}
	return store.Open(base)
}

type blobPayload struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	PAT   string `json:"pat"`
}

func encodePayload(p blobPayload) ([]byte, error) { return json.Marshal(p) }

func decodePayload(data []byte) (blobPayload, error) {
	var p blobPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return p, fmt.Errorf("corrupt blob: %w", err)
	}
	return p, nil
}

var execShell = syscall.Exec

func sessionExec(argv0 string, argv []string, env []string) error {
	return execShell(argv0, argv, env)
}
