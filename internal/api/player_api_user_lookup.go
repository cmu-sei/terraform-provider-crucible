// Copyright 2026 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/cmu-sei/terraform-provider-crucible/internal/playerclient"
)

// PlayerUserFilter selects a single Player user. Name and every entry in
// IdentityAttributes must match; an empty Name is not used as a filter.
type PlayerUserFilter struct {
	Name               string
	IdentityAttributes map[string]string
	CaseInsensitive    bool
}

// PlayerUserDirectoryEntry is a Player user as listed by GET /users, with the
// identity attributes Player holds for it keyed by attribute key. Attributes
// Player has no value for are omitted.
type PlayerUserDirectoryEntry struct {
	ID                 string
	Name               string
	RoleID             string
	RoleName           string
	IdentityAttributes map[string]string
}

// userDirectoryEntry decodes one element of GET /users. The committed
// playerclient spec types that response as []User, which predates the
// identityAttributes field, so the body is decoded here instead.
type userDirectoryEntry struct {
	playerclient.User
	IdentityAttributes []userIdentityAttribute `json:"identityAttributes"`
}

type userIdentityAttribute struct {
	Key   *string `json:"key"`
	Name  *string `json:"name"`
	Value *string `json:"value"`
}

// FindPlayerUser returns the one Player user that matches the filter.
//
// Player only includes identity attributes in GET /users when the caller holds
// the ViewUsers system permission, and only for the attributes configured in
// its ClaimsTransformation:UserAttributes setting.
func FindPlayerUser(filter PlayerUserFilter, m map[string]string) (*PlayerUserDirectoryEntry, error) {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return nil, err
	}
	resp, err := client.GetUsersWithResponse(context.Background())
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("player API returned status %d when listing users", resp.StatusCode())
	}

	var entries []userDirectoryEntry
	if err := json.Unmarshal(resp.Body, &entries); err != nil {
		return nil, fmt.Errorf("decode Player user list: %w", err)
	}
	return matchPlayerUser(entries, filter)
}

func matchPlayerUser(entries []userDirectoryEntry, filter PlayerUserFilter) (*PlayerUserDirectoryEntry, error) {
	keys := make([]string, 0, len(filter.IdentityAttributes))
	for key := range filter.IdentityAttributes {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Player lists every configured attribute on every user, with a null value
	// when the user has none, so a key no user carries is not configured or not
	// visible to this account. Say so rather than report zero matches.
	for _, key := range keys {
		if !attributeKeyListed(entries, key) {
			return nil, fmt.Errorf("player API returned no identity attribute with key %q; check that the key is configured in Player's ClaimsTransformation:UserAttributes setting and that the provider account has the ViewUsers permission", key)
		}
	}

	var matches []userDirectoryEntry
	for _, entry := range entries {
		if filter.Name != "" && (entry.Name == nil || !namesMatch(*entry.Name, filter.Name, filter.CaseInsensitive)) {
			continue
		}
		if !attributesMatch(entry, filter.IdentityAttributes, filter.CaseInsensitive) {
			continue
		}
		matches = append(matches, entry)
	}
	if len(matches) != 1 {
		return nil, fmt.Errorf("user lookup by %s resolved to %d users; expected exactly one", describeUserFilter(filter, keys), len(matches))
	}

	match := matches[0]
	if match.Id == nil {
		return nil, fmt.Errorf("player API returned user lookup by %s without an id", describeUserFilter(filter, keys))
	}
	out := &PlayerUserDirectoryEntry{
		ID:                 match.Id.String(),
		Name:               derefStr(match.Name),
		RoleName:           derefStr(match.RoleName),
		IdentityAttributes: map[string]string{},
	}
	if match.RoleId != nil {
		out.RoleID = match.RoleId.String()
	}
	for _, attribute := range match.IdentityAttributes {
		if attribute.Key != nil && attribute.Value != nil {
			out.IdentityAttributes[*attribute.Key] = *attribute.Value
		}
	}
	return out, nil
}

func attributeKeyListed(entries []userDirectoryEntry, key string) bool {
	for _, entry := range entries {
		for _, attribute := range entry.IdentityAttributes {
			if attribute.Key != nil && *attribute.Key == key {
				return true
			}
		}
	}
	return false
}

func attributesMatch(entry userDirectoryEntry, want map[string]string, caseInsensitive bool) bool {
	for key, value := range want {
		found := false
		for _, attribute := range entry.IdentityAttributes {
			if attribute.Key != nil && *attribute.Key == key &&
				attribute.Value != nil && namesMatch(*attribute.Value, value, caseInsensitive) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func describeUserFilter(filter PlayerUserFilter, sortedKeys []string) string {
	var parts []string
	if filter.Name != "" {
		parts = append(parts, fmt.Sprintf("name %q", filter.Name))
	}
	for _, key := range sortedKeys {
		parts = append(parts, fmt.Sprintf("%s = %q", key, filter.IdentityAttributes[key]))
	}
	return strings.Join(parts, ", ")
}
