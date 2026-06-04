// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

// Package vmclient wraps the oapi-codegen–generated Player VM API client
// (vmclient.gen.go) with the provider's existing auth and URL conventions.
//
// The generated request paths already include the leading "/api/" segment and
// resolve relative to the server base URL, so the base URL must be the host
// root WITHOUT the "/api" suffix. util.GetVmApiUrl normalizes the configured
// vm_api_url to end in "/api/"; we trim that back off to recover the root.
package vmclient

import (
	"context"
	"net/http"
	"strings"

	"github.com/cmu-sei/terraform-provider-crucible/internal/util"
)

// NewAuthed builds a ClientWithResponses for the Player VM API using the
// provider config map. It derives the server root from vm_api_url and attaches
// a request editor that injects a fresh OAuth2 bearer token (the same token
// flow the hand-written client used). The token is fetched once per client.
func NewAuthed(m map[string]string) (*ClientWithResponses, error) {
	// util.GetVmApiUrl returns "<root>/api/"; the generated paths add "/api/"
	// themselves, so strip it to avoid a doubled "/api/api/" path.
	server := strings.TrimSuffix(util.GetVmApiUrl(m), "/api/")

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
