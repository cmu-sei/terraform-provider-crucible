// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

// Package casterclient wraps the oapi-codegen–generated Caster API client
// (casterclient.gen.go) with the provider's existing auth and URL conventions,
// mirroring internal/vmclient.
//
// The generated request paths already include the leading "/api/" segment and
// resolve relative to the server base URL, so the base URL must be the host
// root WITHOUT the "/api" suffix. util.GetCasterApiUrl normalizes the configured
// caster_api_url to end in "/api/"; we trim that back off to recover the root.
package casterclient

import (
	"strings"

	"github.com/cmu-sei/terraform-provider-crucible/internal/util"
)

// NewAuthed builds a ClientWithResponses for the Caster API using the provider
// config map: server root derived from caster_api_url, plus an OAuth2-
// authenticated HTTP client whose bearer token is cached and refreshed by
// util.AuthedHTTPClient (shared across all three generated clients).
func NewAuthed(m map[string]string) (*ClientWithResponses, error) {
	server := strings.TrimSuffix(util.GetCasterApiUrl(m), "/api/")
	return NewClientWithResponses(server, WithHTTPClient(util.AuthedHTTPClient(m)))
}
