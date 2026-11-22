// SPDX-License-Identifier: GPL-2.0-or-later
// Copyright (C) 2026 Marcus Beauregard

package main

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
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

func TestShortLinkUsesRequestScheme(t *testing.T) {
	tests := []struct {
		name       string
		tls        bool
		forwarded  string
		wantScheme string
	}{
		{name: "plain HTTP", wantScheme: "http://"},
		{name: "TLS", tls: true, wantScheme: "https://"},
		{name: "forwarded HTTPS", forwarded: "https", wantScheme: "https://"},
		{name: "forwarded HTTP", forwarded: "http", wantScheme: "http://"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := handler{s: newStore()}
			req := httptest.NewRequest(http.MethodPost, "http://example.com/", strings.NewReader(`{"url":"https://example.org"}`))
			if tc.tls {
				req.TLS = &tls.ConnectionState{}
			}
			if tc.forwarded != "" {
				req.Header.Set("X-Forwarded-Proto", tc.forwarded)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			var resp struct {
				Short string `json:"short"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("decode response %q: %v", rec.Body.String(), err)
			}
			if !strings.HasPrefix(resp.Short, tc.wantScheme) {
				t.Fatalf("short = %q, want prefix %q", resp.Short, tc.wantScheme)
			}
			if !strings.HasPrefix(resp.Short, tc.wantScheme+"example.com/") {
				t.Fatalf("short = %q, want %qexample.com/<code>", resp.Short, tc.wantScheme)
			}
		})
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

func TestHandlerRejectsUnsupportedMethods(t *testing.T) {
	for _, method := range []string{http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions, http.MethodTrace} {
		for _, path := range []string{"/existing", "/missing"} {
			t.Run(method+path, func(t *testing.T) {
				s := newStore()
				s.put("existing", "https://example.org")
				rec := httptest.NewRecorder()
				handler{s: s}.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
				if rec.Code != http.StatusMethodNotAllowed {
					t.Errorf("status = %d, want 405", rec.Code)
				}
				if got := rec.Header().Get("Allow"); got != "GET, HEAD, POST" {
					t.Errorf("Allow = %q", got)
				}
				if got := rec.Header().Get("Location"); got != "" {
					t.Errorf("unexpected redirect to %q", got)
				}
				if got, ok := s.get("existing"); !ok || got != "https://example.org" {
					t.Fatal("stored link changed")
				}
			})
		}
	}
}

func TestHandlerLookupMethods(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		for _, path := range []string{"/existing", "/missing"} {
			t.Run(method+path, func(t *testing.T) {
				s := newStore()
				s.put("existing", "https://example.org")
				rec := httptest.NewRecorder()
				handler{s: s}.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
				wantStatus, wantLocation := http.StatusNotFound, ""
				if path == "/existing" {
					wantStatus, wantLocation = http.StatusFound, "https://example.org"
				}
				if rec.Code != wantStatus || rec.Header().Get("Location") != wantLocation {
					t.Fatalf("status = %d, Location = %q; want %d, %q", rec.Code, rec.Header().Get("Location"), wantStatus, wantLocation)
				}
				if method == http.MethodHead && path == "/existing" && rec.Body.Len() != 0 {
					t.Fatal("HEAD redirect wrote a body")
				}
			})
		}
	}
}
