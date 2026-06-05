// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

// Package playerclient wraps the oapi-codegen–generated Player API client
// (playerclient.gen.go) with the provider's existing auth and URL conventions,
// mirroring internal/vmclient.
//
// The generated request paths already include the leading "/api/" segment and
// resolve relative to the server base URL, so the base URL must be the host
// root WITHOUT the "/api" suffix. util.GetPlayerApiUrl normalizes the configured
// player_api_url to end in "/api/"; we trim that back off to recover the root.
package playerclient

import (
	"context"
	"net/http"
	"strings"

	"github.com/cmu-sei/terraform-provider-crucible/internal/util"
)

// NewAuthed builds a ClientWithResponses for the Player API using the provider
// config map: server root derived from player_api_url, plus a request editor
// that injects a fresh OAuth2 bearer token (the same flow the hand-written
// client used).
func NewAuthed(m map[string]string) (*ClientWithResponses, error) {
	server := strings.TrimSuffix(util.GetPlayerApiUrl(m), "/api/")

	auth, err := util.GetAuth(m)
	if err != nil {
		return nil, err
	}

	bearer := func(_ context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+auth)
		return nil
	}

	return NewClientWithResponses(server, WithRequestEditorFn(bearer))
}
