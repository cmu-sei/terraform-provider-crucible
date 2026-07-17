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

func TestAccPlayerTeamUser(t *testing.T) {
	const viewName = "tf-acc-player-team-user"
	userID := envOrDefault("TF_TEST_VIEW_USER_ID", "9b3b331c-10c1-448b-8114-21b2586d8e38")
	var membershipID string
	sweepViewByName(t, viewName)
	registerViewCleanupByName(t, viewName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccViewDestroyed,
		Steps: []resource.TestStep{
			{
				Config: tfConfig(playerTeamUserConfig(viewName, userID, false)),
				Check: resource.ComposeAggregateTestCheckFunc(
					captureID("crucible_player_team_user.test", &membershipID),
					resource.TestCheckResourceAttrSet("crucible_player_team_user.test", "id"),
					resource.TestCheckResourceAttr("crucible_player_team_user.test", "user_id", userID),
					resource.TestCheckNoResourceAttr("crucible_player_team_user.test", "role"),
					resource.TestCheckResourceAttrPair(
						"crucible_player_team_user.test", "team_id",
						"crucible_player_team.team_user_parent", "id"),
				),
			},
			{
				Config: tfConfig(playerTeamUserConfig(viewName, userID, true)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_player_team_user.test", "role", "View Member"),
				),
			},
			{
				Config: tfConfig(playerTeamUserConfig(viewName, userID, true)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				ResourceName:      "crucible_player_team_user.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: tfConfig(playerTeamUserParentConfig(viewName)),
				Check:  testAccPlayerTeamUserAbsent(&membershipID),
			},
		},
	})
}

func playerTeamUserParentConfig(viewName string) string {
	return fmt.Sprintf(`
resource "crucible_player_view" "team_user_parent" {
  name              = %[1]q
  create_admin_team = false
  child_management  = "standalone"
}

resource "crucible_player_team" "team_user_parent" {
  view_id = crucible_player_view.team_user_parent.id
  name    = "students"
}
`, viewName)
}

func playerTeamUserConfig(viewName, userID string, updated bool) string {
	role := ""
	if updated {
		role = `
  role = "View Member"`
	}
	return playerTeamUserParentConfig(viewName) + fmt.Sprintf(`
resource "crucible_player_team_user" "test" {
  team_id = crucible_player_team.team_user_parent.id
  user_id = %[1]q%[2]s
}
`, userID, role)
}

func testAccPlayerTeamUserAbsent(id *string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if *id == "" {
			return fmt.Errorf("player team membership id was not captured")
		}
		_, exists, err := api.ReadPlayerTeamUser(*id, getMap())
		if err != nil {
			return fmt.Errorf("checking deleted player team membership %s: %w", *id, err)
		}
		if exists {
			return fmt.Errorf("player team membership %s still exists after removal", *id)
		}
		return nil
	}
}
