package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
)

// ---------------------------------------------------------------------------
// Mock storage
// ---------------------------------------------------------------------------

type mockStorage struct {
	UploadFn  func(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error
	GetURLFn  func(key string) string
	DeleteFn  func(ctx context.Context, key string) error
}

func (m *mockStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	if m.UploadFn != nil {
		return m.UploadFn(ctx, key, reader, size, contentType)
	}
	return nil
}

func (m *mockStorage) GetURL(key string) string {
	if m.GetURLFn != nil {
		return m.GetURLFn(key)
	}
	return "http://localhost:9000/retrocast/" + key
}

func (m *mockStorage) Delete(ctx context.Context, key string) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, key)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock attachment repo
// ---------------------------------------------------------------------------

type mockAttachmentRepo struct {
	CreateFn         func(ctx context.Context, a *models.Attachment) error
	GetByMessageIDFn func(ctx context.Context, messageID int64) ([]models.Attachment, error)
	DeleteFn         func(ctx context.Context, id int64) error
}

func (m *mockAttachmentRepo) Create(ctx context.Context, a *models.Attachment) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, a)
	}
	return nil
}

func (m *mockAttachmentRepo) GetByMessageID(ctx context.Context, messageID int64) ([]models.Attachment, error) {
	if m.GetByMessageIDFn != nil {
		return m.GetByMessageIDFn(ctx, messageID)
	}
	return nil, nil
}

func (m *mockAttachmentRepo) Delete(ctx context.Context, id int64) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newUploadHandler(
	att *mockAttachmentRepo,
	chs *mockChannelRepo,
	mems *mockMemberRepo,
	roles *mockRoleRepo,
	guilds *mockGuildRepo,
	overrides *mockChannelOverrideRepo,
	store *mockStorage,
) *UploadHandler {
	return NewUploadHandler(att, chs, mems, roles, guilds, overrides, testSnowflake(), store)
}

func newMultipartContext(t *testing.T, filename, contentType string, fileContent []byte) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	mh := make(map[string][]string)
	mh["Content-Disposition"] = []string{fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename)}
	mh["Content-Type"] = []string{contentType}
	part, err := writer.CreatePart(mh)
	if err != nil {
		t.Fatalf("create part: %v", err)
	}
	if _, err := part.Write(fileContent); err != nil {
		t.Fatalf("write file content: %v", err)
	}
	writer.Close()

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/channels/2000/attachments", body)
	req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestUpload_Success(t *testing.T) {
	guilds, members, roles, overrides := permMocks(permissions.PermAttachFiles | permissions.PermViewChannel)
	channels := channelMock()
	store := &mockStorage{}
	att := &mockAttachmentRepo{}

	h := newUploadHandler(att, channels, members, roles, guilds, overrides, store)

	c, rec := newMultipartContext(t, "photo.png", "image/png", []byte("fake png data"))
	c.SetParamNames("id")
	c.SetParamValues("2000")
	setAuthUser(c, testUserID)

	err := h.Upload(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var result models.Attachment
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if result.Filename != "photo.png" {
		t.Fatalf("expected filename 'photo.png', got %q", result.Filename)
	}
	if result.URL == "" {
		t.Fatal("expected non-empty URL")
	}
}

func TestUpload_FileTooLarge(t *testing.T) {
	guilds, members, roles, overrides := permMocks(permissions.PermAttachFiles | permissions.PermViewChannel)
	channels := channelMock()
	store := &mockStorage{}
	att := &mockAttachmentRepo{}

	h := newUploadHandler(att, channels, members, roles, guilds, overrides, store)

	// Create a file exceeding 10 MB.
	largeContent := make([]byte, 11<<20)

	c, rec := newMultipartContext(t, "big.png", "image/png", largeContent)
	c.SetParamNames("id")
	c.SetParamValues("2000")
	setAuthUser(c, testUserID)

	_ = h.Upload(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error: %v", err)
	}
	if errResp.Error.Code != "FILE_TOO_LARGE" {
		t.Fatalf("expected FILE_TOO_LARGE, got %q", errResp.Error.Code)
	}
}

func TestUpload_InvalidContentType(t *testing.T) {
	guilds, members, roles, overrides := permMocks(permissions.PermAttachFiles | permissions.PermViewChannel)
	channels := channelMock()
	store := &mockStorage{}
	att := &mockAttachmentRepo{}

	h := newUploadHandler(att, channels, members, roles, guilds, overrides, store)

	// Use a disallowed content type — we need to set it via the multipart header.
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	mh := make(map[string][]string)
	mh["Content-Disposition"] = []string{`form-data; name="file"; filename="evil.exe"`}
	mh["Content-Type"] = []string{"application/octet-stream"}
	part, err := writer.CreatePart(mh)
	if err != nil {
		t.Fatalf("create part: %v", err)
	}
	part.Write([]byte("evil binary data"))
	writer.Close()

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/channels/2000/attachments", body)
	req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("2000")
	setAuthUser(c, testUserID)

	_ = h.Upload(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error: %v", err)
	}
	if errResp.Error.Code != "INVALID_CONTENT_TYPE" {
		t.Fatalf("expected INVALID_CONTENT_TYPE, got %q", errResp.Error.Code)
	}
}

func TestUpload_NoPermission(t *testing.T) {
	// Has ViewChannel but NOT AttachFiles.
	guilds, members, roles, overrides := permMocks(permissions.PermViewChannel)
	channels := channelMock()
	store := &mockStorage{}
	att := &mockAttachmentRepo{}

	h := newUploadHandler(att, channels, members, roles, guilds, overrides, store)

	c, rec := newMultipartContext(t, "photo.png", "image/png", []byte("data"))
	c.SetParamNames("id")
	c.SetParamValues("2000")
	setAuthUser(c, testUserID)

	_ = h.Upload(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUpload_MissingFile(t *testing.T) {
	guilds, members, roles, overrides := permMocks(permissions.PermAttachFiles | permissions.PermViewChannel)
	channels := channelMock()
	store := &mockStorage{}
	att := &mockAttachmentRepo{}

	h := newUploadHandler(att, channels, members, roles, guilds, overrides, store)

	// Send a request with no file field.
	c, rec := newTestContext(http.MethodPost, "/api/v1/channels/2000/attachments", strings.NewReader(""))
	c.SetParamNames("id")
	c.SetParamValues("2000")
	setAuthUser(c, testUserID)

	_ = h.Upload(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUpload_ChannelNotFound(t *testing.T) {
	guilds, members, roles, overrides := permMocks(permissions.PermAttachFiles | permissions.PermViewChannel)
	channels := &mockChannelRepo{
		GetByIDFn: func(_ context.Context, _ int64) (*models.Channel, error) {
			return nil, nil
		},
	}
	store := &mockStorage{}
	att := &mockAttachmentRepo{}

	h := newUploadHandler(att, channels, members, roles, guilds, overrides, store)

	c, rec := newMultipartContext(t, "photo.png", "image/png", []byte("data"))
	c.SetParamNames("id")
	c.SetParamValues("9999")
	setAuthUser(c, testUserID)

	_ = h.Upload(c)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// Suppress "unused" warning for time import — used by permMocks in testutil_test.go.
var _ = time.Now
