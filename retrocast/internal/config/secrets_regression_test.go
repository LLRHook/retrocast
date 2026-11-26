package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
)

func freshSecret(t *testing.T) string {
	t.Helper()
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatal(err)
	}
	return "s" + hex.EncodeToString(b[:])
}
func strongEnvironment(t *testing.T) string {
	t.Helper()
	root := freshSecret(t)
	for _, name := range []string{"JWT_SECRET", "MINIO_SECRET_KEY", "MINIO_SIGNER_SECRET_KEY", "LIVEKIT_API_SECRET", "POSTGRES_PASSWORD", "MINIO_ROOT_PASSWORD", "GRAFANA_ADMIN_PASSWORD"} {
		t.Setenv(name, freshSecret(t))
	}
	t.Setenv("MINIO_SECRET_KEY", root)
	t.Setenv("MINIO_ROOT_PASSWORD", root)
	t.Setenv("DATABASE_URL", "postgres://retrocast:"+freshSecret(t)+"@localhost:1/retrocast?connect_timeout=1")
	t.Setenv("REDIS_URL", "redis://localhost:1")
	return root
}
func TestRootWriteSecretMismatch(t *testing.T) {
	strongEnvironment(t)
	t.Setenv("MINIO_SECRET_KEY", freshSecret(t))
	err := CheckEnvironment()
	if err == nil || !strings.Contains(err.Error(), "MINIO_ROOT_PASSWORD") || !strings.Contains(err.Error(), "MINIO_SECRET_KEY") {
		t.Fatal("mismatched strong root/write secrets accepted")
	}
}
func TestLoadRejectsCredentialPasswords(t *testing.T) {
	for _, name := range []string{"DATABASE_URL", "REDIS_URL"} {
		for _, raw := range []string{"postgres://user:short@localhost/db", "postgres://user@localhost/db", "host=localhost password=short"} {
			t.Run(name+fmt.Sprint(len(raw)), func(t *testing.T) {
				strongEnvironment(t)
				t.Setenv(name, raw)
				defer func() {
					v := recover()
					if v == nil || !strings.Contains(fmt.Sprint(v), name) {
						t.Fatal("Load accepted invalid or weak URL credentials")
					}
				}()
				Load()
			})
		}
	}
}
func TestSecretPaddingAndCommittedValues(t *testing.T) {
	if err := ValidateSecrets(map[string]string{"JWT_SECRET": strings.Repeat(" ", 32)}); err == nil {
		t.Fatal("whitespace padding accepted")
	}
	for _, value := range []string{strings.Repeat("test-only-", 4), strings.Repeat("compose-test-", 3), "retrocast-ci-postgres-test-only-2026", "storage-root-test-only-secret-123456", "storage-signer-test-only-secret-1234"} {
		if err := ValidateSecrets(map[string]string{"JWT_SECRET": value}); err == nil {
			t.Fatal("committed test secret accepted for startup")
		}
	}
}

func randomUnicode(t *testing.T) string {
	var b strings.Builder
	for _, c := range freshSecret(t)[1:] {
		b.WriteRune(0x4e00 + c)
	}
	return b.String()
}

func TestMinIOHostCompatibility(t *testing.T) {
	for _, field := range []string{"MINIO_ACCESS_KEY", "MINIO_ROOT_PASSWORD"} {
		bad := []string{":", "\n"}
		if field == "MINIO_ACCESS_KEY" {
			bad = append(bad, "\r", "\v", "\f", "\x7f", "\x01")
		}
		for _, char := range bad {
			t.Run(field+fmt.Sprintf("-%x", char), func(t *testing.T) {
				root := strongEnvironment(t)
				if field == "MINIO_ACCESS_KEY" {
					t.Setenv(field, "root"+char+"key")
				} else {
					t.Setenv(field, root+char)
					t.Setenv("MINIO_SECRET_KEY", root+char)
				}
				err := CheckEnvironment()
				if err == nil || !strings.Contains(err.Error(), field) {
					t.Fatal("incompatible MC_HOST credential accepted")
				}
				defer func() {
					v := recover()
					if v == nil {
						t.Fatal("API accepted incompatible MC_HOST credentials")
					}
				}()
				Load()
			})
		}
	}
	for _, char := range []string{"@", "/", "#", "?", "%", " ", "\t"} {
		t.Run("supported"+fmt.Sprintf("%x", char), func(t *testing.T) {
			root := strongEnvironment(t)
			t.Setenv("MINIO_ACCESS_KEY", "root"+char+"key")
			t.Setenv("MINIO_SECRET_KEY", root+char)
			t.Setenv("MINIO_ROOT_PASSWORD", root+char)
			if err := CheckEnvironment(); err != nil {
				t.Fatal("supported MC_HOST punctuation refused:", err)
			}
		})
	}
}

func TestPaddedCommittedSecret(t *testing.T) {
	if ValidateSecrets(map[string]string{"LIVEKIT_API_SECRET": "retrocast-ci-postgres-test-only-2026 "}) == nil {
		t.Fatal("padded committed credential accepted")
	}
}
func TestQueryCredentialRefused(t *testing.T) {
	for _, name := range []string{"DATABASE_URL", "REDIS_URL"} {
		t.Run(name, func(t *testing.T) {
			strongEnvironment(t)
			raw := "postgres://localhost/db?password=short"
			if name == "REDIS_URL" {
				raw = "redis://localhost?password=short"
			}
			t.Setenv(name, raw)
			defer func() {
				v := recover()
				if v == nil || !strings.Contains(fmt.Sprint(v), name) {
					t.Fatal("query credential accepted")
				}
			}()
			Load()
		})
	}
}
func TestPGPasswordValidated(t *testing.T) {
	strongEnvironment(t)
	t.Setenv("PGPASSWORD", "short")
	defer func() {
		v := recover()
		if v == nil || !strings.Contains(fmt.Sprint(v), "PGPASSWORD") {
			t.Fatal("PGPASSWORD accepted")
		}
	}()
	Load()
}
