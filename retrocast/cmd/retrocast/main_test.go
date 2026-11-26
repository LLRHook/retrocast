package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestCheckEnvExitsWithoutInfrastructure(t *testing.T) {
	secret := func() string {
		var b [16]byte
		if _, err := rand.Read(b[:]); err != nil {
			t.Fatal(err)
		}
		return "s" + hex.EncodeToString(b[:])
	}
	root := secret()
	for _, key := range []string{"JWT_SECRET", "MINIO_SIGNER_SECRET_KEY", "LIVEKIT_API_SECRET", "POSTGRES_PASSWORD", "GRAFANA_ADMIN_PASSWORD"} {
		t.Setenv(key, secret())
	}
	t.Setenv("MINIO_SECRET_KEY", root)
	t.Setenv("MINIO_ROOT_PASSWORD", root)
	t.Setenv("DATABASE_URL", "postgres://retrocast:"+secret()+"@127.0.0.1:1/retrocast?connect_timeout=1")
	t.Setenv("REDIS_URL", "redis://127.0.0.1:1")
	binary := filepath.Join(t.TempDir(), "retrocast")
	if err := exec.Command("go", "build", "-o", binary, ".").Run(); err != nil {
		t.Fatal("build startup command:", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "check-env")
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		t.Fatal("check-env must exit successfully without connecting to infrastructure:", err)
	}
}
