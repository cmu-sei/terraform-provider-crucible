// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

// Offline unit tests for the FromMap converters in the structs package that carry real
// parsing/branching logic (id parsing, empty->nil optionals, quote stripping). Trivial
// field-copy converters (the various ToMap methods, HasID searches) are intentionally not
// tested — they restate the implementation and offer no regression value.
//
// CAVEAT: these converters are coupled to the map shapes the internal/api layer passes
// around. That layer is slated for refactoring, so these tests are the most likely in the
// repo to need updating; keep them focused on input->output behavior rather than API
// wiring.
package structs_test

import (
	"testing"

	"github.com/cmu-sei/terraform-provider-crucible/internal/structs"
)

func TestProxmoxInfoFromMap(t *testing.T) {
	t.Run("integer id as float64 (JSON number)", func(t *testing.T) {
		got := structs.ProxmoxInfoFromMap(map[string]interface{}{
			"id":   float64(101),
			"node": "pve1",
			"type": "qemu",
		})
		if got == nil {
			t.Fatal("expected non-nil ProxmoxInfo")
		}
		if got.Id != 101 || got.Node != "pve1" || got.Type != "qemu" {
			t.Errorf("got %+v, want {Id:101 Node:pve1 Type:qemu}", got)
		}
	})

	t.Run("numeric string id", func(t *testing.T) {
		got := structs.ProxmoxInfoFromMap(map[string]interface{}{
			"id":   "102",
			"node": "pve1",
			"type": "qemu",
		})
		if got == nil || got.Id != 102 {
			t.Fatalf("got %+v, want Id=102", got)
		}
	})

	t.Run("provider node/type/id string id", func(t *testing.T) {
		got := structs.ProxmoxInfoFromMap(map[string]interface{}{
			"id":   "pve1/qemu/103",
			"node": "pve1",
			"type": "qemu",
		})
		if got == nil || got.Id != 103 {
			t.Fatalf("got %+v, want Id=103", got)
		}
	})

	t.Run("unparseable id returns nil", func(t *testing.T) {
		got := structs.ProxmoxInfoFromMap(map[string]interface{}{
			"id":   "pve1/qemu/notanint",
			"node": "pve1",
			"type": "qemu",
		})
		if got != nil {
			t.Errorf("expected nil for unparseable id, got %+v", got)
		}
	})
}

func TestAppInfoFromMap(t *testing.T) {
	t.Run("empty optional strings become nil", func(t *testing.T) {
		got := structs.AppInfoFromMap(map[string]interface{}{
			"app_id":             "a1",
			"v_id":               "v1",
			"name":               "",
			"url":                "",
			"icon":               "",
			"embeddable":         "",
			"load_in_background": "",
			"app_template_id":    "",
		})
		if got.ID != "a1" || got.ViewID != "v1" {
			t.Errorf("required fields wrong: %+v", got)
		}
		if got.Name != nil || got.URL != nil || got.Icon != nil ||
			got.Embeddable != nil || got.LoadInBackground != nil || got.AppTemplateID != nil {
			t.Errorf("expected nil optionals, got %+v", got)
		}
	})

	t.Run("quotes stripped from embeddable / load_in_background", func(t *testing.T) {
		got := structs.AppInfoFromMap(map[string]interface{}{
			"app_id":             "a1",
			"v_id":               "v1",
			"name":               "myApp",
			"url":                "http://x",
			"icon":               "ico",
			"embeddable":         `"true"`,
			"load_in_background": `"false"`,
			"app_template_id":    "tmpl",
		})
		if got.Name != "myApp" || got.URL != "http://x" || got.Icon != "ico" {
			t.Errorf("string passthrough wrong: %+v", got)
		}
		if got.Embeddable != "true" {
			t.Errorf("Embeddable = %v, want true (quotes stripped)", got.Embeddable)
		}
		if got.LoadInBackground != "false" {
			t.Errorf("LoadInBackground = %v, want false (quotes stripped)", got.LoadInBackground)
		}
		if got.AppTemplateID != "tmpl" {
			t.Errorf("AppTemplateID = %v, want tmpl", got.AppTemplateID)
		}
	})
}

func TestTeamInfoFromMap(t *testing.T) {
	// Exercises userInfoFromMap (unexported) indirectly.
	got := structs.TeamInfoFromMap(map[string]interface{}{
		"team_id":     "t1",
		"name":        "Admins",
		"role":        "View Member",
		"permissions": []interface{}{"perm-a", "perm-b"},
		"user": []interface{}{
			map[string]interface{}{"user_id": "u1", "role": "Owner"},
			map[string]interface{}{"user_id": "u2", "role": nil},
		},
	})

	if got.ID != "t1" || got.Name != "Admins" || got.Role != "View Member" {
		t.Errorf("scalar fields wrong: %+v", got)
	}
	if len(got.Permissions) != 2 || got.Permissions[0] != "perm-a" || got.Permissions[1] != "perm-b" {
		t.Errorf("permissions = %v, want [perm-a perm-b]", got.Permissions)
	}
	if len(got.Users) != 2 {
		t.Fatalf("users len = %d, want 2", len(got.Users))
	}
	if got.Users[0].ID != "u1" || got.Users[0].Role != "Owner" {
		t.Errorf("user[0] = %+v, want {u1 Owner}", got.Users[0])
	}
	if got.Users[1].ID != "u2" || got.Users[1].Role != nil {
		t.Errorf("user[1] = %+v, want {u2 <nil>}", got.Users[1])
	}
	if got.AppInstances != nil {
		t.Errorf("expected nil AppInstances when none provided, got %v", got.AppInstances)
	}
}
