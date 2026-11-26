package config

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"unicode/utf8"
)

// ValidateSecrets applies the same startup policy to server and Compose secrets.
// Errors contain variable names only, never credential values or URLs.
func ValidateSecrets(values map[string]string) error {
	placeholders := map[string]bool{"change-me-in-production": true, "changeme": true, "devsecret": true, "password": true, "retrocast": true, "minioadmin": true, "signer-changeme": true, strings.Repeat("test-only-", 4): true, strings.Repeat("compose-test-", 3): true, "retrocast-ci-postgres-test-only-2026": true, "storage-root-test-only-secret-123456": true, "storage-signer-test-only-secret-1234": true, strings.Repeat("?", 32): true, strings.Repeat("\u754c", 32): true}
	var invalid []string
	for name, value := range values {
		if utf8.RuneCountInString(strings.TrimSpace(value)) < 32 || placeholders[value] {
			invalid = append(invalid, name)
		}
	}
	if len(invalid) == 0 {
		return nil
	}
	sort.Strings(invalid)
	return fmt.Errorf("unsafe secret configuration: %s; secrets require at least 32 characters and must not be committed placeholders; for a new installation run bash scripts/init-env.sh; existing deployments must rotate credentials as documented in docs/OPERATIONS.md", strings.Join(invalid, ", "))
}

func credentialPasswords(values map[string]string, databaseURL, redisURL string) error {
	for name, raw := range map[string]string{"DATABASE_URL": databaseURL, "REDIS_URL": redisURL} {
		if raw == "" {
			continue
		}
		u, err := url.Parse(raw)
		validScheme := err == nil && ((name == "DATABASE_URL" && (u.Scheme == "postgres" || u.Scheme == "postgresql")) || (name == "REDIS_URL" && (u.Scheme == "redis" || u.Scheme == "rediss")))
		if !validScheme || u.Opaque != "" {
			return fmt.Errorf("%s must be a valid connection URL; keyword DSNs are not supported", name)
		}
		if u.User != nil {
			password, _ := u.User.Password()
			values[name+" password"] = password
		}
	}
	return nil
}

// The pinned mc parses MC_HOST literally. These characters cannot authenticate
// through that parser or HTTP header encoding. Supported punctuation stays valid.
func validateMinIOHost(accessKey string, passwords map[string]string) error {
	var invalid []string
	for _, c := range accessKey {
		if c == ':' || c < 32 && c != '\t' || c == 127 {
			invalid = append(invalid, "MINIO_ACCESS_KEY")
			break
		}
	}
	for name, value := range passwords {
		if strings.ContainsAny(value, ":\n") {
			invalid = append(invalid, name)
		}
	}
	if len(invalid) == 0 {
		return nil
	}
	sort.Strings(invalid)
	return fmt.Errorf("incompatible MinIO credentials: %s; rotate with init-env-style values as documented in docs/OPERATIONS.md", strings.Join(invalid, ", "))
}

// CheckEnvironment validates services that consume .env directly, before they run.
// It does not connect to a database or any other dependency.
func CheckEnvironment() error {
	values := map[string]string{}
	for _, name := range []string{"JWT_SECRET", "MINIO_SECRET_KEY", "MINIO_SIGNER_SECRET_KEY", "LIVEKIT_API_SECRET", "POSTGRES_PASSWORD", "MINIO_ROOT_PASSWORD", "GRAFANA_ADMIN_PASSWORD"} {
		values[name] = os.Getenv(name)
	}
	if err := credentialPasswords(values, os.Getenv("DATABASE_URL"), os.Getenv("REDIS_URL")); err != nil {
		return err
	}
	if err := ValidateSecrets(values); err != nil {
		return err
	}
	if os.Getenv("MINIO_ROOT_PASSWORD") != os.Getenv("MINIO_SECRET_KEY") {
		return fmt.Errorf("MINIO_ROOT_PASSWORD and MINIO_SECRET_KEY must match")
	}
	return validateMinIOHost(os.Getenv("MINIO_ACCESS_KEY"), map[string]string{"MINIO_ROOT_PASSWORD": os.Getenv("MINIO_ROOT_PASSWORD"), "MINIO_SECRET_KEY": os.Getenv("MINIO_SECRET_KEY")})
}
