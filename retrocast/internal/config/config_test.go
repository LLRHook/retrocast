package config

import (
	"fmt"
	"strings"
	"testing"
)

func TestRegistrationMode(t *testing.T) {
	t.Setenv("DATABASE_URL", "fixture")
	for _, name := range []string{"JWT_SECRET", "MINIO_SECRET_KEY", "MINIO_SIGNER_SECRET_KEY", "LIVEKIT_API_SECRET"} {
		t.Setenv(name, strings.Repeat("test-only-", 4))
	}
	t.Setenv("RETROCAST_CONFIG", t.TempDir()+"/absent")
	for _, mode := range []string{"", "open", "invite", "closed", "public"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("REGISTRATION_MODE", mode)
			if mode == "public" {
				defer func() {
					if r := recover(); r == nil || !strings.Contains(fmt.Sprint(r), "REGISTRATION_MODE") {
						t.Fatal("invalid mode must name key")
					}
				}()
				Load()
				return
			}
			want := mode
			if want == "" {
				want = "invite"
			}
			if got := Load().RegistrationMode; got != want {
				t.Fatalf("got %s want %s", got, want)
			}
		})
	}
}
