// Copyright 2026 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package util

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewHTTPClientHasBoundedTimeouts(t *testing.T) {
	client := NewHTTPClient()

	if client.Timeout != HTTPRequestTimeout {
		t.Fatalf("client timeout = %s, want %s", client.Timeout, HTTPRequestTimeout)
	}
}

func TestHTTPClientStopsWaitingForAnUnresponsiveServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		<-request.Context().Done()
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.Timeout = 50 * time.Millisecond
	started := time.Now()
	_, err := client.Get(server.URL)

	if err == nil {
		t.Fatal("request unexpectedly succeeded")
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("request timed out too slowly: %s", elapsed)
	}
}
