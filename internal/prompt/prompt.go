// Package prompt owns all interactive tty input for gitswitch. It guarantees
// that tty echo state is restored on any exit path, including SIGINT/SIGTERM.
package prompt

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"unicode/utf8"

	"golang.org/x/term"
)

const maxInput = 1024

func ReadPassword(prompt string) (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		r := bufio.NewReader(os.Stdin)
		line, err := r.ReadString('\n')
		if err != nil && line == "" {
			return "", err
		}
		line = trimNewline(line)
		if err := validate(line); err != nil {
			return "", err
		}
		return line, nil
	}
	saved, err := term.GetState(fd)
	if err != nil {
		return "", err
	}
	var once sync.Once
	restore := func() { _ = term.Restore(fd, saved) }

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(sigCh)

	done := make(chan struct{})
	var (
		pw    []byte
		ioErr error
	)
	go func() {
		defer close(done)
		fmt.Fprint(os.Stderr, prompt)
		pw, ioErr = term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr)
	}()

	select {
	case <-done:
	case sig := <-sigCh:
		once.Do(restore)
		fmt.Fprintln(os.Stderr)
		return "", fmt.Errorf("aborted by signal: %v", sig)
	}
	once.Do(restore)
	if ioErr != nil {
		return "", ioErr
	}
	s := string(pw)
	if err := validate(s); err != nil {
		return "", err
	}
	return s, nil
}

func ReadLine(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	line = trimNewline(line)
	if err := validate(line); err != nil {
		return "", err
	}
	return line, nil
}

func validate(s string) error {
	if len(s) == 0 {
		return errors.New("empty input")
	}
	if len(s) > maxInput {
		return fmt.Errorf("input exceeds %d bytes", maxInput)
	}
	if !utf8.ValidString(s) {
		return errors.New("input is not valid utf-8")
	}
	return nil
}

func trimNewline(s string) string {
	if len(s) > 0 && s[len(s)-1] == '\n' {
		s = s[:len(s)-1]
	}
	if len(s) > 0 && s[len(s)-1] == '\r' {
		s = s[:len(s)-1]
	}
	return s
}
