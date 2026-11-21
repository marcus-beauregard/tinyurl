// SPDX-License-Identifier: GPL-2.0-or-later
// Copyright (C) 2026 Marcus Beauregard

package main

import (
	"crypto/rand"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type failingReader struct{}

func (failingReader) Read(p []byte) (int, error) {
	return 0, errors.New("entropy source unavailable")
}

func TestEntropyFailureReturns500AndStoresNoCode(t *testing.T) {
	original := rand.Reader
	rand.Reader = failingReader{}
	defer func() { rand.Reader = original }()

	s := newStore()
	h := handler{s: s}

	req := httptest.NewRequest(http.MethodPost, "http://example.com/", strings.NewReader(`{"url":"https://example.org"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if _, ok := s.get("00000000"); ok {
		t.Fatal("zero code was stored on entropy failure")
	}
	if len(s.m) != 0 {
		t.Fatalf("store has %d entries after entropy failure, want 0", len(s.m))
	}
}

func TestShortCodeSuccess(t *testing.T) {
	code, err := shortCode()
	if err != nil {
		t.Fatalf("shortCode: %v", err)
	}
	if len(code) != 8 {
		t.Fatalf("code = %q, want 8 hex chars", code)
	}
	if code == "00000000" {
		t.Fatal("shortCode returned the zero code")
	}
}
