package storage

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLeastPrivilegeSignerStack(t *testing.T) {
	if os.Getenv("L48_STACK_TEST") != "1" {
		t.Skip("owned MinIO/Caddy stack required")
	}
	ctx := context.Background()
	key, secret := os.Getenv("MINIO_SIGNER_ACCESS_KEY"), os.Getenv("MINIO_SIGNER_SECRET_KEY")
	if key == "" || secret == "" || key == os.Getenv("MINIO_ACCESS_KEY") {
		t.Fatal("expected dedicated signer inputs")
	}
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, "caddy:443")
	}}
	defer tr.CloseIdleConnections()
	client := &http.Client{Transport: tr}
	request := func(method, path, contentType, token string, body io.Reader) json.RawMessage {
		t.Helper()
		req, err := http.NewRequest(method, "https://localhost"+path, body)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", contentType)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = res.Body.Close() }()
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			t.Fatalf("%s %s status %d", method, path, res.StatusCode)
		}
		var envelope struct {
			Data json.RawMessage `json:"data"`
		}
		if err = json.NewDecoder(res.Body).Decode(&envelope); err != nil {
			t.Fatal(err)
		}
		return envelope.Data
	}
	authData := request("POST", "/api/v1/auth/register", "application/json", "", strings.NewReader(fmt.Sprintf(`{"username":"l48%d","password":"l48-local-fixture-pass"}`, time.Now().UnixNano())))
	var auth struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(authData, &auth); err != nil || auth.AccessToken == "" {
		t.Fatal("auth fixture failed")
	}
	guildData := request("POST", "/api/v1/guilds", "application/json", auth.AccessToken, strings.NewReader(`{"name":"l48-signer"}`))
	var guild struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(guildData, &guild); err != nil {
		t.Fatal(err)
	}
	channelData := request("GET", "/api/v1/guilds/"+guild.ID+"/channels", "", auth.AccessToken, nil)
	var channels []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(channelData, &channels); err != nil || len(channels) == 0 {
		t.Fatal("channel fixture failed")
	}
	fixture := []byte("L48 exact attachment bytes\n")
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "l48.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(fixture)
	_ = writer.Close()
	// CreateFormFile defaults to octet-stream; the upload allowlist requires text/plain.
	payload := bytes.Replace(body.Bytes(), []byte("application/octet-stream"), []byte("text/plain"), 1)
	data := request("POST", "/api/v1/channels/"+channels[0].ID+"/attachments", writer.FormDataContentType(), auth.AccessToken, bytes.NewReader(payload))
	var attachment struct {
		URL string `json:"url"`
	}
	if err = json.Unmarshal(data, &attachment); err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(attachment.URL)
	if err != nil {
		t.Fatal(err)
	}
	id := strings.Split(u.Query().Get("X-Amz-Credential"), "/")[0]
	if id != key {
		t.Fatal("wrong signing account")
	}
	res, err := client.Get(attachment.URL)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if res.StatusCode != 200 || !bytes.Equal(got, fixture) {
		t.Fatalf("signed GET status %d; byte equality %v", res.StatusCode, bytes.Equal(got, fixture))
	}
	t.Logf("Caddy multipart upload 201; signed GET 200 exact %d bytes; credential ID %s (not root)", len(got), id)
	root, err := minio.New("minio:9000", &minio.Options{Creds: credentials.NewStaticV4(os.Getenv("MINIO_ACCESS_KEY"), os.Getenv("MINIO_ROOT_PASSWORD"), ""), Region: "us-east-1"})
	if err != nil {
		t.Fatal(err)
	}
	limited, err := minio.New("minio:9000", &minio.Options{Creds: credentials.NewStaticV4(key, secret, ""), Region: "us-east-1"})
	if err != nil {
		t.Fatal(err)
	}
	deny := func(op string, err error) {
		t.Helper()
		if minio.ToErrorResponse(err).Code != "AccessDenied" {
			t.Fatalf("%s expected AccessDenied, got %v", op, err)
		}
		t.Log(op + " AccessDenied")
	}
	_, err = limited.PutObject(ctx, "retrocast", "l48-denied", bytes.NewReader(fixture), int64(len(fixture)), minio.PutObjectOptions{})
	deny("put", err)
	listed := false
	for obj := range limited.ListObjects(ctx, "retrocast", minio.ListObjectsOptions{}) {
		deny("list", obj.Err)
		listed = true
		break
	}
	if !listed {
		t.Fatal("list returned no denial")
	}
	objectKey := strings.TrimPrefix(u.Path, "/retrocast/")
	deny("delete", limited.RemoveObject(ctx, "retrocast", objectKey, minio.RemoveObjectOptions{}))
	other := "l48-other"
	if err = root.MakeBucket(ctx, other, minio.MakeBucketOptions{}); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = root.RemoveObject(ctx, other, "fixture", minio.RemoveObjectOptions{})
		_ = root.RemoveBucket(ctx, other)
	}()
	if _, err = root.PutObject(ctx, other, "fixture", bytes.NewReader(fixture), int64(len(fixture)), minio.PutObjectOptions{}); err != nil {
		t.Fatal(err)
	}
	obj, err := limited.GetObject(ctx, other, "fixture", minio.GetObjectOptions{})
	if err == nil {
		_, err = io.ReadAll(obj)
		_ = obj.Close()
	}
	deny("other bucket GET", err)
	// The same fixture remains readable after denied delete.
	res, err = client.Get(attachment.URL)
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatal(fmt.Sprint("denied delete changed object: ", res.StatusCode))
	}
}
