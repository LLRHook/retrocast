package service

import "testing"

func TestUploadContentAllowlist(t *testing.T) {
	for _, ct := range []string{"image/jpeg", "image/png", "image/gif", "image/webp", "application/pdf", "text/plain"} {
		if !isAllowedContentType(ct) {
			t.Errorf("refused %s", ct)
		}
	}
	for _, ct := range []string{"image/svg+xml", "image/unknown", "application/octet-stream", ""} {
		if isAllowedContentType(ct) {
			t.Errorf("accepted %s", ct)
		}
	}
}
