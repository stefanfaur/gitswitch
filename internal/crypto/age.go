// Package crypto wraps filippo.io/age with a pinned scrypt passphrase recipient
// for gitswitch blobs.
package crypto

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"

	"filippo.io/age"
)

const (
	MaxInputBytes    = 1024
	ScryptWorkFactor = 18
)

func Encrypt(plaintext, password []byte) ([]byte, error) {
	if err := validate(plaintext, password); err != nil {
		return nil, err
	}
	r, err := age.NewScryptRecipient(string(password))
	if err != nil {
		return nil, fmt.Errorf("recipient: %w", err)
	}
	r.SetWorkFactor(ScryptWorkFactor)
	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, r)
	if err != nil {
		return nil, fmt.Errorf("encrypt: %w", err)
	}
	if _, err := w.Write(plaintext); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Decrypt(blob, password []byte) ([]byte, error) {
	if len(password) == 0 || len(password) > MaxInputBytes {
		return nil, errors.New("password size out of range")
	}
	if !utf8.Valid(password) {
		return nil, errors.New("password must be valid utf-8")
	}
	id, err := age.NewScryptIdentity(string(password))
	if err != nil {
		return nil, fmt.Errorf("identity: %w", err)
	}
	r, err := age.Decrypt(bytes.NewReader(blob), id)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	out, err := io.ReadAll(io.LimitReader(r, MaxInputBytes+1))
	if err != nil {
		return nil, err
	}
	if len(out) > MaxInputBytes {
		return nil, errors.New("plaintext exceeds size cap")
	}
	return out, nil
}

func validate(plaintext, password []byte) error {
	if len(plaintext) == 0 || len(plaintext) > MaxInputBytes {
		return errors.New("plaintext size out of range")
	}
	if len(password) == 0 || len(password) > MaxInputBytes {
		return errors.New("password size out of range")
	}
	if !utf8.Valid(plaintext) {
		return errors.New("plaintext must be valid utf-8")
	}
	if !utf8.Valid(password) {
		return errors.New("password must be valid utf-8")
	}
	return nil
}
