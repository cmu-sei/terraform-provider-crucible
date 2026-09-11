// Copyright 2026 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package util

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestParseTimeout(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		defaultVal time.Duration
		want       time.Duration
		wantErr    bool
	}{
		{name: "empty uses default", value: "", defaultVal: 5 * time.Second, want: 5 * time.Second},
		{name: "duration", value: "250ms", defaultVal: 5 * time.Second, want: 250 * time.Millisecond},
		{name: "zero disables timeout", value: "0s", defaultVal: 5 * time.Second, want: 0},
		{name: "malformed", value: "fast", defaultVal: 5 * time.Second, wantErr: true},
		{name: "negative", value: "-1s", defaultVal: 5 * time.Second, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseTimeout(test.value, test.defaultVal)
			if (err != nil) != test.wantErr {
				t.Fatalf("ParseTimeout() error = %v, wantErr %t", err, test.wantErr)
			}
			if !test.wantErr && got != test.want {
				t.Fatalf("ParseTimeout() = %s, want %s", got, test.want)
			}
		})
	}
}

func TestNewHTTPClientHasConfiguredTimeout(t *testing.T) {
	client := NewHTTPClient(HTTPTimeouts{
		Connect: 250 * time.Millisecond,
		Request: 500 * time.Millisecond,
	})

	if client.Timeout != 500*time.Millisecond {
		t.Fatalf("client timeout = %s, want 500ms", client.Timeout)
	}
}

func TestHTTPClientStopsWaitingForAnUnresponsiveServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		<-request.Context().Done()
	}))
	defer server.Close()

	client := NewHTTPClient(HTTPTimeouts{
		Connect: HTTPConnectTimeout,
		Request: 50 * time.Millisecond,
	})
	started := time.Now()
	_, err := client.Get(server.URL)

	if err == nil {
		t.Fatal("request unexpectedly succeeded")
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("request timed out too slowly: %s", elapsed)
	}
}

func TestAuthedHTTPClientStopsWaitingForUnresponsiveTokenServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		// The OAuth transport does not reliably cancel the server-side handler
		// when its client-side request timeout expires. Keep the mock silent
		// longer than the configured timeout, but let httptest.Server close
		// cleanly after the assertion.
		time.Sleep(250 * time.Millisecond)
	}))
	defer server.Close()

	client, err := AuthedHTTPClient(testAuthConfig(server.URL, HTTPTimeouts{
		Connect: HTTPConnectTimeout,
		Request: 50 * time.Millisecond,
	}))
	if err != nil {
		t.Fatalf("AuthedHTTPClient() error = %v", err)
	}

	started := time.Now()
	_, err = client.Get(server.URL + "/api/roles")
	if err == nil {
		t.Fatal("request unexpectedly succeeded")
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("token request timed out too slowly: %s", elapsed)
	}
}

func TestAuthedHTTPClientStopsWaitingForUnresponsiveAPIServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/token" {
			response.Header().Set("Content-Type", "application/json")
			_, _ = response.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`))
			return
		}
		time.Sleep(250 * time.Millisecond)
	}))
	defer server.Close()

	client, err := AuthedHTTPClient(testAuthConfig(server.URL, HTTPTimeouts{
		Connect: HTTPConnectTimeout,
		Request: 50 * time.Millisecond,
	}))
	if err != nil {
		t.Fatalf("AuthedHTTPClient() error = %v", err)
	}

	started := time.Now()
	_, err = client.Get(server.URL + "/api/roles")
	if err == nil {
		t.Fatal("request unexpectedly succeeded")
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("API request timed out too slowly: %s", elapsed)
	}
}

func TestAuthedHTTPClientCacheIncludesTimeouts(t *testing.T) {
	base := testAuthConfig("http://127.0.0.1:1", HTTPTimeouts{
		Connect: 1 * time.Second,
		Request: 2 * time.Second,
	})
	first, err := AuthedHTTPClient(base)
	if err != nil {
		t.Fatalf("AuthedHTTPClient(first) error = %v", err)
	}

	changed := testAuthConfig("http://127.0.0.1:1", HTTPTimeouts{
		Connect: 1 * time.Second,
		Request: 3 * time.Second,
	})
	second, err := AuthedHTTPClient(changed)
	if err != nil {
		t.Fatalf("AuthedHTTPClient(second) error = %v", err)
	}

	if first == second {
		t.Fatal("clients with different request timeouts share a cache entry")
	}
	if first.Timeout != 2*time.Second {
		t.Fatalf("first client timeout = %s, want 2s", first.Timeout)
	}
	if second.Timeout != 3*time.Second {
		t.Fatalf("second client timeout = %s, want 3s", second.Timeout)
	}
}

func testAuthConfig(serverURL string, timeouts HTTPTimeouts) map[string]string {
	return map[string]string{
		"username":             "test-user",
		"password":             "test-password",
		"auth_url":             serverURL + "/auth",
		"player_token_url":     serverURL + "/token",
		"client_id":            "test-client",
		"client_secret":        "test-secret",
		"http_connect_timeout": timeouts.Connect.String(),
		"http_request_timeout": timeouts.Request.String(),
	}
}
