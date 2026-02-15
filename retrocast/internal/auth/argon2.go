package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    = 1
	argonMemory  = 64 * 1024 // 64 MB
	argonThreads = 4
	argonKeyLen  = 32
	argonSaltLen = 16
)

// HashPassword returns an encoded Argon2id hash string.
// Format: $argon2id$v=19$m=65536,t=1,p=4$<base64-salt>$<base64-hash>
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generating salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argonMemory,
		argonTime,
		argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)
	return encoded, nil
}

// VerifyPassword checks a password against an encoded Argon2id hash.
func VerifyPassword(password, encodedHash string) (bool, error) {
	salt, hash, memory, time, threads, keyLen, err := decodeHash(encodedHash)
	if err != nil {
		return false, err
	}

	candidate := argon2.IDKey([]byte(password), salt, time, memory, threads, keyLen)

	if subtle.ConstantTimeCompare(hash, candidate) == 1 {
		return true, nil
	}
	return false, nil
}

func decodeHash(encodedHash string) (salt, hash []byte, memory, time uint32, threads uint8, keyLen uint32, err error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return nil, nil, 0, 0, 0, 0, fmt.Errorf("invalid hash format: expected 6 parts, got %d", len(parts))
	}

	if parts[1] != "argon2id" {
		return nil, nil, 0, 0, 0, 0, fmt.Errorf("unsupported algorithm: %s", parts[1])
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return nil, nil, 0, 0, 0, 0, fmt.Errorf("parsing version: %w", err)
	}
	if version != argon2.Version {
		return nil, nil, 0, 0, 0, 0, fmt.Errorf("unsupported argon2 version: %d", version)
	}

	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return nil, nil, 0, 0, 0, 0, fmt.Errorf("parsing parameters: %w", err)
	}

	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, 0, 0, 0, 0, fmt.Errorf("decoding salt: %w", err)
	}

	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, 0, 0, 0, 0, fmt.Errorf("decoding hash: %w", err)
	}
	keyLen = uint32(len(hash))

	return salt, hash, memory, time, threads, keyLen, nil
}
