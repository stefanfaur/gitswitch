// Package store owns the on-disk layout of gitswitch blobs and enforces
// TOCTOU-safe read/write semantics. Opaque bytes — encryption lives in crypto.
package store

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

var nameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)

type Store struct{ base string }

type ErrNotFound struct{ Name string }

func (e *ErrNotFound) Error() string { return fmt.Sprintf("no such user %q", e.Name) }

func IsNotFound(err error) bool {
	var e *ErrNotFound
	return errors.As(err, &e)
}

func Open(base string) (*Store, error) {
	st, err := lstatRegularDir(base)
	if err != nil {
		return nil, err
	}
	if st.Mode().Perm() != 0o700 {
		return nil, fmt.Errorf("base dir %s: mode is %o, want 0700", base, st.Mode().Perm())
	}
	if err := checkOwner(st); err != nil {
		return nil, fmt.Errorf("base dir %s: %w", base, err)
	}
	return &Store{base: base}, nil
}

func EnsureBase(base string) error {
	if _, err := os.Lstat(base); err == nil {
		return nil
	}
	return os.Mkdir(base, 0o700)
}

func (s *Store) Base() string { return s.base }

func (s *Store) path(name string) (string, error) {
	if !nameRe.MatchString(name) {
		return "", fmt.Errorf("invalid user name %q", name)
	}
	return filepath.Join(s.base, name+".age"), nil
}

func (s *Store) Write(name string, data []byte) error {
	return s.writeBlob(name, data, false)
}

func (s *Store) Overwrite(name string, data []byte) error {
	return s.writeBlob(name, data, true)
}

func (s *Store) writeBlob(name string, data []byte, overwrite bool) error {
	p, err := s.path(name)
	if err != nil {
		return err
	}
	if !overwrite {
		if _, err := os.Lstat(p); err == nil {
			return fmt.Errorf("%q already exists", name)
		}
	}
	tmp := p + ".tmp"
	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL | unix.O_NOFOLLOW
	f, err := os.OpenFile(tmp, flags, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, p)
}

func (s *Store) Read(name string) ([]byte, error) {
	p, err := s.path(name)
	if err != nil {
		return nil, err
	}
	st, err := os.Lstat(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, &ErrNotFound{Name: name}
		}
		return nil, err
	}
	if !st.Mode().IsRegular() {
		return nil, fmt.Errorf("%s: not a regular file", p)
	}
	if st.Mode().Perm() != 0o600 {
		return nil, fmt.Errorf("%s: mode is %o, want 0600", p, st.Mode().Perm())
	}
	if err := checkOwner(st); err != nil {
		return nil, fmt.Errorf("%s: %w", p, err)
	}
	f, err := os.OpenFile(p, os.O_RDONLY|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, 1<<16))
}

func (s *Store) Remove(name string) error {
	p, err := s.path(name)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(p); errors.Is(err, os.ErrNotExist) {
		return &ErrNotFound{Name: name}
	}
	return os.Remove(p)
}

func (s *Store) List() ([]string, error) {
	entries, err := os.ReadDir(s.base)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		n := e.Name()
		if !strings.HasSuffix(n, ".age") {
			continue
		}
		out = append(out, strings.TrimSuffix(n, ".age"))
	}
	return out, nil
}

func lstatRegularDir(p string) (os.FileInfo, error) {
	st, err := os.Lstat(p)
	if err != nil {
		return nil, err
	}
	if st.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("%s: is a symlink", p)
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("%s: not a directory", p)
	}
	return st, nil
}

func checkOwner(st os.FileInfo) error {
	sys, ok := st.Sys().(*syscall.Stat_t)
	if !ok {
		return nil
	}
	if sys.Uid != uint32(os.Getuid()) {
		return errors.New("owner is not current uid")
	}
	return nil
}
