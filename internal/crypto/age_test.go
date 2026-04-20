package crypto

import (
	"bytes"
	"strings"
	"testing"
)

func TestRoundtrip(t *testing.T) {
	plaintext := []byte(`{"name":"Alice","email":"a@x","pat":"tok"}`)
	blob, err := Encrypt(plaintext, []byte("correct horse"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	got, err := Decrypt(blob, []byte("correct horse"))
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("roundtrip mismatch: got %q", got)
	}
}

func TestWrongPassword(t *testing.T) {
	blob, _ := Encrypt([]byte("x"), []byte("a"))
	if _, err := Decrypt(blob, []byte("b")); err == nil {
		t.Fatal("expected error on wrong password")
	}
}

func TestTruncatedBlob(t *testing.T) {
	blob, _ := Encrypt([]byte("x"), []byte("a"))
	if _, err := Decrypt(blob[:len(blob)/2], []byte("a")); err == nil {
		t.Fatal("expected error on truncated blob")
	}
}

func TestInputCapsOversize(t *testing.T) {
	big := bytes.Repeat([]byte("a"), 1025)
	if _, err := Encrypt(big, []byte("p")); err == nil {
		t.Fatal("expected error on >1 KiB plaintext")
	}
	if _, err := Encrypt([]byte("x"), big); err == nil {
		t.Fatal("expected error on >1 KiB password")
	}
}

func TestRejectNonUTF8Password(t *testing.T) {
	if _, err := Encrypt([]byte("x"), []byte{0xff, 0xfe}); err == nil || !strings.Contains(err.Error(), "utf-8") {
		t.Fatalf("expected utf-8 error, got %v", err)
	}
}
