// Copyright 2026 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package api

import (
	"context"
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
// Player has no value for are omitted. RoleID and RoleName are nil when the
// user has no system role.
type PlayerUserDirectoryEntry struct {
	ID                 string
	Name               string
	RoleID             *string
	RoleName           *string
	IdentityAttributes map[string]string
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

	var entries []playerclient.UserDirectoryEntry
	if resp.JSON200 != nil {
		entries = *resp.JSON200
	}
	return matchPlayerUser(entries, filter)
}

func matchPlayerUser(entries []playerclient.UserDirectoryEntry, filter PlayerUserFilter) (*PlayerUserDirectoryEntry, error) {
	keys := make([]string, 0, len(filter.IdentityAttributes))
	for key := range filter.IdentityAttributes {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Player lists every configured attribute on every user, with a null value
	// when the user has none, so a key no user carries is not configured or not
	// visible to this account. Say so rather than report zero matches. Keys
	// match exactly: Player treats them as case-sensitive identifiers, so
	// "Email" and "email" can be two different attributes.
	listed := listedAttributeKeys(entries)
	for _, key := range keys {
		if _, ok := listed[key]; !ok {
			return nil, unknownAttributeKeyError(key, listed)
		}
	}

	var matches []playerclient.UserDirectoryEntry
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
		RoleName:           match.RoleName,
		IdentityAttributes: map[string]string{},
	}
	if match.RoleId != nil {
		roleID := match.RoleId.String()
		out.RoleID = &roleID
	}
	for _, attribute := range identityAttributes(match) {
		if attribute.Key != nil && attribute.Value != nil {
			out.IdentityAttributes[*attribute.Key] = *attribute.Value
		}
	}
	return out, nil
}

func listedAttributeKeys(entries []playerclient.UserDirectoryEntry) map[string]struct{} {
	listed := map[string]struct{}{}
	for _, entry := range entries {
		for _, attribute := range identityAttributes(entry) {
			if attribute.Key != nil {
				listed[*attribute.Key] = struct{}{}
			}
		}
	}
	return listed
}

// unknownAttributeKeyError names the keys Player did return, so a key that
// differs only by case, or a key that is not configured, is easy to correct.
func unknownAttributeKeyError(key string, listed map[string]struct{}) error {
	if len(listed) == 0 {
		return fmt.Errorf("player API returned no identity attributes, so key %q cannot be matched; check that Player's ClaimsTransformation:UserAttributes setting configures it and that the provider account has the ViewUsers permission", key)
	}

	available := make([]string, 0, len(listed))
	var sameExceptCase []string
	for candidate := range listed {
		available = append(available, fmt.Sprintf("%q", candidate))
		if strings.EqualFold(candidate, key) {
			sameExceptCase = append(sameExceptCase, fmt.Sprintf("%q", candidate))
		}
	}
	sort.Strings(available)
	sort.Strings(sameExceptCase)

	if len(sameExceptCase) > 0 {
		return fmt.Errorf("player API returned no identity attribute with key %q; keys match exactly, and Player has %s", key, strings.Join(sameExceptCase, ", "))
	}
	return fmt.Errorf("player API returned no identity attribute with key %q; Player has %s. Check that Player's ClaimsTransformation:UserAttributes setting configures it", key, strings.Join(available, ", "))
}

func attributesMatch(entry playerclient.UserDirectoryEntry, want map[string]string, caseInsensitive bool) bool {
	for key, value := range want {
		found := false
		for _, attribute := range identityAttributes(entry) {
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

func identityAttributes(entry playerclient.UserDirectoryEntry) []playerclient.UserIdentityAttribute {
	if entry.IdentityAttributes == nil {
		return nil
	}
	return *entry.IdentityAttributes
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
