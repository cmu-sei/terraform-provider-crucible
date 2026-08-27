// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package util

import (
	"context"
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

// NewHTTPClient returns the client used for all provider API and OAuth calls.
// Terraform otherwise waits indefinitely when a service accepts a TCP
// connection but never sends an HTTP response.
func NewHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = (&net.Dialer{
		Timeout:   HTTPConnectTimeout,
		KeepAlive: 30 * time.Second,
	}).DialContext

	return &http.Client{
		Transport: transport,
		Timeout:   HTTPRequestTimeout,
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
}

func (s passwordTokenSource) Token() (*oauth2.Token, error) {
	ctx, cancel := context.WithTimeout(context.Background(), HTTPRequestTimeout)
	defer cancel()
	ctx = context.WithValue(ctx, oauth2.HTTPClient, NewHTTPClient())

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
func AuthedHTTPClient(m map[string]string) *http.Client {
	key := strings.Join([]string{
		m["client_id"], m["client_secret"], m["username"], m["password"],
		m["auth_url"], m["player_token_url"], m["client_scopes"],
	}, "|")

	authClientsMu.Lock()
	defer authClientsMu.Unlock()
	if c, ok := authClients[key]; ok {
		return c
	}

	src := passwordTokenSource{cfg: oauthConfig(m), username: m["username"], pass: m["password"]}
	// nil seed token => the grant is deferred to the first request and refreshed
	// automatically on expiry.
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, NewHTTPClient())
	c := oauth2.NewClient(ctx, oauth2.ReuseTokenSource(nil, src))
	c.Timeout = HTTPRequestTimeout
	authClients[key] = c
	return c
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
