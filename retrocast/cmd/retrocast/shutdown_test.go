package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestShutdownServerDeadline(t *testing.T) {
	var captured context.Context
	sentinel := errors.New("shutdown failed")
	start := time.Now()
	err := shutdownServer(func(ctx context.Context) error {
		captured = ctx
		deadline, ok := ctx.Deadline()
		if !ok || deadline.Before(start.Add(10*time.Second)) || deadline.After(time.Now().Add(10*time.Second)) {
			t.Fatal("shutdown must have a ten second deadline")
		}
		if ctx.Err() != nil {
			t.Fatal("shutdown context already cancelled")
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatal("shutdown error lost")
	}
	if captured.Err() != context.Canceled {
		t.Fatal("deadline resources not released")
	}
}
