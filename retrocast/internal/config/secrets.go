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
	placeholders := map[string]bool{"change-me-in-production": true, "changeme": true, "devsecret": true, "password": true, "retrocast": true, "minioadmin": true, "signer-changeme": true}
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

func credentialPasswords(values map[string]string, databaseURL, redisURL string) {
	for name, raw := range map[string]string{"DATABASE_URL": databaseURL, "REDIS_URL": redisURL} {
		u, err := url.Parse(raw)
		if err == nil && u.User != nil {
			password, _ := u.User.Password()
			values[name+" password"] = password
		}
	}
}

// CheckEnvironment validates services that consume .env directly, before they run.
// It does not connect to a database or any other dependency.
func CheckEnvironment() error {
	values := map[string]string{}
	for _, name := range []string{"JWT_SECRET", "MINIO_SECRET_KEY", "MINIO_SIGNER_SECRET_KEY", "LIVEKIT_API_SECRET", "POSTGRES_PASSWORD", "MINIO_ROOT_PASSWORD", "GRAFANA_ADMIN_PASSWORD"} {
		values[name] = os.Getenv(name)
	}
	credentialPasswords(values, os.Getenv("DATABASE_URL"), os.Getenv("REDIS_URL"))
	if err := ValidateSecrets(values); err != nil {
		return err
	}
	if os.Getenv("MINIO_ROOT_PASSWORD") != os.Getenv("MINIO_SECRET_KEY") {
		return fmt.Errorf("MINIO_ROOT_PASSWORD and MINIO_SECRET_KEY must match")
	}
	return nil
}
