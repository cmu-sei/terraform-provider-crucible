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

func TestAccPlayerApplicationInstance(t *testing.T) {
	const viewName = "tf-acc-player-application-instance"
	var instanceID, teamID string
	sweepViewByName(t, viewName)
	registerViewCleanupByName(t, viewName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccViewDestroyed,
		Steps: []resource.TestStep{
			{
				Config: tfConfig(playerApplicationInstanceConfig(viewName, 0)),
				Check: resource.ComposeAggregateTestCheckFunc(
					captureID("crucible_player_application_instance.test", &instanceID),
					captureID("crucible_player_team.application_instance_parent", &teamID),
					resource.TestCheckResourceAttrSet("crucible_player_application_instance.test", "id"),
					resource.TestCheckResourceAttr("crucible_player_application_instance.test", "display_order", "0"),
					resource.TestCheckResourceAttrPair(
						"crucible_player_application_instance.test", "team_id",
						"crucible_player_team.application_instance_parent", "id"),
					resource.TestCheckResourceAttrPair(
						"crucible_player_application_instance.test", "application_id",
						"crucible_player_application.application_instance_parent", "id"),
				),
			},
			{
				Config: tfConfig(playerApplicationInstanceConfig(viewName, 2)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_player_application_instance.test", "display_order", "2"),
				),
			},
			{
				Config: tfConfig(playerApplicationInstanceConfig(viewName, 2)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				ResourceName:      "crucible_player_application_instance.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(*terraform.State) (string, error) {
					return teamID + "/" + instanceID, nil
				},
			},
			{
				Config: tfConfig(playerApplicationInstanceParentConfig(viewName)),
				Check:  testAccPlayerApplicationInstanceAbsent(&instanceID, &teamID),
			},
		},
	})
}

func playerApplicationInstanceParentConfig(viewName string) string {
	return fmt.Sprintf(`
resource "crucible_player_view" "application_instance_parent" {
  name              = %[1]q
  create_admin_team = false
  child_management  = "standalone"
}

resource "crucible_player_application" "application_instance_parent" {
  view_id = crucible_player_view.application_instance_parent.id
  name    = "terminal"
}

resource "crucible_player_team" "application_instance_parent" {
  view_id = crucible_player_view.application_instance_parent.id
  name    = "students"
}
`, viewName)
}

func playerApplicationInstanceConfig(viewName string, displayOrder int) string {
	return playerApplicationInstanceParentConfig(viewName) + fmt.Sprintf(`
resource "crucible_player_application_instance" "test" {
  team_id        = crucible_player_team.application_instance_parent.id
  application_id = crucible_player_application.application_instance_parent.id
  display_order  = %[1]d
}
`, displayOrder)
}

func testAccPlayerApplicationInstanceAbsent(instanceID, teamID *string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if *instanceID == "" || *teamID == "" {
			return fmt.Errorf("player application instance identifiers were not captured")
		}
		_, exists, err := api.ReadPlayerApplicationInstance(*instanceID, *teamID, getMap())
		if err != nil {
			return fmt.Errorf("checking deleted player application instance %s: %w", *instanceID, err)
		}
		if exists {
			return fmt.Errorf("player application instance %s still exists after removal", *instanceID)
		}
		return nil
	}
}
