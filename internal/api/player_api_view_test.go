// Copyright 2026 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package api

import (
	"testing"

	"github.com/cmu-sei/terraform-provider-crucible/internal/structs"
)

func TestCreateViewCommandIncludesTemplateStatus(t *testing.T) {
	command := createViewCommand(&structs.ViewInfo{
		Name:            "template",
		Description:     "description",
		Status:          "Active",
		CreateAdminTeam: true,
		IsTemplate:      true,
	})

	if command.IsTemplate == nil || !*command.IsTemplate {
		t.Fatalf("IsTemplate = %v, want true", command.IsTemplate)
	}
}
