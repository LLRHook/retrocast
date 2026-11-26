package storage

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCaddySignedDownloads(t *testing.T) {
	if os.Getenv("L34_STACK_TEST") != "1" {
		t.Skip("owned Caddy stack required")
	}
	m, err := NewMinIOClient("minio:9000", os.Getenv("MINIO_ACCESS_KEY"), os.Getenv("MINIO_SECRET_KEY"), "retrocast", "localhost", "us-east-1", os.Getenv("MINIO_SIGNER_ACCESS_KEY"), os.Getenv("MINIO_SIGNER_SECRET_KEY"))
	if err != nil {
		t.Fatal(err)
	}
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, "caddy:443")
	}}
	client := &http.Client{Transport: tr}
	defer tr.CloseIdleConnections()
	get := func(method, raw string, want int) http.Header {
		t.Helper()
		req, _ := http.NewRequest(method, raw, nil)
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = res.Body.Close() }()
		if res.StatusCode != want {
			t.Fatalf("%s status %d want %d", method, res.StatusCode, want)
		}
		_, _ = io.Copy(io.Discard, res.Body)
		return res.Header
	}
	for _, ct := range []string{"image/png", "application/pdf"} {
		key := "attachments/l34/test"
		if err := m.Upload(context.Background(), key, bytes.NewReader([]byte("fixture")), 7, ct); err != nil {
			t.Fatal(err)
		}
		raw, err := m.GetURL(key, ct)
		if err != nil {
			t.Fatal(err)
		}
		h := get("GET", raw, 200)
		if h.Get("Content-Security-Policy") != "sandbox allow-downloads" || !strings.Contains(h.Get("X-Content-Type-Options"), "nosniff") {
			t.Fatal("missing content isolation")
		}
		want := "attachment"
		if ct == "image/png" {
			want = "inline"
		}
		if h.Get("Content-Disposition") != want {
			t.Fatal("disposition")
		}
		u, _ := url.Parse(raw)
		q := u.Query()
		q.Set("X-Amz-Signature", strings.Repeat("0", 64))
		u.RawQuery = q.Encode()
		get("GET", u.String(), 403)
		get("GET", "https://localhost/retrocast/"+key, 403)
		get("PUT", raw, 405)
		get("HEAD", raw, 405)
		short, err := m.signer.PresignedGetObject(context.Background(), m.bucket, key, time.Second, nil)
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(2100 * time.Millisecond)
		get("GET", short.String(), 403)
		_ = m.Delete(context.Background(), key)
	}
	get("GET", "https://localhost/retrocast/", 403)
	// Other bucket and console paths fall through to the SPA, never MinIO.
	for _, path := range []string{"/other-bucket/", "/minio/", "/retrocast"} {
		req, _ := http.NewRequest("GET", "https://localhost"+path, nil)
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(res.Body)
		_ = res.Body.Close()
		if !bytes.Contains(body, []byte("<html")) || bytes.Contains(body, []byte("ListBucketResult")) {
			t.Fatal("unexpected exposed MinIO path")
		}
	}
}
