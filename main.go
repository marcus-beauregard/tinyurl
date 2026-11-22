// SPDX-License-Identifier: GPL-2.0-or-later
// Copyright (C) 2026 Marcus Beauregard

package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

type store struct {
	mu sync.RWMutex
	m  map[string]string
}

func newStore() *store { return &store{m: make(map[string]string)} }

func (s *store) put(code, url string) { s.mu.Lock(); s.m[code] = url; s.mu.Unlock() }
func (s *store) get(code string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.m[code]
	return u, ok
}

func shortCode() (string, error) {
	b := make([]byte, 4)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}

type handler struct{ s *store }

// requestScheme returns the scheme clients should use in links generated from
// this request. A direct TLS connection is authoritative; behind a proxy the
// original scheme is signalled by X-Forwarded-Proto, whose first value is the
// one set by the outermost proxy.
func requestScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		if first := strings.TrimSpace(strings.Split(proto, ",")[0]); first == "https" {
			return "https"
		}
	}
	return "http"
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req map[string]string
		json.NewDecoder(r.Body).Decode(&req)
		code, err := shortCode()
		if err != nil {
			http.Error(w, "short code generation failed", http.StatusInternalServerError)
			return
		}
		h.s.put(code, req["url"])
		json.NewEncoder(w).Encode(map[string]string{"short": requestScheme(r) + "://" + r.Host + "/" + code})
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	code := r.URL.Path[1:]
	if u, ok := h.s.get(code); ok {
		http.Redirect(w, r, u, http.StatusFound)
		return
	}
	http.NotFound(w, r)
}

func main() {
	s := newStore()
	http.ListenAndServe(":8080", handler{s: s})
}
