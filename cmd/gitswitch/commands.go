//go:build linux || darwin

package main

import (
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/stefanfaur/gitswitch/internal/crypto"
	"github.com/stefanfaur/gitswitch/internal/prompt"
	"github.com/stefanfaur/gitswitch/internal/session"
)

func cmdAdd(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: gitswitch add <name>")
	}
	name := args[0]
	s, err := openStore()
	if err != nil {
		return err
	}
	full, err := prompt.ReadLine("Full name:   ")
	if err != nil {
		return err
	}
	email, err := prompt.ReadLine("Email:       ")
	if err != nil {
		return err
	}
	pat, err := prompt.ReadPassword("Gitea PAT:   ")
	if err != nil {
		return err
	}
	pw, err := prompt.ReadPassword("Password:    ")
	if err != nil {
		return err
	}
	confirm, err := prompt.ReadPassword("Confirm:     ")
	if err != nil {
		return err
	}
	if pw != confirm {
		return errors.New("passwords do not match")
	}
	payload, err := encodePayload(blobPayload{Name: full, Email: email, PAT: pat})
	if err != nil {
		return err
	}
	blob, err := crypto.Encrypt(payload, []byte(pw))
	if err != nil {
		return err
	}
	if err := s.Write(name, blob); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Stored %s.\n", name)
	return nil
}

func cmdList(args []string) error {
	if len(args) != 0 {
		return errors.New("usage: gitswitch list")
	}
	s, err := openStore()
	if err != nil {
		return err
	}
	names, err := s.List()
	if err != nil {
		return err
	}
	sort.Strings(names)
	for _, n := range names {
		fmt.Println(n)
	}
	return nil
}

func cmdRm(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: gitswitch rm <name>")
	}
	name := args[0]
	s, err := openStore()
	if err != nil {
		return err
	}
	blob, err := s.Read(name)
	if err != nil {
		return err
	}
	pw, err := prompt.ReadPassword(fmt.Sprintf("Password for %s: ", name))
	if err != nil {
		return err
	}
	if _, err := crypto.Decrypt(blob, []byte(pw)); err != nil {
		return errors.New("wrong password")
	}
	if err := s.Remove(name); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Removed %s.\n", name)
	return nil
}

func cmdWhoami(args []string) error {
	if len(args) != 0 {
		return errors.New("usage: gitswitch whoami")
	}
	if u := os.Getenv("GITSWITCH_USER"); u != "" {
		fmt.Println(u)
	} else {
		fmt.Println("none")
	}
	return nil
}

func cmdUse(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: gitswitch use <name>")
	}
	name := args[0]
	if os.Getenv("GITSWITCH_USER") != "" {
		return fmt.Errorf("already in gitswitch session for %q; exit first", os.Getenv("GITSWITCH_USER"))
	}
	s, err := openStore()
	if err != nil {
		return err
	}
	blob, err := s.Read(name)
	if err != nil {
		return err
	}
	pw, err := prompt.ReadPassword(fmt.Sprintf("Password for %s: ", name))
	if err != nil {
		return err
	}
	plain, err := crypto.Decrypt(blob, []byte(pw))
	if err != nil {
		return errors.New("wrong password")
	}
	p, err := decodePayload(plain)
	if err != nil {
		return err
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	return session.Launch(session.Options{
		ParentEnv: os.Environ(),
		Identity:  session.Identity{Name: p.Name, Email: p.Email, PAT: p.PAT, User: name},
		SelfPath:  self,
		ExecFn:    sessionExec,
	})
}

func cmdRotate(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: gitswitch rotate <name>")
	}
	name := args[0]
	s, err := openStore()
	if err != nil {
		return err
	}
	blob, err := s.Read(name)
	if err != nil {
		return err
	}
	pw, err := prompt.ReadPassword(fmt.Sprintf("Password for %s: ", name))
	if err != nil {
		return err
	}
	plain, err := crypto.Decrypt(blob, []byte(pw))
	if err != nil {
		return errors.New("wrong password")
	}
	p, err := decodePayload(plain)
	if err != nil {
		return err
	}
	newPAT, err := prompt.ReadPassword("New PAT:     ")
	if err != nil {
		return err
	}
	p.PAT = newPAT
	payload, err := encodePayload(p)
	if err != nil {
		return err
	}
	newBlob, err := crypto.Encrypt(payload, []byte(pw))
	if err != nil {
		return err
	}
	if err := s.Overwrite(name, newBlob); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Rotated PAT for %s.\n", name)
	return nil
}

func cmdPasswd(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: gitswitch passwd <name>")
	}
	name := args[0]
	s, err := openStore()
	if err != nil {
		return err
	}
	blob, err := s.Read(name)
	if err != nil {
		return err
	}
	oldPW, err := prompt.ReadPassword("Old password: ")
	if err != nil {
		return err
	}
	plain, err := crypto.Decrypt(blob, []byte(oldPW))
	if err != nil {
		return errors.New("wrong password")
	}
	newPW, err := prompt.ReadPassword("New password: ")
	if err != nil {
		return err
	}
	confirm, err := prompt.ReadPassword("Confirm:      ")
	if err != nil {
		return err
	}
	if newPW != confirm {
		return errors.New("passwords do not match")
	}
	newBlob, err := crypto.Encrypt(plain, []byte(newPW))
	if err != nil {
		return err
	}
	if err := s.Overwrite(name, newBlob); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Password changed for %s.\n", name)
	return nil
}
