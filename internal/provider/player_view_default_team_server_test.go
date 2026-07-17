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

func TestAccPlayerViewDefaultTeam(t *testing.T) {
	const (
		viewName     = "tf-acc-player-view-default-team"
		resourceName = "crucible_player_view_default_team.test"
	)
	sweepViewByName(t, viewName)
	registerViewCleanupByName(t, viewName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccViewDestroyed,
		Steps: []resource.TestStep{
			{
				Config: tfConfig(playerViewDefaultTeamConfig(viewName, "red", true)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrPair(resourceName, "id", "crucible_player_view.default_parent", "id"),
					resource.TestCheckResourceAttrPair(resourceName, "view_id", "crucible_player_view.default_parent", "id"),
					resource.TestCheckResourceAttrPair(resourceName, "team_id", "crucible_player_team.red", "id"),
					testAccVerifyPlayerViewDefaultTeam("crucible_player_team.red"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: tfConfig(playerViewDefaultTeamConfig(viewName, "blue", true)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_player_view.default_parent", "description", "default team blue"),
					resource.TestCheckResourceAttr("crucible_player_view.default_parent", "is_template", "true"),
					resource.TestCheckResourceAttrPair(resourceName, "team_id", "crucible_player_team.blue", "id"),
					testAccVerifyPlayerViewDefaultTeam("crucible_player_team.blue"),
				),
			},
			{
				Config: tfConfig(playerViewDefaultTeamConfig(viewName, "blue", true)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: tfConfig(playerViewDefaultTeamConfig(viewName, "", false)),
				Check:  testAccVerifyNoPlayerViewDefaultTeam,
			},
		},
	})
}

func TestAccPlayerViewDefaultTeamRejectsDifferentView(t *testing.T) {
	const viewName = "tf-acc-default-team-wrong-view"
	sweepViewByName(t, viewName+"-one")
	sweepViewByName(t, viewName+"-two")
	registerViewCleanupByName(t, viewName+"-one")
	registerViewCleanupByName(t, viewName+"-two")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      tfConfig(playerViewDefaultTeamWrongViewConfig(viewName)),
			ExpectError: regexp.MustCompile(`(?s)belongs to view.*not view`),
		}},
	})
}

func TestAccPlayerViewDefaultTeamRejectsDuplicate(t *testing.T) {
	const viewName = "tf-acc-default-team-duplicate"
	sweepViewByName(t, viewName)
	registerViewCleanupByName(t, viewName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      tfConfig(playerViewDefaultTeamDuplicateConfig(viewName)),
			ExpectError: regexp.MustCompile(`Player view already has a default team`),
		}},
	})
}

func playerViewDefaultTeamConfig(viewName, selected string, includeAssociation bool) string {
	description := "no default team"
	isTemplate := false
	if selected != "" {
		description = "default team " + selected
		isTemplate = selected == "blue"
	}
	config := fmt.Sprintf(`
resource "crucible_player_view" "default_parent" {
  name              = %[1]q
  description       = %[2]q
  create_admin_team = false
  child_management  = "standalone"
  is_template       = %[3]t
}

resource "crucible_player_team" "red" {
  view_id = crucible_player_view.default_parent.id
  name    = "red"
}

resource "crucible_player_team" "blue" {
  view_id = crucible_player_view.default_parent.id
  name    = "blue"
}
`, viewName, description, isTemplate)
	if includeAssociation {
		config += fmt.Sprintf(`
resource "crucible_player_view_default_team" "test" {
  view_id = crucible_player_view.default_parent.id
  team_id = crucible_player_team.%[1]s.id
}
`, selected)
	}
	return config
}

func playerViewDefaultTeamWrongViewConfig(name string) string {
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
resource "crucible_player_team" "one" {
  view_id = crucible_player_view.one.id
  name    = "one"
}
resource "crucible_player_view_default_team" "invalid" {
  view_id = crucible_player_view.two.id
  team_id = crucible_player_team.one.id
}
`, name+"-one", name+"-two")
}

func playerViewDefaultTeamDuplicateConfig(name string) string {
	return playerViewDefaultTeamConfig(name, "", false) + `
resource "crucible_player_view_default_team" "red" {
  view_id = crucible_player_view.default_parent.id
  team_id = crucible_player_team.red.id
}
resource "crucible_player_view_default_team" "blue" {
  view_id = crucible_player_view.default_parent.id
  team_id = crucible_player_team.blue.id
}
`
}

func testAccVerifyPlayerViewDefaultTeam(teamResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		view := s.RootModule().Resources["crucible_player_view.default_parent"]
		team := s.RootModule().Resources[teamResource]
		remote, err := api.ReadViewTopLevel(view.Primary.ID, getMap())
		if err != nil {
			return err
		}
		if remote.DefaultTeamID != team.Primary.ID {
			return fmt.Errorf("default team id = %q, want %q", remote.DefaultTeamID, team.Primary.ID)
		}
		return nil
	}
}

func testAccVerifyNoPlayerViewDefaultTeam(s *terraform.State) error {
	view := s.RootModule().Resources["crucible_player_view.default_parent"]
	remote, err := api.ReadViewTopLevel(view.Primary.ID, getMap())
	if err != nil {
		return err
	}
	if remote.DefaultTeamID != "" {
		return fmt.Errorf("default team id = %q, want empty", remote.DefaultTeamID)
	}
	return nil
}
