// Copyright 2026 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPlayerRoleDataSources(t *testing.T) {
	const config = `
data "crucible_player_role" "exact" {
  name = "Content Developer"
}

data "crucible_player_role" "case_insensitive" {
  name             = "CONTENT DEVELOPER"
  case_insensitive = true
}

data "crucible_player_team_role" "exact" {
  name = "Observer"
}

data "crucible_player_team_role" "case_insensitive" {
  name             = "OBSERVER"
  case_insensitive = true
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: tfConfig(config),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.crucible_player_role.exact", "id", "7fd6aa3e-a765-47b8-a77e-f58eae53a82f"),
				resource.TestCheckResourceAttr("data.crucible_player_role.exact", "name", "Content Developer"),
				resource.TestCheckResourceAttr("data.crucible_player_role.exact", "all_permissions", "false"),
				resource.TestCheckResourceAttr("data.crucible_player_role.exact", "immutable", "false"),
				resource.TestCheckResourceAttr("data.crucible_player_role.exact", "permissions.#", "1"),
				resource.TestCheckResourceAttrPair(
					"data.crucible_player_role.exact", "id",
					"data.crucible_player_role.case_insensitive", "id"),
				resource.TestCheckResourceAttr("data.crucible_player_team_role.exact", "id", "c875dcce-2488-4e73-8585-8375b4730151"),
				resource.TestCheckResourceAttr("data.crucible_player_team_role.exact", "name", "Observer"),
				resource.TestCheckResourceAttr("data.crucible_player_team_role.exact", "all_permissions", "false"),
				resource.TestCheckResourceAttr("data.crucible_player_team_role.exact", "immutable", "false"),
				resource.TestCheckResourceAttr("data.crucible_player_team_role.exact", "permissions.#", "3"),
				resource.TestCheckResourceAttrPair(
					"data.crucible_player_team_role.exact", "id",
					"data.crucible_player_team_role.case_insensitive", "id"),
			),
		}},
	})
}

func TestAccPlayerRoleDataSourcesNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: tfConfig(`
data "crucible_player_role" "missing" {
  name = "terraform-role-does-not-exist"
}
`),
			ExpectError: regexp.MustCompile(`resolved to 0 roles`),
		}},
	})
}

func TestAccPlayerTeamRoleDataSourceNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: tfConfig(`
data "crucible_player_team_role" "missing" {
  name = "terraform-team-role-does-not-exist"
}
`),
			ExpectError: regexp.MustCompile(`resolved to 0 team roles`),
		}},
	})
}
