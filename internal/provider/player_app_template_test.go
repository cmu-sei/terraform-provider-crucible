// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/cmu-sei/terraform-provider-crucible/internal/api"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// Test case for the creation and updating of an application template resource
//
// Execution steps
// 1. Terraform calls apply
// 2. Verify local and remote states
// 3. Terraform calls apply again to update resource
// 4. Verify state
// 5. Terraform destroys resource
//
// Expected behavior:
// Resource is created, updated, and destroyed without error
func TestAccAppTemplate(t *testing.T) {
	var templateID string
	cleanupAppTemplate(t, &templateID)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccTemplateDestroyed("crucible_player_application_template.test"),
		Steps: []resource.TestStep{
			{
				Config: correctCreds + configAppTemplate,
				Check: resource.ComposeTestCheckFunc(
					captureID("crucible_player_application_template.test", "", &templateID),
					resource.TestCheckResourceAttr("crucible_player_application_template.test", "name", "TestTemplate"),
					resource.TestCheckResourceAttr("crucible_player_application_template.test", "url", "http://example.com"),
					resource.TestCheckResourceAttr("crucible_player_application_template.test", "icon", "https://upload.wikimedia.org/wikipedia/en/thumb/9/9e/Buffalo_Sabres_Logo.svg/1200px-Buffalo_Sabres_Logo.svg.png"),
					resource.TestCheckResourceAttr("crucible_player_application_template.test", "embeddable", "false"),
					resource.TestCheckResourceAttr("crucible_player_application_template.test", "load_in_background", "false"),
					verifyRemoteTemplate("TestTemplate", "http://example.com", "https://upload.wikimedia.org/wikipedia/en/thumb/9/9e/Buffalo_Sabres_Logo.svg/1200px-Buffalo_Sabres_Logo.svg.png",
						"false", "false"),
				),
			},
			{
				Config: correctCreds + configAppTemplateUpdated,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_player_application_template.test", "name", "TestTemplateUpdated"),
					resource.TestCheckResourceAttr("crucible_player_application_template.test", "url", "http://example.com"),
					resource.TestCheckResourceAttr("crucible_player_application_template.test", "icon", "https://upload.wikimedia.org/wikipedia/en/thumb/9/9e/Buffalo_Sabres_Logo.svg/1200px-Buffalo_Sabres_Logo.svg.png"),
					resource.TestCheckResourceAttr("crucible_player_application_template.test", "embeddable", "false"),
					resource.TestCheckResourceAttr("crucible_player_application_template.test", "load_in_background", "false"),
					verifyRemoteTemplate("TestTemplateUpdated", "http://example.com", "https://upload.wikimedia.org/wikipedia/en/thumb/9/9e/Buffalo_Sabres_Logo.svg/1200px-Buffalo_Sabres_Logo.svg.png",
						"false", "false"),
				),
			},
			{
				ResourceName:      "crucible_player_application_template.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccAppTemplateOmittedFieldsStable guards the Optional+Computed fields
// (url/icon/embeddable/load_in_background) that the API echoes back. The config
// sets ONLY name and leaves those fields omitted, then EDITS name on the second
// step. Editing a sibling attribute is what triggers the bug: without
// UseStateForUnknown the omitted computed fields re-plan as "known after apply".
// The plan check asserts url stays known, so a missing modifier fails the test.
// (Re-applying the SAME config would not catch it — the field must be omitted
// AND a sibling must change.)
func TestAccAppTemplateOmittedFieldsStable(t *testing.T) {
	var templateID string
	cleanupAppTemplate(t, &templateID)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccTemplateDestroyed("crucible_player_application_template.minimal"),
		Steps: []resource.TestStep{
			{
				Config: correctCreds + appTemplateMinimalConfig("acc-min-template"),
				Check:  captureID("crucible_player_application_template.minimal", "", &templateID),
			},
			{
				// Edit name (a sibling); the omitted url/icon/embeddable/
				// load_in_background must stay known, carried from state.
				Config: correctCreds + appTemplateMinimalConfig("acc-min-template-2"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectKnownValue("crucible_player_application_template.minimal",
							tfjsonpath.New("url"), knownvalue.StringExact("")),
						plancheck.ExpectKnownValue("crucible_player_application_template.minimal",
							tfjsonpath.New("icon"), knownvalue.StringExact("")),
						plancheck.ExpectKnownValue("crucible_player_application_template.minimal",
							tfjsonpath.New("embeddable"), knownvalue.Bool(false)),
						plancheck.ExpectKnownValue("crucible_player_application_template.minimal",
							tfjsonpath.New("load_in_background"), knownvalue.Bool(false)),
					},
				},
			},
		},
	})
}

func appTemplateMinimalConfig(name string) string {
	return fmt.Sprintf(`resource "crucible_player_application_template" "minimal" {
	name = "%s"
}
`, name)
}

// testAccTemplateDestroyed asserts the application template no longer exists.
//
// The Player API returns 200 with an empty body for a missing template (rather
// than 404), so AppTemplateExists can't distinguish a deleted template. Instead
// read it back: a deleted template yields an error (empty body fails to decode)
// or a zero-valued struct (empty name).
func testAccTemplateDestroyed(res string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[res]
		if !ok {
			return nil
		}
		tmpl, err := api.AppTemplateRead(rs.Primary.ID, getMap())
		if err != nil {
			return nil
		}
		if tmpl != nil && tmpl.Name != "" {
			return fmt.Errorf("template %s still exists after destroy", rs.Primary.ID)
		}
		return nil
	}
}

func verifyRemoteTemplate(name, url, icon, embeddable, load string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		mod := s.Modules[0]
		str := fmt.Sprintf("%+v", mod)
		str = strings.TrimSpace(str)

		lines := strings.Split(str, "\n")
		lines = lines[1:]
		var id string

		for _, line := range lines {
			line = strings.TrimSpace(line)

			split := strings.Split(line, " = ")

			if split[0] == "ID" {
				id = split[1]
			}
		}

		remote, err := api.AppTemplateRead(id, getMap())
		if err != nil {
			return err
		}

		if remote.Name != name {
			return fmt.Errorf("for app template, remote name %s does not equal expected name %s", remote.Name, name)
		}

		if remote.URL != url {
			return fmt.Errorf("for app template, remote url %s does not equal expected url %s", remote.URL, url)
		}

		if remote.Icon != icon {
			return fmt.Errorf("for app template, remote icon %s does not equal expected icon %s", remote.Icon, icon)
		}

		if strconv.FormatBool(remote.Embeddable) != embeddable {
			return fmt.Errorf("for app template, remote value for embeddable %v does not equal expected value for embeddable %v", remote.Embeddable, embeddable)
		}

		if strconv.FormatBool(remote.LoadInBackground) != load {
			return fmt.Errorf("for app template, remote value for load_in_background %v does not equal expected value for load_in_background %v", remote.LoadInBackground, load)
		}
		return nil
	}
}
