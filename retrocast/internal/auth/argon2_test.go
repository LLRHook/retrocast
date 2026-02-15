package auth

import (
	"strings"
	"testing"
)

func TestHashAndVerifyCorrectPassword(t *testing.T) {
	password := "correct-horse-battery-staple"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error: %v", err)
	}

	if !strings.HasPrefix(hash, "$argon2id$v=19$") {
		t.Errorf("hash has unexpected format: %s", hash)
	}

	ok, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("VerifyPassword() error: %v", err)
	}
	if !ok {
		t.Error("VerifyPassword() returned false for correct password")
	}
}

func TestRejectWrongPassword(t *testing.T) {
	hash, err := HashPassword("real-password")
	if err != nil {
		t.Fatalf("HashPassword() error: %v", err)
	}

	ok, err := VerifyPassword("wrong-password", hash)
	if err != nil {
		t.Fatalf("VerifyPassword() error: %v", err)
	}
	if ok {
		t.Error("VerifyPassword() returned true for wrong password")
	}
}

func TestDifferentPasswordsDifferentHashes(t *testing.T) {
	h1, err := HashPassword("password-one")
	if err != nil {
		t.Fatalf("HashPassword() error: %v", err)
	}
	h2, err := HashPassword("password-two")
	if err != nil {
		t.Fatalf("HashPassword() error: %v", err)
	}
	if h1 == h2 {
		t.Error("different passwords produced identical hashes")
	}
}

func TestSamePasswordDifferentSalts(t *testing.T) {
	password := "same-password"

	h1, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error: %v", err)
	}
	h2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error: %v", err)
	}
	if h1 == h2 {
		t.Error("same password produced identical hashes (salts should differ)")
	}

	// Both should still verify
	ok1, _ := VerifyPassword(password, h1)
	ok2, _ := VerifyPassword(password, h2)
	if !ok1 || !ok2 {
		t.Error("same password should verify against both hashes")
	}
}
