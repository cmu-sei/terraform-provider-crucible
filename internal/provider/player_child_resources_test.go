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

func TestAccViewStandaloneValidation(t *testing.T) {
	tests := map[string]string{
		"application": `
  create_admin_team = false
  application { name = "invalid" }`,
		"team": `
  create_admin_team = false
  team { name = "invalid" }`,
		"admin": `
  create_admin_team = true`,
		"all": `
  create_admin_team = true
  application { name = "invalid" }
  team { name = "invalid" }`,
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{{
					Config: tfConfig(fmt.Sprintf(`
resource "crucible_player_view" "invalid" {
  name             = "tf-acc-invalid-standalone"
  child_management = "standalone"
%s
}`, body)),
					PlanOnly:    true,
					ExpectError: regexp.MustCompile(`Invalid child ownership configuration`),
				}},
			})
		})
	}
}

func TestAccViewSeparateModeRejected(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: tfConfig(`
resource "crucible_player_view" "invalid" {
  name              = "tf-acc-invalid-separate-mode"
  create_admin_team = false
  child_management  = "separate"
}
`),
			PlanOnly:    true,
			ExpectError: regexp.MustCompile(`Invalid Attribute Value Match`),
		}},
	})
}

func TestAccStandalonePlayerChildren(t *testing.T) {
	viewName := "tf-acc-standalone-children"
	userID := envOrDefault("TF_TEST_VIEW_USER_ID", "9b3b331c-10c1-448b-8114-21b2586d8e38")
	sweepViewByName(t, viewName)
	registerViewCleanupByName(t, viewName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccViewDestroyed,
		Steps: []resource.TestStep{
			{
				Config: tfConfig(standaloneChildrenConfig(viewName, userID, false)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_player_view.standalone", "child_management", "standalone"),
					resource.TestCheckResourceAttr("crucible_player_application.terminal", "name", "terminal"),
					resource.TestCheckResourceAttrSet("crucible_player_application.terminal", "id"),
					resource.TestCheckResourceAttr("crucible_player_team.students", "permissions.#", "0"),
					resource.TestCheckResourceAttr("crucible_player_team.students", "scoped_team_ids.#", "0"),
					resource.TestCheckNoResourceAttr("crucible_player_team_user.student", "view_id"),
					resource.TestCheckNoResourceAttr("crucible_player_team_user.student", "role"),
					resource.TestCheckResourceAttr("crucible_player_application_instance.terminal", "display_order", "0"),
				),
			},
			{
				Config: tfConfig(standaloneChildrenConfig(viewName, userID, true)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_player_application.terminal", "name", "terminal-updated"),
					resource.TestCheckResourceAttr("crucible_player_team.students", "name", "students-updated"),
					resource.TestCheckResourceAttr("crucible_player_team_user.student", "role", "View Member"),
					resource.TestCheckResourceAttr("crucible_player_application_instance.terminal", "display_order", "2"),
				),
			},
			{
				Config: tfConfig(standaloneChildrenConfig(viewName, userID, true)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				ResourceName:      "crucible_player_application.terminal",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "crucible_player_team.students",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "crucible_player_team_user.student",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "crucible_player_application_instance.terminal",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					instance := s.RootModule().Resources["crucible_player_application_instance.terminal"]
					team := s.RootModule().Resources["crucible_player_team.students"]
					return team.Primary.ID + "/" + instance.Primary.ID, nil
				},
			},
		},
	})
}

func TestAccStandaloneDefaultTeam(t *testing.T) {
	const viewName = "tf-acc-standalone-default-team"
	sweepViewByName(t, viewName)
	registerViewCleanupByName(t, viewName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccViewDestroyed,
		Steps: []resource.TestStep{
			{
				Config: tfConfig(standaloneDefaultTeamConfig(viewName, true)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_player_team.default_team", "default", "true"),
					testAccVerifyStandaloneDefaultTeam(true),
				),
			},
			{
				ResourceName:      "crucible_player_team.default_team",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: tfConfig(standaloneDefaultTeamConfig(viewName, false)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_player_team.default_team", "default", "false"),
					testAccVerifyStandaloneDefaultTeam(false),
				),
			},
		},
	})
}

func standaloneDefaultTeamConfig(viewName string, defaultTeam bool) string {
	return fmt.Sprintf(`
resource "crucible_player_view" "standalone_default" {
  name              = %[1]q
  create_admin_team = false
  child_management  = "standalone"
}

resource "crucible_player_team" "default_team" {
  view_id = crucible_player_view.standalone_default.id
  name    = "default-team"
  default = %[2]t
}
`, viewName, defaultTeam)
}

func testAccVerifyStandaloneDefaultTeam(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		view := s.RootModule().Resources["crucible_player_view.standalone_default"]
		team := s.RootModule().Resources["crucible_player_team.default_team"]
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

func standaloneChildrenConfig(viewName, userID string, updated bool) string {
	appName, teamName, role, order := "terminal", "students", "", 0
	if updated {
		appName, teamName, role, order = "terminal-updated", "students-updated", "\n  role = \"View Member\"", 2
	}
	return fmt.Sprintf(`
resource "crucible_player_view" "standalone" {
  name              = %[1]q
  description       = "standalone children"
  create_admin_team = false
  child_management  = "standalone"
}

resource "crucible_player_application" "terminal" {
  view_id            = crucible_player_view.standalone.id
  name               = %[2]q
  url                = "https://terminal.example.test"
  embeddable         = false
  load_in_background = false
}

resource "crucible_player_team" "students" {
  view_id = crucible_player_view.standalone.id
  name    = %[3]q
}

resource "crucible_player_team_user" "student" {
  team_id = crucible_player_team.students.id
  user_id = %[4]q%[5]s
}

resource "crucible_player_application_instance" "terminal" {
  team_id        = crucible_player_team.students.id
  application_id = crucible_player_application.terminal.id
  display_order  = %[6]d
}
`, viewName, appName, teamName, userID, role, order)
}

func TestAccStandaloneViewIgnoresUnmanagedChildren(t *testing.T) {
	viewName := "tf-acc-standalone-unmanaged"
	var unmanagedID string
	sweepViewByName(t, viewName)
	registerViewCleanupByName(t, viewName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccViewDestroyed,
		Steps: []resource.TestStep{
			{
				Config: tfConfig(standaloneViewOnlyConfig(viewName, "before")),
				Check: func(s *terraform.State) error {
					viewID := s.RootModule().Resources["crucible_player_view.standalone"].Primary.ID
					app := &api.PlayerApplication{ViewID: viewID, Name: "unmanaged"}
					if err := api.CreatePlayerApplication(app, getMap()); err != nil {
						return err
					}
					unmanagedID = app.ID
					return nil
				},
				ExpectNonEmptyPlan: false,
			},
			{
				Config: tfConfig(standaloneViewOnlyConfig(viewName, "after")),
				Check: func(*terraform.State) error {
					_, exists, err := api.ReadPlayerApplication(unmanagedID, getMap())
					if err != nil {
						return err
					}
					if !exists {
						return fmt.Errorf("standalone view deleted unmanaged application %s", unmanagedID)
					}
					return nil
				},
			},
		},
	})
}

func standaloneViewOnlyConfig(name, description string) string {
	return fmt.Sprintf(`
resource "crucible_player_view" "standalone" {
  name              = %q
  description       = %q
  create_admin_team = false
  child_management  = "standalone"
}
`, name, description)
}

func TestAccStandaloneApplicationsForEachPlan(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: tfConfig(`
resource "crucible_player_view" "standalone" {
  name              = "tf-acc-standalone-foreach-plan"
  create_admin_team = false
  child_management  = "standalone"
}
resource "crucible_player_application" "apps" {
  for_each = toset(["one", "two"])
  view_id  = crucible_player_view.standalone.id
  name     = each.key
}
`),
			PlanOnly:           true,
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccInlineToStandaloneMigration(t *testing.T) {
	viewName := "tf-acc-child-migration"
	sweepViewByName(t, viewName)
	registerViewCleanupByName(t, viewName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccViewDestroyed,
		Steps: []resource.TestStep{
			{
				Config: tfConfig(inlineMigrationConfig(viewName)),
				Check:  testAccMigrationInlineChildrenExist,
			},
			{
				Config: tfConfig(migrationImportConfig(viewName)),
				Check:  testAccMigrationStandaloneMatchesRemote,
			},
			{
				Config: tfConfig(migrationStandaloneConfig(viewName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_player_view.migration", "child_management", "standalone"),
					resource.TestCheckResourceAttr("crucible_player_view.migration", "application.#", "0"),
					resource.TestCheckResourceAttr("crucible_player_view.migration", "team.#", "0"),
					testAccMigrationStandaloneMatchesRemote,
				),
			},
		},
	})
}

func testAccMigrationInlineChildrenExist(s *terraform.State) error {
	viewID := s.RootModule().Resources["crucible_player_view.migration"].Primary.ID
	view, err := api.ReadView(viewID, getMap())
	if err != nil {
		return err
	}
	if len(view.Applications) != 1 || view.Applications[0].ID == "" || len(view.Teams) != 1 || ifaceString(view.Teams[0].ID) == "" {
		return fmt.Errorf("inline child ids were not populated")
	}
	return nil
}

func testAccMigrationStandaloneMatchesRemote(s *terraform.State) error {
	viewID := s.RootModule().Resources["crucible_player_view.migration"].Primary.ID
	view, err := api.ReadView(viewID, getMap())
	if err != nil {
		return err
	}
	if len(view.Applications) != 1 || len(view.Teams) != 1 {
		return fmt.Errorf("migration changed remote child counts: %d applications, %d teams", len(view.Applications), len(view.Teams))
	}
	appState := s.RootModule().Resources["crucible_player_application.imported"].Primary.ID
	teamState := s.RootModule().Resources["crucible_player_team.imported"].Primary.ID
	if view.Applications[0].ID != appState || ifaceString(view.Teams[0].ID) != teamState {
		return fmt.Errorf("migration replaced remote children")
	}
	return nil
}

func inlineMigrationConfig(name string) string {
	return fmt.Sprintf(`
resource "crucible_player_view" "migration" {
  name              = %q
  create_admin_team = false
  application { name = "migrated-app" }
  team { name = "migrated-team" }
}
`, name)
}

func migrationImportConfig(name string) string {
	return inlineMigrationConfig(name) + `
resource "crucible_player_application" "imported" {
  view_id = crucible_player_view.migration.id
  name    = "migrated-app"
}
resource "crucible_player_team" "imported" {
  view_id = crucible_player_view.migration.id
  name    = "migrated-team"
}
import {
  to = crucible_player_application.imported
  id = crucible_player_view.migration.application[0].app_id
}
import {
  to = crucible_player_team.imported
  id = crucible_player_view.migration.team[0].team_id
}
`
}

func migrationStandaloneConfig(name string) string {
	return fmt.Sprintf(`
resource "crucible_player_view" "migration" {
  name              = %q
  create_admin_team = false
  child_management  = "standalone"
}
resource "crucible_player_application" "imported" {
  view_id = crucible_player_view.migration.id
  name    = "migrated-app"
}
resource "crucible_player_team" "imported" {
  view_id = crucible_player_view.migration.id
  name    = "migrated-team"
}
`, name)
}
