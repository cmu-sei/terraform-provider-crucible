// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccAppTemplateDataSource verifies the crucible_player_application_template
// data source resolves a template by name to the same id as the backing
// resource. The data source references the resource's name, so Terraform creates
// the template first and the data source reads it back in the same apply.
//
// It also exercises the case_insensitive flag: a second data source looks the
// template up by an upper-cased name and must resolve to the same id. (A default,
// case-sensitive lookup of a mismatched case would error; that path is covered by
// the dedicated not-found test below.)
func TestAccAppTemplateDataSource(t *testing.T) {
	var templateID string
	registerAppTemplateCleanup(t, &templateID)

	const name = "acc-ds-template"
	config := fmt.Sprintf(`
resource "crucible_player_application_template" "test" {
	name = "%s"
}

data "crucible_player_application_template" "by_name" {
	name = crucible_player_application_template.test.name
}

data "crucible_player_application_template" "by_name_ci" {
	name             = upper(crucible_player_application_template.test.name)
	case_insensitive = true
}
`, name)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccTemplateDestroyed("crucible_player_application_template.test"),
		Steps: []resource.TestStep{
			{
				Config: tfConfig(config),
				Check: resource.ComposeTestCheckFunc(
					captureID("crucible_player_application_template.test", &templateID),
					// Exact, case-sensitive lookup resolves to the backing resource's id.
					resource.TestCheckResourceAttrPair(
						"data.crucible_player_application_template.by_name", "id",
						"crucible_player_application_template.test", "id"),
					resource.TestCheckResourceAttr(
						"data.crucible_player_application_template.by_name", "name", name),
					resource.TestCheckResourceAttr(
						"data.crucible_player_application_template.by_name", "embeddable", "false"),
					resource.TestCheckResourceAttr(
						"data.crucible_player_application_template.by_name", "load_in_background", "false"),
					// Case-insensitive lookup of an upper-cased name resolves to the same id.
					resource.TestCheckResourceAttrPair(
						"data.crucible_player_application_template.by_name_ci", "id",
						"crucible_player_application_template.test", "id"),
				),
			},
		},
	})
}

// TestAccAppTemplateDataSourceNotFound asserts the data source errors (rather
// than returning empty state) when no template matches the given name. No
// backing resource is created, so there is nothing to clean up.
func TestAccAppTemplateDataSourceNotFound(t *testing.T) {
	const config = `
data "crucible_player_application_template" "missing" {
	name = "acc-ds-template-does-not-exist"
}
`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      tfConfig(config),
				ExpectError: regexp.MustCompile("no application template found"),
			},
		},
	})
}
