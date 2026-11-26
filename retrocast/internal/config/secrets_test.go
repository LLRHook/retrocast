package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecretStartupPolicy(t *testing.T) {
	keys := []string{"JWT_SECRET", "MINIO_SECRET_KEY", "MINIO_SIGNER_SECRET_KEY", "LIVEKIT_API_SECRET"}
	strong := freshSecret(t)
	for _, key := range keys {
		for _, bad := range []string{"", "short", "change-me-in-production", "changeme", "devsecret", "signer-changeme"} {
			t.Run(key+fmt.Sprint(len(bad)), func(t *testing.T) {
				for _, k := range keys {
					t.Setenv(k, strong)
				}
				t.Setenv("RETROCAST_CONFIG", t.TempDir()+"/absent")
				t.Setenv("DATABASE_URL", "postgres://user:"+strong+"@localhost/db")
				t.Setenv(key, bad)
				defer func() {
					r := recover()
					if r == nil || !strings.Contains(fmt.Sprint(r), key) {
						t.Fatal("startup did not reject unsafe secret by name")
					}
					if bad != "" && strings.Contains(fmt.Sprint(r), bad) {
						t.Fatal("error leaked a credential")
					}
				}()
				Load()
			})
		}
	}
	for _, key := range keys {
		t.Setenv(key, strong)
	}
	t.Setenv("DATABASE_URL", "postgres://user:"+strong+"@localhost/db")
	if Load().JWTSecret != strong {
		t.Fatal("strong credentials refused")
	}
	// The contract counts characters, not UTF-8 bytes.
	if err := ValidateSecrets(map[string]string{"fixture": strings.Repeat("?", 31)}); err == nil {
		t.Fatal("31 multibyte characters accepted")
	}
	if err := ValidateSecrets(map[string]string{"fixture": randomUnicode(t)}); err != nil {
		t.Fatal("32 multibyte characters refused")
	}
}

func TestComposeSecretPolicy(t *testing.T) {
	keys := []string{"JWT_SECRET", "MINIO_SECRET_KEY", "MINIO_SIGNER_SECRET_KEY", "LIVEKIT_API_SECRET", "POSTGRES_PASSWORD", "MINIO_ROOT_PASSWORD", "GRAFANA_ADMIN_PASSWORD"}
	strong := freshSecret(t)
	for _, key := range keys {
		t.Setenv(key, strong)
	}
	t.Setenv("DATABASE_URL", "postgres://user:"+strong+"@db/database")
	t.Setenv("REDIS_URL", "redis://redis:6379")
	if err := CheckEnvironment(); err != nil {
		t.Fatal(err)
	}
	for _, key := range keys {
		for _, bad := range []string{"", "short", "retrocast"} {
			t.Run(key+fmt.Sprint(len(bad)), func(t *testing.T) {
				t.Setenv(key, bad)
				err := CheckEnvironment()
				if err == nil || !strings.Contains(err.Error(), key) {
					t.Fatal("unsafe service secret accepted")
				}
			})
		}
	}
	for _, key := range []string{"DATABASE_URL", "REDIS_URL"} {
		t.Run(key, func(t *testing.T) {
			t.Setenv(key, "postgres://user:short@db/db")
			if err := CheckEnvironment(); err == nil || !strings.Contains(err.Error(), key) {
				t.Fatal("unsafe URL password accepted")
			}
		})
	}
}

func TestInitEnvironmentFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	cmd := exec.Command("bash", "../../scripts/init-env.sh", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generator failed: %v (%s)", err, out)
	}
	values, err := parseConfigFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"JWT_SECRET", "MINIO_SECRET_KEY", "MINIO_SIGNER_SECRET_KEY", "LIVEKIT_API_SECRET", "POSTGRES_PASSWORD", "MINIO_ROOT_PASSWORD", "GRAFANA_ADMIN_PASSWORD"} {
		if err = ValidateSecrets(map[string]string{key: values[key]}); err != nil {
			t.Fatal(err)
		}
	}
	if values["MINIO_ROOT_PASSWORD"] != values["MINIO_SECRET_KEY"] {
		t.Fatal("root/write mismatch")
	}
	if len(values["MINIO_ACCESS_KEY"]) != 20 || values["MINIO_ACCESS_KEY"] == values["MINIO_SIGNER_ACCESS_KEY"] {
		t.Fatal("invalid dedicated IDs")
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatal("environment is not private")
	}
	before, _ := os.ReadFile(path)
	if err = exec.Command("bash", "../../scripts/init-env.sh", path).Run(); err == nil {
		t.Fatal("existing file overwritten")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("existing environment changed")
	}
}

func TestInitEnvironmentWindowsWarning(t *testing.T) {
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "uname"), []byte("#!/bin/sh\necho MSYS_NT-10.0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	cmd := exec.Command("bash", "../../scripts/init-env.sh", filepath.Join(t.TempDir(), ".env"))
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "NTFS") || strings.Contains(string(out), "Created private") {
		t.Fatal("missing Windows permissions warning")
	}
}
