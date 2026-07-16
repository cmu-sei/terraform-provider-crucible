// Copyright 2026 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider_test

import (
	"fmt"
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
				Config: tfConfig(playerTeamConfig(viewName, "students", false)),
				Check: resource.ComposeAggregateTestCheckFunc(
					captureID("crucible_player_team.test", &teamID),
					resource.TestCheckResourceAttrSet("crucible_player_team.test", "id"),
					resource.TestCheckResourceAttr("crucible_player_team.test", "name", "students"),
					resource.TestCheckResourceAttr("crucible_player_team.test", "role", "View Member"),
					resource.TestCheckResourceAttr("crucible_player_team.test", "default", "false"),
					resource.TestCheckResourceAttr("crucible_player_team.test", "permissions.#", "0"),
					resource.TestCheckResourceAttr("crucible_player_team.test", "scoped_team_ids.#", "0"),
					testAccVerifyPlayerTeamDefault(false),
				),
			},
			{
				Config: tfConfig(playerTeamConfig(viewName, "students-updated", true)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_player_team.test", "name", "students-updated"),
					resource.TestCheckResourceAttr("crucible_player_team.test", "default", "true"),
					testAccVerifyPlayerTeamDefault(true),
				),
			},
			{
				ResourceName:      "crucible_player_team.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: tfConfig(playerTeamConfig(viewName, "students-updated", false)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_player_team.test", "default", "false"),
					testAccVerifyPlayerTeamDefault(false),
				),
			},
			{
				Config: tfConfig(playerTeamConfig(viewName, "students-updated", false)),
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

func playerTeamParentConfig(viewName string) string {
	return fmt.Sprintf(`
resource "crucible_player_view" "team_parent" {
  name              = %[1]q
  create_admin_team = false
  child_management  = "standalone"
}
`, viewName)
}

func playerTeamConfig(viewName, teamName string, defaultTeam bool) string {
	return playerTeamParentConfig(viewName) + fmt.Sprintf(`
resource "crucible_player_team" "test" {
  view_id = crucible_player_view.team_parent.id
  name    = %[1]q
  default = %[2]t
}
`, teamName, defaultTeam)
}

func testAccVerifyPlayerTeamDefault(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		view := s.RootModule().Resources["crucible_player_view.team_parent"]
		team := s.RootModule().Resources["crucible_player_team.test"]
		remote, err := api.ReadViewTopLevel(view.Primary.ID, getMap())
		if err != nil {
			return err
		}
		expectedID := ""
		if expected {
			expectedID = team.Primary.ID
		}
		if remote.DefaultTeamID != expectedID {
			return fmt.Errorf("default team id = %q, want %q", remote.DefaultTeamID, expectedID)
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
