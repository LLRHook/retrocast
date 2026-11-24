package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/victorivanov/retrocast/internal/gateway"
	"github.com/victorivanov/retrocast/internal/service"
	"testing"
)

func TestGatewayPermissionErrorClassification(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		hidden bool
	}{
		{"allowed", nil, false}, {"forbidden", service.Forbidden("FORBIDDEN", "hidden"), true}, {"not-found", service.NotFound("NOT_FOUND", "missing"), true}, {"wrapped-forbidden", fmt.Errorf("wrapped: %w", service.ErrForbidden), true}, {"internal", service.Internal("INTERNAL", "lookup failed"), false}, {"deadline", context.DeadlineExceeded, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := gatewayPermissionResult(tc.err)
			if errors.Is(got, gateway.ErrChannelHidden) != tc.hidden {
				t.Fatal("wrong definitive classification")
			}
			if !tc.hidden && got != tc.err {
				t.Fatal("infrastructure error not preserved")
			}
		})
	}
}
