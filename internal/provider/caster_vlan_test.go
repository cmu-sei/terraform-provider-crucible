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

// TestAccVlan covers acquire (create), ForceNew replacement, and release
// (destroy) for the crucible_vlan resource against the Caster API.
//
// By default the test targets Caster's system-wide default partition (no
// project_id or partition_id), which the local crucible-development stack
// provisions out of the box, and acquires two specific high-numbered VLAN ids
// to drive a deterministic replacement: every configurable attribute forces
// replacement, so changing vlan_id releases the first VLAN and acquires the
// second. The second step asserts a brand-new VLAN id was acquired.
//
// Overrides for other environments:
//   - TF_TEST_PARTITION_ID : acquire from a specific partition instead of the
//     default one.
//   - TF_TEST_VLAN_ID / TF_TEST_VLAN_ID_2 : the two VLAN numbers to acquire;
//     both must exist and be free in the target partition.
//
// Note: the Caster `tag` field on acquire is a *selection filter* (it returns a
// VLAN that already carries that tag), not a label applied during acquire, so a
// tag change cannot be used to differentiate acquisitions here.
func TestAccVlan(t *testing.T) {
	partitionID := os.Getenv("TF_TEST_PARTITION_ID") // "" => system default partition
	vlanID1 := envOrDefault("TF_TEST_VLAN_ID", "1000")
	vlanID2 := envOrDefault("TF_TEST_VLAN_ID_2", "1001")

	var firstID, secondID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: correctCreds + vlanConfig(partitionID, vlanID1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_vlan.test", "vlan_id", vlanID1),
					resource.TestCheckResourceAttrSet("crucible_vlan.test", "partition_id"),
					testAccVlanRemoteInUse("crucible_vlan.test", &firstID),
				),
			},
			{
				// Changing vlan_id forces replacement; capture the new id and
				// assert it differs from the first.
				Config: correctCreds + vlanConfig(partitionID, vlanID2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_vlan.test", "vlan_id", vlanID2),
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

// TestAccVlanByProject exercises the by-project acquire path: Caster resolves
// the partition assigned to the given project and acquires a VLAN from it.
//
// This is opt-in (gated on TF_TEST_PROJECT_ID) because the standard
// crucible-development projects have no assigned VLAN partition, so acquiring by
// project there returns 404 "Partition not found". Supply a project GUID whose
// project has been assigned a partition to run it. No vlan_id is requested, so
// Caster takes any free VLAN from the resolved partition.
func TestAccVlanByProject(t *testing.T) {
	projectID := os.Getenv("TF_TEST_PROJECT_ID")
	if projectID == "" {
		t.Skip("TF_TEST_PROJECT_ID not set; skipping by-project crucible_vlan acceptance test")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: correctCreds + vlanByProjectConfig(projectID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_vlan.test", "project_id", projectID),
					resource.TestCheckResourceAttrSet("crucible_vlan.test", "vlan_id"),
					resource.TestCheckResourceAttrSet("crucible_vlan.test", "partition_id"),
					testAccVlanRemoteInUse("crucible_vlan.test", new(string)),
				),
			},
		},
	})
}

// vlanByProjectConfig builds a crucible_vlan resource that acquires from the
// partition assigned to the given project. Only project_id is set; partition_id
// and vlan_id are omitted (project_id conflicts with partition_id, and pinning a
// vlan_id may select one that is not free in the project's partition).
func vlanByProjectConfig(projectID string) string {
	return fmt.Sprintf(`resource "crucible_vlan" "test" {
	project_id = %q
}
`, projectID)
}

// vlanConfig builds a crucible_vlan resource that acquires a specific vlan_id.
// When partitionID is empty the partition_id attribute is omitted so Caster
// falls back to its system-wide default partition.
func vlanConfig(partitionID, vlanID string) string {
	partitionLine := ""
	if partitionID != "" {
		partitionLine = fmt.Sprintf("partition_id = %q\n\t", partitionID)
	}
	return fmt.Sprintf(`resource "crucible_vlan" "test" {
	%svlan_id = %s
}
`, partitionLine, vlanID)
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
