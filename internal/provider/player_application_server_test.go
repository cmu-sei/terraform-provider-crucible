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

func TestAccPlayerApplication(t *testing.T) {
	const viewName = "tf-acc-player-application"
	var applicationID string
	sweepViewByName(t, viewName)
	registerViewCleanupByName(t, viewName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccViewDestroyed,
		Steps: []resource.TestStep{
			{
				Config: tfConfig(playerApplicationConfig(viewName, false)),
				Check: resource.ComposeAggregateTestCheckFunc(
					captureID("crucible_player_application.test", &applicationID),
					resource.TestCheckResourceAttrSet("crucible_player_application.test", "id"),
					resource.TestCheckResourceAttr("crucible_player_application.test", "name", "terminal"),
				),
			},
			{
				Config: tfConfig(playerApplicationConfig(viewName, true)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_player_application.test", "name", "terminal-updated"),
					resource.TestCheckResourceAttr("crucible_player_application.test", "url", "https://terminal.example.test"),
					resource.TestCheckResourceAttr("crucible_player_application.test", "icon", "terminal"),
					resource.TestCheckResourceAttr("crucible_player_application.test", "embeddable", "true"),
					resource.TestCheckResourceAttr("crucible_player_application.test", "load_in_background", "true"),
				),
			},
			{
				Config: tfConfig(playerApplicationConfig(viewName, true)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				ResourceName:      "crucible_player_application.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: tfConfig(playerApplicationParentConfig(viewName)),
				Check:  testAccPlayerApplicationAbsent(&applicationID),
			},
		},
	})
}

func playerApplicationParentConfig(viewName string) string {
	return fmt.Sprintf(`
resource "crucible_player_view" "application_parent" {
  name              = %[1]q
  create_admin_team = false
  child_management  = "standalone"
}
`, viewName)
}

func playerApplicationConfig(viewName string, updated bool) string {
	application := `
resource "crucible_player_application" "test" {
  view_id = crucible_player_view.application_parent.id
  name    = "terminal"
}
`
	if updated {
		application = `
resource "crucible_player_application" "test" {
  view_id            = crucible_player_view.application_parent.id
  name               = "terminal-updated"
  url                = "https://terminal.example.test"
  icon               = "terminal"
  embeddable         = true
  load_in_background = true
}
`
	}
	return playerApplicationParentConfig(viewName) + application
}

func testAccPlayerApplicationAbsent(id *string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if *id == "" {
			return fmt.Errorf("player application id was not captured")
		}
		_, exists, err := api.ReadPlayerApplication(*id, getMap())
		if err != nil {
			return fmt.Errorf("checking deleted player application %s: %w", *id, err)
		}
		if exists {
			return fmt.Errorf("player application %s still exists after removal", *id)
		}
		return nil
	}
}
