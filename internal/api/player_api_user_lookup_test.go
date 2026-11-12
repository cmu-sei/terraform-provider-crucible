// Copyright 2026 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/cmu-sei/terraform-provider-crucible/internal/playerclient"
)

// usersBody mirrors a GET /users response from a Player configured with an
// "email" identity attribute. Alice and the second Bob share a name; Carol has
// no email value.
const usersBody = `[
  {"id": "00000000-0000-0000-0000-00000000000a", "name": "Alice", "roleId": "7fd6aa3e-a765-47b8-a77e-f58eae53a82f", "roleName": "Content Developer",
   "identityAttributes": [{"key": "email", "name": "Email", "value": "alice@example.com"}]},
  {"id": "00000000-0000-0000-0000-00000000000b", "name": "Bob", "roleId": null, "roleName": null,
   "identityAttributes": [{"key": "email", "name": "Email", "value": "bob@example.com"}]},
  {"id": "00000000-0000-0000-0000-00000000000c", "name": "Bob", "roleId": null, "roleName": null,
   "identityAttributes": [{"key": "email", "name": "Email", "value": "Bob.Two@example.com"}]},
  {"id": "00000000-0000-0000-0000-00000000000d", "name": "Carol", "roleId": null, "roleName": null,
   "identityAttributes": [{"key": "email", "name": "Email", "value": null}]}
]`

func TestMatchPlayerUser(t *testing.T) {
	var entries []playerclient.UserDirectoryEntry
	if err := json.Unmarshal([]byte(usersBody), &entries); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}

	tests := []struct {
		name    string
		filter  PlayerUserFilter
		wantID  string
		wantErr string
	}{
		{name: "unique name", filter: PlayerUserFilter{Name: "Alice"}, wantID: "00000000-0000-0000-0000-00000000000a"},
		{name: "duplicate name", filter: PlayerUserFilter{Name: "Bob"}, wantErr: "resolved to 2 users"},
		{name: "case-sensitive name miss", filter: PlayerUserFilter{Name: "alice"}, wantErr: "resolved to 0 users"},
		{name: "case-insensitive name hit", filter: PlayerUserFilter{Name: "alice", CaseInsensitive: true}, wantID: "00000000-0000-0000-0000-00000000000a"},
		{
			name:   "attribute separates duplicate names",
			filter: PlayerUserFilter{IdentityAttributes: map[string]string{"email": "bob@example.com"}},
			wantID: "00000000-0000-0000-0000-00000000000b",
		},
		{
			name:   "name and attribute together",
			filter: PlayerUserFilter{Name: "Bob", IdentityAttributes: map[string]string{"email": "Bob.Two@example.com"}},
			wantID: "00000000-0000-0000-0000-00000000000c",
		},
		{
			name:    "name and attribute disagree",
			filter:  PlayerUserFilter{Name: "Alice", IdentityAttributes: map[string]string{"email": "bob@example.com"}},
			wantErr: `name "Alice", email = "bob@example.com" resolved to 0 users`,
		},
		{
			name:    "case-sensitive attribute miss",
			filter:  PlayerUserFilter{IdentityAttributes: map[string]string{"email": "bob.two@example.com"}},
			wantErr: "resolved to 0 users",
		},
		{
			name:   "case-insensitive attribute hit",
			filter: PlayerUserFilter{IdentityAttributes: map[string]string{"email": "bob.two@example.com"}, CaseInsensitive: true},
			wantID: "00000000-0000-0000-0000-00000000000c",
		},
		{
			name:    "attribute key not listed",
			filter:  PlayerUserFilter{IdentityAttributes: map[string]string{"username": "alice"}},
			wantErr: `no identity attribute with key "username"`,
		},
		{
			name:    "null attribute value never matches",
			filter:  PlayerUserFilter{Name: "Carol", IdentityAttributes: map[string]string{"email": ""}},
			wantErr: "resolved to 0 users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := matchPlayerUser(entries, tt.filter)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("got error %v, want one containing %q (result %+v)", err, tt.wantErr, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ID != tt.wantID {
				t.Fatalf("got id %q, want %q", got.ID, tt.wantID)
			}
		})
	}
}

func TestMatchPlayerUserResult(t *testing.T) {
	var entries []playerclient.UserDirectoryEntry
	if err := json.Unmarshal([]byte(usersBody), &entries); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}

	alice, err := matchPlayerUser(entries, PlayerUserFilter{Name: "Alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alice.Name != "Alice" || alice.RoleID != "7fd6aa3e-a765-47b8-a77e-f58eae53a82f" || alice.RoleName != "Content Developer" {
		t.Fatalf("unexpected user %+v", alice)
	}
	if len(alice.IdentityAttributes) != 1 || alice.IdentityAttributes["email"] != "alice@example.com" {
		t.Fatalf("unexpected identity attributes %v", alice.IdentityAttributes)
	}

	carol, err := matchPlayerUser(entries, PlayerUserFilter{Name: "Carol"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if carol.RoleID != "" || len(carol.IdentityAttributes) != 0 {
		t.Fatalf("null role and attribute values should be empty, got %+v", carol)
	}
}
