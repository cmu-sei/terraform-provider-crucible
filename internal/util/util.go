// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package util

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// Helper functions used throughout provider

const (
	// HTTPConnectTimeout bounds how long a provider API call can spend opening a
	// TCP connection.
	HTTPConnectTimeout = 5 * time.Second

	// HTTPRequestTimeout bounds the complete lifetime of a provider API call,
	// including connecting, sending the request, and reading the response.
	HTTPRequestTimeout = 60 * time.Second
)

// HTTPTimeouts controls the connection and complete-request bounds used by
// provider API and OAuth calls. A zero duration disables the corresponding
// bound.
type HTTPTimeouts struct {
	Connect time.Duration
	Request time.Duration
}

// ParseTimeout parses a configured timeout, using defaultTimeout when value is
// empty. Zero is valid and disables the timeout; negative values are rejected.
func ParseTimeout(value string, defaultTimeout time.Duration) (time.Duration, error) {
	if value == "" {
		return defaultTimeout, nil
	}

	timeout, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("must be a valid duration such as %q or %q: %w", "5s", "2m", err)
	}
	if timeout < 0 {
		return 0, fmt.Errorf("must be zero or greater")
	}
	return timeout, nil
}

// ParseHTTPTimeouts parses provider configuration values, applying the
// existing defaults when either value is absent.
func ParseHTTPTimeouts(connect, request string) (HTTPTimeouts, error) {
	connectTimeout, err := ParseTimeout(connect, HTTPConnectTimeout)
	if err != nil {
		return HTTPTimeouts{}, fmt.Errorf("http connect timeout: %w", err)
	}

	requestTimeout, err := ParseTimeout(request, HTTPRequestTimeout)
	if err != nil {
		return HTTPTimeouts{}, fmt.Errorf("http request timeout: %w", err)
	}

	return HTTPTimeouts{
		Connect: connectTimeout,
		Request: requestTimeout,
	}, nil
}

// HTTPTimeoutsFromConfig parses timeout values from provider resource data.
// Missing keys retain the default behavior used by callers that predate
// configurable timeouts.
func HTTPTimeoutsFromConfig(m map[string]string) (HTTPTimeouts, error) {
	return ParseHTTPTimeouts(m["http_connect_timeout"], m["http_request_timeout"])
}

// NewHTTPClient returns the client used for all provider API and OAuth calls.
// Terraform otherwise waits indefinitely when a service accepts a TCP
// connection but never sends an HTTP response.
func NewHTTPClient(timeouts HTTPTimeouts) *http.Client {
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return &http.Client{
			Transport: http.DefaultTransport,
			Timeout:   timeouts.Request,
		}
	}
	transport = transport.Clone()
	transport.DialContext = (&net.Dialer{
		Timeout:   timeouts.Connect,
		KeepAlive: 30 * time.Second,
	}).DialContext

	return &http.Client{
		Transport: transport,
		Timeout:   timeouts.Request,
	}
}

// As performs a checked type assertion of v to T, returning the zero value of T
// when v is nil or not a T. It exists so the many decodes of Terraform's
// map[string]interface{} schema data don't each repeat the comma-ok dance (and
// don't trip forcetypeassert with bare v.(T) assertions). The zero-value-on-miss
// behavior matches the previous code's effective handling: an absent optional
// attribute decodes to "" / 0 / nil rather than panicking.
func As[T any](v interface{}) T {
	t, _ := v.(T)
	return t
}

// ToStringSlice converts a slice of empty interfaces to a slice of strings. Go won't let us do this implicitly.
func ToStringSlice(data *[]interface{}) *[]string {
	var converted []string
	for _, entry := range *data {
		converted = append(converted, As[string](entry))
	}
	return &converted
}

// oauthConfig builds the OAuth2 password-grant config from the provider's
// settings map, so the credential/endpoint wiring lives in one place.
func oauthConfig(m map[string]string) *oauth2.Config {
	scopes := strings.Split(m["client_scopes"], ",")
	if len(scopes) == 0 || (len(scopes) == 1 && scopes[0] == "") {
		scopes = nil
	}
	return &oauth2.Config{
		ClientID:     m["client_id"],
		ClientSecret: m["client_secret"],
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  m["auth_url"],
			TokenURL: m["player_token_url"],
		},
	}
}

// passwordTokenSource is an oauth2.TokenSource that performs the password grant
// on demand. Wrapped in oauth2.ReuseTokenSource it caches the token and only
// re-runs the grant when the cached token expires.
type passwordTokenSource struct {
	cfg            *oauth2.Config
	username, pass string
	httpTimeouts   HTTPTimeouts
}

func (s passwordTokenSource) Token() (*oauth2.Token, error) {
	ctx := context.Background()
	if s.httpTimeouts.Request > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.httpTimeouts.Request)
		defer cancel()
	}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, NewHTTPClient(s.httpTimeouts))

	return s.cfg.PasswordCredentialsToken(ctx, s.username, s.pass)
}

// authClients caches one token-backed *http.Client per full-credential+endpoint
// key (including the password), so the OAuth2 password grant runs once per process
// (per distinct config) and tokens auto-refresh thereafter, rather than once per API
// call. The password must be part of the key: two configs that differ only by
// password are distinct credentials and must not share a cached token.
var (
	authClientsMu sync.Mutex
	authClients   = map[string]*http.Client{}
)

// AuthedHTTPClient returns an *http.Client that injects (and refreshes) an
// OAuth2 bearer token built from the provider config map. The underlying token
// source is cached by credentials+endpoints, so repeated calls — including the
// per-item calls inside the team/user/permission loops — reuse a single token
// instead of re-running the password grant every time.
func AuthedHTTPClient(m map[string]string) (*http.Client, error) {
	httpTimeouts, err := HTTPTimeoutsFromConfig(m)
	if err != nil {
		return nil, err
	}

	key := strings.Join([]string{
		m["client_id"], m["client_secret"], m["username"], m["password"],
		m["auth_url"], m["player_token_url"], m["client_scopes"],
		httpTimeouts.Connect.String(), httpTimeouts.Request.String(),
	}, "|")

	authClientsMu.Lock()
	defer authClientsMu.Unlock()
	if c, ok := authClients[key]; ok {
		return c, nil
	}

	src := passwordTokenSource{
		cfg:          oauthConfig(m),
		username:     m["username"],
		pass:         m["password"],
		httpTimeouts: httpTimeouts,
	}
	// nil seed token => the grant is deferred to the first request and refreshed
	// automatically on expiry.
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, NewHTTPClient(httpTimeouts))
	c := oauth2.NewClient(ctx, oauth2.ReuseTokenSource(nil, src))
	c.Timeout = httpTimeouts.Request
	authClients[key] = c
	return c, nil
}

// PairInList returns true if a given key/value pair exists somewhere in a list of maps.
func PairInList(list []interface{}, key, value string) bool {
	for _, curr := range list {
		asMap := As[map[string]interface{}](curr)
		if asMap[key] == value {
			return true
		}
	}
	return false
}

// Ternary returns a if the condition is true and b otherwise. Go doesn't have an actual ternary operator.
//
// param condition: The condition to evaluate
//
// param a: The value to return if condition is true
//
// param b: The value to return if condition is false.
func Ternary(condition bool, a, b interface{}) interface{} {
	if condition {
		return a
	}
	return b
}

// StrSliceContains returns true if a string slice contains a given string.
//
// param arr: The slice to look in
//
// param str: The string to look for
//
// Returns true if str is in arr and false if not.
func StrSliceContains(arr *[]string, str string) bool {
	for _, elem := range *arr {
		if elem == str {
			return true
		}
	}
	return false
}

// Returns the normalized url for the player api.
func GetPlayerApiUrl(m map[string]string) string {
	return GetApiUrl(m, "player_api_url")
}

// Returns the normalized url for the vm api.
func GetVmApiUrl(m map[string]string) string {
	return GetApiUrl(m, "vm_api_url")
}

// Returns the normalized url for the caster api.
func GetCasterApiUrl(m map[string]string) string {
	return GetApiUrl(m, "caster_api_url")
}

// GetApiUrl returns a url from the settings map, normalized to end in /api/
//
// param m: The settings map
//
// param urlName: The name of the url setting in the map
//
// Returns empty string is urlName is not found in the map.
func GetApiUrl(m map[string]string, urlName string) string {
	log.Printf("! Getting API Url for %s", urlName)
	if url, exists := m[urlName]; exists {
		log.Printf("! URL = %s", url)
		url = strings.TrimSuffix(url, "/")
		url = strings.TrimSuffix(url, "/api")
		url = url + "/api/"
		log.Printf("! Normalized URL = %s", url)
		return url
	}

	log.Printf("! URL not found")
	return ""
}
