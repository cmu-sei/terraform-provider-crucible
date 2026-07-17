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
				ResourceName:      "crucible_player_application_instance.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(*terraform.State) (string, error) {
					return teamID + "/" + instanceID, nil
				},
			},
			{
				Config: tfConfig(playerApplicationInstanceConfig(viewName, 0.1)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"crucible_player_application_instance.test",
						"display_order",
						"0.1",
					),
				),
			},
			{
				Config: tfConfig(playerApplicationInstanceConfig(viewName, 0.1)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
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

func playerApplicationInstanceConfig(viewName string, displayOrder float64) string {
	return playerApplicationInstanceParentConfig(viewName) + fmt.Sprintf(`
resource "crucible_player_application_instance" "test" {
  team_id        = crucible_player_team.application_instance_parent.id
  application_id = crucible_player_application.application_instance_parent.id
  display_order  = %[1]g
}
`, displayOrder)
}

func TestAccPlayerApplicationInstanceRejectsCrossViewCreate(t *testing.T) {
	const name = "tf-acc-application-instance-cross-view-create"
	sweepViewByName(t, name+"-application")
	sweepViewByName(t, name+"-team")
	registerViewCleanupByName(t, name+"-application")
	registerViewCleanupByName(t, name+"-team")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      tfConfig(playerApplicationInstanceCrossViewConfig(name)),
			ExpectError: regexp.MustCompile(`player API returned with status 4[0-9]{2} when adding application to team`),
		}},
	})
}

func TestAccPlayerApplicationInstanceRejectsCrossViewUpdate(t *testing.T) {
	const name = "tf-acc-application-instance-cross-view-update"
	sweepViewByName(t, name+"-one")
	sweepViewByName(t, name+"-two")
	registerViewCleanupByName(t, name+"-one")
	registerViewCleanupByName(t, name+"-two")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccViewDestroyed,
		Steps: []resource.TestStep{{
			Config: tfConfig(playerApplicationInstanceCrossViewUpdateConfig(name)),
			Check:  testAccPlayerApplicationInstanceCrossViewUpdateRejected,
		}},
	})
}

func playerApplicationInstanceCrossViewConfig(name string) string {
	return fmt.Sprintf(`
resource "crucible_player_view" "application" {
  name              = %[1]q
  create_admin_team = false
  child_management  = "standalone"
}
resource "crucible_player_view" "team" {
  name              = %[2]q
  create_admin_team = false
  child_management  = "standalone"
}
resource "crucible_player_application" "cross_view" {
  view_id = crucible_player_view.application.id
  name    = "cross-view"
}
resource "crucible_player_team" "cross_view" {
  view_id = crucible_player_view.team.id
  name    = "cross-view"
}
resource "crucible_player_application_instance" "cross_view" {
  team_id        = crucible_player_team.cross_view.id
  application_id = crucible_player_application.cross_view.id
}
`, name+"-application", name+"-team")
}

func playerApplicationInstanceCrossViewUpdateConfig(name string) string {
	return fmt.Sprintf(`
resource "crucible_player_view" "one" {
  name              = %[1]q
  create_admin_team = false
  child_management  = "standalone"
}
resource "crucible_player_view" "two" {
  name              = %[2]q
  create_admin_team = false
  child_management  = "standalone"
}
resource "crucible_player_application" "one" {
  view_id = crucible_player_view.one.id
  name    = "one"
}
resource "crucible_player_application" "two" {
  view_id = crucible_player_view.two.id
  name    = "two"
}
resource "crucible_player_team" "one" {
  view_id = crucible_player_view.one.id
  name    = "one"
}
resource "crucible_player_application_instance" "test" {
  team_id        = crucible_player_team.one.id
  application_id = crucible_player_application.one.id
}
`, name+"-one", name+"-two")
}

func testAccPlayerApplicationInstanceCrossViewUpdateRejected(s *terraform.State) error {
	instance := s.RootModule().Resources["crucible_player_application_instance.test"].Primary
	team := s.RootModule().Resources["crucible_player_team.one"].Primary
	originalApplication := s.RootModule().Resources["crucible_player_application.one"].Primary
	crossViewApplication := s.RootModule().Resources["crucible_player_application.two"].Primary

	err := api.UpdatePlayerApplicationInstance(&api.PlayerApplicationInstance{
		ID:            instance.ID,
		TeamID:        team.ID,
		ApplicationID: crossViewApplication.ID,
		DisplayOrder:  0,
	}, getMap())
	if err == nil {
		return fmt.Errorf("cross-view application instance update unexpectedly succeeded")
	}

	actual, exists, readErr := api.ReadPlayerApplicationInstance(instance.ID, team.ID, getMap())
	if readErr != nil {
		return readErr
	}
	if !exists {
		return fmt.Errorf("application instance %s disappeared after rejected update", instance.ID)
	}
	if actual.ApplicationID != originalApplication.ID {
		return fmt.Errorf("application instance application_id = %s, want %s", actual.ApplicationID, originalApplication.ID)
	}
	return nil
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
