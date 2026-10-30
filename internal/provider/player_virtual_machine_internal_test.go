// Copyright 2026 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestVMCreateCheckpoint(t *testing.T) {
	plan := vmModel{
		ID:         types.StringValue("6a7ec409-d275-4b31-94d3-a51cb61d2519"),
		VMID:       types.StringUnknown(),
		URL:        types.StringUnknown(),
		DefaultURL: types.BoolUnknown(),
		Name:       types.StringValue("vm"),
		UserID:     types.StringUnknown(),
		Embeddable: types.BoolValue(true),
	}

	got := vmCreateCheckpoint(plan)

	if got.VMID.ValueString() != plan.ID.ValueString() {
		t.Errorf("VMID = %v, want the new id %v", got.VMID, plan.ID)
	}
	for name, v := range map[string]interface{ IsUnknown() bool }{
		"url": got.URL, "default_url": got.DefaultURL, "user_id": got.UserID,
	} {
		if v.IsUnknown() {
			t.Errorf("%s is still unknown; state can't hold unknown values", name)
		}
	}
	if got.Name.ValueString() != "vm" || !got.Embeddable.ValueBool() {
		t.Errorf("configured values changed: name=%v embeddable=%v", got.Name, got.Embeddable)
	}
}

func TestVMCreateCheckpointKeepsConfiguredValues(t *testing.T) {
	plan := vmModel{
		ID:     types.StringValue("6a7ec409-d275-4b31-94d3-a51cb61d2519"),
		URL:    types.StringValue("https://example.com/console"),
		UserID: types.StringValue("33605140-f28f-4722-b161-8540e97e6bab"),
	}

	got := vmCreateCheckpoint(plan)

	if got.URL.ValueString() != "https://example.com/console" {
		t.Errorf("URL = %v, want the configured value", got.URL)
	}
	if got.UserID.ValueString() != "33605140-f28f-4722-b161-8540e97e6bab" {
		t.Errorf("UserID = %v, want the configured value", got.UserID)
	}
}
