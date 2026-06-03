// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/cmu-sei/terraform-provider-crucible/internal/api"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccVlan covers acquire (create), tag-change replacement, and release
// (destroy) for the crucible_vlan resource.
//
// A VLAN is acquired from a Caster partition/project pool, so the test requires
// a real project id via TF_TEST_PROJECT_ID and the Caster API (TF_CASTER_API_URL).
// Every configurable attribute forces replacement, so the second step (changed
// tag) asserts a brand-new VLAN id was acquired.
func TestAccVlan(t *testing.T) {
	projectID := os.Getenv("TF_TEST_PROJECT_ID")
	if projectID == "" {
		t.Skip("TF_TEST_PROJECT_ID not set; skipping crucible_vlan acceptance test")
	}

	var firstID, secondID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: correctCreds + vlanConfig(projectID, "acc-test"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_vlan.test", "project_id", projectID),
					resource.TestCheckResourceAttr("crucible_vlan.test", "tag", "acc-test"),
					resource.TestCheckResourceAttrSet("crucible_vlan.test", "vlan_id"),
					testAccVlanRemoteInUse("crucible_vlan.test", &firstID),
				),
			},
			{
				// Changing the tag forces replacement; capture the new id and
				// assert it differs from the first.
				Config: correctCreds + vlanConfig(projectID, "acc-test-2"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_vlan.test", "tag", "acc-test-2"),
					testAccVlanRemoteInUse("crucible_vlan.test", &secondID),
					func(_ *terraform.State) error {
						if firstID != "" && firstID == secondID {
							return fmt.Errorf("expected vlan to be replaced, but id %s was reused", firstID)
						}
						return nil
					},
				),
			},
		},
	})
}

func vlanConfig(projectID, tag string) string {
	return fmt.Sprintf(`resource "crucible_vlan" "test" {
		project_id = "%s"
		tag        = "%s"
	}
	`, projectID, tag)
}

// testAccVlanRemoteInUse verifies the VLAN exists and is in use in Caster, and
// records its id into out for cross-step comparison.
func testAccVlanRemoteInUse(res string, out *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[res]
		if !ok {
			return fmt.Errorf("resource %s not found in state", res)
		}
		id := rs.Primary.ID
		*out = id

		vlan, err := api.ReadVlan(id, getMap())
		if err != nil {
			return err
		}
		if !vlan.InUse {
			return fmt.Errorf("expected vlan %s to be in use", id)
		}
		return nil
	}
}
