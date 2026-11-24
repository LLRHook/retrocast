package storage

import (
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"net/http"
	"net/url"
	"testing"
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
	for _, ct := range []string{"image/png", "application/pdf", "text/plain"} {
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
			if ct == "image/png" {
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
