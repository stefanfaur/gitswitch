// Package askpass implements the GIT_ASKPASS mode. Called by git during HTTPS
// auth; reads identity from env. Must never fail loudly — git falls back to
// interactive tty on empty output.
package askpass

import (
	"fmt"
	"io"
	"strings"
)

func Run(args []string, env map[string]string, w io.Writer) int {
	if len(args) == 0 {
		return 1
	}
	user := env["GITSWITCH_USER"]
	pat := env["GITSWITCH_PAT"]
	if user == "" || pat == "" {
		return 1
	}
	prompt := strings.ToLower(args[0])
	if strings.HasPrefix(strings.TrimSpace(prompt), "username") {
		fmt.Fprintln(w, user)
	} else {
		fmt.Fprintln(w, pat)
	}
	return 0
}
