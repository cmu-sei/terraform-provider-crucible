// Copyright 2026 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/cmu-sei/terraform-provider-crucible/internal/api"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccPlayerTeam(t *testing.T) {
	const viewName = "tf-acc-player-team"
	var teamID string
	sweepViewByName(t, viewName)
	registerViewCleanupByName(t, viewName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccViewDestroyed,
		Steps: []resource.TestStep{
			{
				Config: tfConfig(playerTeamConfig(viewName, "students")),
				Check: resource.ComposeAggregateTestCheckFunc(
					captureID("crucible_player_team.test", &teamID),
					resource.TestCheckResourceAttrSet("crucible_player_team.test", "id"),
					resource.TestCheckResourceAttr("crucible_player_team.test", "name", "students"),
					resource.TestCheckResourceAttr("crucible_player_team.test", "role", "View Member"),
					resource.TestCheckResourceAttr("crucible_player_team.test", "permissions.#", "0"),
					resource.TestCheckResourceAttr("crucible_player_team.test", "scoped_team_ids.#", "0"),
				),
			},
			{
				Config: tfConfig(playerTeamConfig(viewName, "students-updated")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_player_team.test", "name", "students-updated"),
				),
			},
			{
				ResourceName:      "crucible_player_team.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: tfConfig(playerTeamConfig(viewName, "students-updated")),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: tfConfig(playerTeamParentConfig(viewName)),
				Check:  testAccPlayerTeamAbsent(&teamID),
			},
		},
	})
}

func TestAccPlayerTeamCreateRelationshipFailurePreservesState(t *testing.T) {
	const viewName = "tf-acc-player-team-create-recovery"
	sweepViewByName(t, viewName)
	registerViewCleanupByName(t, viewName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccViewDestroyed,
		Steps: []resource.TestStep{
			{
				Config:      tfConfig(playerTeamInvalidPermissionConfig(viewName)),
				ExpectError: regexp.MustCompile(`Error configuring Player team relationships`),
			},
			{
				Config: tfConfig(playerTeamConfig(viewName, "students")),
				Check:  testAccPlayerTeamCount("students", 1),
			},
		},
	})
}

func playerTeamParentConfig(viewName string) string {
	return fmt.Sprintf(`
resource "crucible_player_view" "team_parent" {
  name              = %[1]q
  create_admin_team = false
  child_management  = "standalone"
}
`, viewName)
}

func playerTeamConfig(viewName, teamName string) string {
	return playerTeamParentConfig(viewName) + fmt.Sprintf(`
resource "crucible_player_team" "test" {
  view_id = crucible_player_view.team_parent.id
  name    = %[1]q
}
`, teamName)
}

func playerTeamInvalidPermissionConfig(viewName string) string {
	return playerTeamParentConfig(viewName) + `
resource "crucible_player_team" "test" {
  view_id     = crucible_player_view.team_parent.id
  name        = "students"
  permissions = ["00000000-0000-0000-0000-000000000000"]
}
`
}

func testAccPlayerTeamCount(name string, expected int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		view, ok := s.RootModule().Resources["crucible_player_view.team_parent"]
		if !ok {
			return fmt.Errorf("player view not found in state")
		}
		remote, err := api.ReadView(view.Primary.ID, getMap())
		if err != nil {
			return err
		}
		actual := 0
		for _, team := range remote.Teams {
			if ifaceString(team.Name) == name {
				actual++
			}
		}
		if actual != expected {
			return fmt.Errorf("found %d teams named %q, want %d", actual, name, expected)
		}
		return nil
	}
}

func testAccPlayerTeamAbsent(id *string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if *id == "" {
			return fmt.Errorf("player team id was not captured")
		}
		_, exists, err := api.ReadPlayerTeam(*id, getMap())
		if err != nil {
			return fmt.Errorf("checking deleted player team %s: %w", *id, err)
		}
		if exists {
			return fmt.Errorf("player team %s still exists after removal", *id)
		}
		return nil
	}
}
