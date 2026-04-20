//go:build linux || darwin

package prompt

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"
)

func TestMain(m *testing.M) {
	switch os.Getenv("GITSWITCH_PROMPT_HELPER") {
	case "read":
		pw, err := ReadPassword("Password: ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		fmt.Print(pw)
		os.Exit(0)
	case "signal":
		fd := int(os.Stdin.Fd())
		before, err := term.GetState(fd)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		_, _ = ReadPassword("Password: ")
		after, err := term.GetState(fd)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		_ = after
		_ = before
		fmt.Println("RESTORED")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestReadPasswordEchoOff(t *testing.T) {
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), "GITSWITCH_PROMPT_HELPER=read")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		t.Fatal(err)
	}
	defer ptmx.Close()

	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, ptmx)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	ptmx.Write([]byte("hunter2\n"))
	if err := cmd.Wait(); err != nil {
		t.Fatalf("child: %v, output=%q", err, buf.String())
	}
	ptmx.Close()
	<-done

	out := buf.String()
	firstLine := strings.SplitN(out, "\n", 2)[0]
	if strings.Contains(firstLine, "hunter2") {
		t.Fatalf("password echoed on prompt line: %q", out)
	}
	if !strings.Contains(out, "hunter2") {
		t.Fatalf("child did not print password: %q", out)
	}
}

func TestReadPasswordSIGINTReturnsError(t *testing.T) {
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), "GITSWITCH_PROMPT_HELPER=signal")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		t.Fatal(err)
	}
	defer ptmx.Close()

	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, ptmx)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	_ = cmd.Process.Signal(os.Interrupt)
	if err := cmd.Wait(); err != nil {
		t.Fatalf("child exit: %v, output=%q", err, buf.String())
	}
	ptmx.Close()
	<-done
	if !strings.Contains(buf.String(), "RESTORED") {
		t.Fatalf("signal path did not reach restore sentinel: %q", buf.String())
	}
}
