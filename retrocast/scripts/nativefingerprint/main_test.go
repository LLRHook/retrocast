package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestArchiveContentChangesGoCacheInputAtStablePaths(t *testing.T) {
	directory := t.TempDir()
	archive := filepath.Join(directory, "permissions.a")
	output := filepath.Join(directory, "native_fingerprint.go")
	if err := os.WriteFile(archive, []byte("first native implementation"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := writeFingerprint(archive, output); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archive, []byte("second native implementation"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := writeFingerprint(archive, output); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(first, second) {
		t.Fatal("changed native bytes did not change the Go package input")
	}
}

func TestUnchangedArchivePreservesSourceTimestamp(t *testing.T) {
	directory := t.TempDir()
	archive := filepath.Join(directory, "permissions.a")
	output := filepath.Join(directory, "native_fingerprint.go")
	if err := os.WriteFile(archive, []byte("same native bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := writeFingerprint(archive, output); err != nil {
		t.Fatal(err)
	}
	stamp := time.Unix(100, 0)
	if err := os.Chtimes(output, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	if err := writeFingerprint(archive, output); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(output)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(stamp) {
		t.Fatal("unchanged archive needlessly rewrote the Go cache input")
	}
}

func TestMissingArchivePreservesPreviousFingerprint(t *testing.T) {
	directory := t.TempDir()
	output := filepath.Join(directory, "native_fingerprint.go")
	previous := []byte("previous successful fingerprint")
	if err := os.WriteFile(output, previous, 0600); err != nil {
		t.Fatal(err)
	}
	if err := writeFingerprint(filepath.Join(directory, "missing.a"), output); err == nil {
		t.Fatal("missing archive succeeded")
	}
	actual, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, previous) {
		t.Fatal("failed native build replaced the previous Go cache input")
	}
}
