package storage

import (
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type noNetwork struct{ calls int }

func (n *noNetwork) RoundTrip(*http.Request) (*http.Response, error) {
	n.calls++
	return nil, fmt.Errorf("network forbidden")
}
func TestPublicPresignOffline(t *testing.T) {
	transport := &noNetwork{}
	signer, err := minio.New("unreachable.invalid", &minio.Options{Secure: true, Region: "us-east-1", Creds: credentials.NewStaticV4("retrocast", "changeme", ""), Transport: transport})
	if err != nil {
		t.Fatal(err)
	}
	m := &MinIOClient{signer: signer, bucket: "retrocast"}
	for _, ct := range []string{"image/png", "image/jpeg", "image/gif", "image/webp", "application/pdf", "text/plain"} {
		for i := 0; i < 50; i++ {
			raw, err := m.GetURL("attachments/1/2/a file", ct)
			if err != nil {
				t.Fatal(err)
			}
			u, _ := url.Parse(raw)
			q := u.Query()
			if u.Scheme != "https" || u.Host != "unreachable.invalid" || u.Path != "/retrocast/attachments/1/2/a file" || q.Get("X-Amz-Expires") != "900" || q.Get("X-Amz-Signature") == "" {
				t.Fatalf("invalid URL fields")
			}
			want := "attachment"
			if strings.HasPrefix(ct, "image/") {
				want = "inline"
			}
			if q.Get("response-content-disposition") != want {
				t.Fatal("disposition")
			}
		}
	}
	if transport.calls != 0 {
		t.Fatalf("network calls %d", transport.calls)
	}
	_, err = m.signer.PresignedGetObject(context.Background(), "retrocast", "", 0, nil)
	if err == nil {
		t.Fatal("invalid input accepted")
	}
}

func TestConstructorDomainAndOfflinePresign(t *testing.T) {
	var calls atomic.Int32
	internal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(http.StatusOK) }))
	defer internal.Close()
	endpoint := strings.TrimPrefix(internal.URL, "http://")
	for _, domain := range []string{"", "https://example.com", "example.com:443", "example.com/path", "user@example.com", "example.com?x=1", "example.com#x", "example.com bad"} {
		t.Run("refuse_"+domain, func(t *testing.T) {
			before := calls.Load()
			m, err := NewMinIOClient(endpoint, "retrocast", "changeme", "retrocast", domain, "us-east-1", "retrocast-signer", "signer-changeme")
			if err == nil || m != nil {
				t.Fatal("invalid DOMAIN accepted")
			}
			if calls.Load() != before {
				t.Fatal("invalid domain reached storage")
			}
		})
	}
	m, err := NewMinIOClient(endpoint, "retrocast", "changeme", "retrocast", "UNREACHABLE.INVALID", "us-east-1", "retrocast-signer", "signer-changeme")
	if err != nil {
		t.Fatal(err)
	}
	before := calls.Load()
	done := make(chan error, 1)
	go func() {
		raw, err := m.GetURL("attachments/test", "image/png")
		if err == nil {
			u, _ := url.Parse(raw)
			if u.Scheme != "https" || u.Host != "unreachable.invalid" || u.Query().Get("X-Amz-Expires") != "900" || !strings.HasPrefix(u.Query().Get("X-Amz-Credential"), "retrocast-signer/") {
				err = fmt.Errorf("incorrect production signer host or expiry")
			}
		}
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("production presign attempted public network lookup")
	}
	if calls.Load() != before {
		t.Fatal("production presign called internal storage")
	}
}

func TestSignerCredentialsRequired(t *testing.T) {
	for _, pair := range [][2]string{{"", ""}, {"signer", ""}, {"", "fixture"}, {"retrocast", "fixture"}} {
		m, err := NewMinIOClient("unreachable.invalid", "retrocast", "changeme", "retrocast", "localhost", "us-east-1", pair[0], pair[1])
		if err == nil || m != nil {
			t.Fatal("missing or root signer credentials accepted")
		}
	}
}
