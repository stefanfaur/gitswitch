package store

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	base := filepath.Join(dir, "gitswitch")
	if err := os.Mkdir(base, 0o700); err != nil {
		t.Fatal(err)
	}
	s, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestWriteReadListRemove(t *testing.T) {
	s := newStore(t)
	if err := s.Write("alice", []byte("blob-a")); err != nil {
		t.Fatal(err)
	}
	if err := s.Write("bob", []byte("blob-b")); err != nil {
		t.Fatal(err)
	}
	names, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(names)
	if len(names) != 2 || names[0] != "alice" || names[1] != "bob" {
		t.Fatalf("list: %v", names)
	}
	got, err := s.Read("alice")
	if err != nil || string(got) != "blob-a" {
		t.Fatalf("read: %q %v", got, err)
	}
	if err := s.Remove("alice"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read("alice"); !IsNotFound(err) {
		t.Fatalf("expected not-found, got %v", err)
	}
}

func TestRejectExistingOnWrite(t *testing.T) {
	s := newStore(t)
	_ = s.Write("alice", []byte("x"))
	if err := s.Write("alice", []byte("y")); err == nil {
		t.Fatal("expected error on existing blob")
	}
}

func TestOverwriteAllowedByWriteForce(t *testing.T) {
	s := newStore(t)
	_ = s.Write("alice", []byte("x"))
	if err := s.Overwrite("alice", []byte("y")); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Read("alice")
	if string(got) != "y" {
		t.Fatalf("overwrite: %q", got)
	}
}

func TestFileModeEnforced(t *testing.T) {
	s := newStore(t)
	_ = s.Write("alice", []byte("x"))
	p := filepath.Join(s.Base(), "alice.age")
	if err := os.Chmod(p, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read("alice"); err == nil {
		t.Fatal("expected refusal on wrong mode")
	}
}

func TestSymlinkRejectedOnRead(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	s := newStore(t)
	target := filepath.Join(t.TempDir(), "evil")
	os.WriteFile(target, []byte("evil"), 0o600)
	if err := os.Symlink(target, filepath.Join(s.Base(), "alice.age")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read("alice"); err == nil {
		t.Fatal("expected symlink refusal")
	}
}

func TestRejectDirNotOwnedOrWrongMode(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "gitswitch")
	if err := os.Mkdir(base, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(base); err == nil {
		t.Fatal("expected dir-mode refusal")
	}
}

func TestInvalidName(t *testing.T) {
	s := newStore(t)
	for _, n := range []string{"", "../x", "a/b", "a.b", ".hidden", "withspace "} {
		if err := s.Write(n, []byte("x")); err == nil {
			t.Fatalf("expected rejection of %q", n)
		}
	}
}
