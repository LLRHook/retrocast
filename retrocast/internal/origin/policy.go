// Package origin implements the browser origin policy shared by REST and gateway.
// Forwarding headers are trusted only behind the bundled, unpublished Caddy hop.
package origin

import (
	"net/http"
	"net/url"
	"strings"
)

type Policy struct{ Allowed string }

func canonical(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || u.Opaque != "" || u.RawQuery != "" || u.Fragment != "" || u.Path != "" || u.Hostname() == "" {
		return ""
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return ""
	}
	port := u.Port()
	if port == "" {
		if scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	return scheme + "://" + strings.ToLower(u.Hostname()) + ":" + port
}

func (p Policy) Check(r *http.Request) bool {
	raw := r.Header.Get("Origin")
	if raw == "" {
		return true
	}
	candidate := canonical(raw)
	if candidate == "" {
		return false
	}
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		scheme = "http"
		if r.TLS != nil {
			scheme = "https"
		}
	}
	if candidate == canonical(scheme+"://"+r.Host) {
		return true
	}
	for _, allowed := range strings.Split(p.Allowed, ",") {
		if c := canonical(strings.TrimSpace(allowed)); c != "" && c == candidate {
			return true
		}
	}
	return false
}
