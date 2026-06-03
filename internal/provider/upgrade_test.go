// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// oldProviderVersion is the last published SDKv1 release on the Terraform
// Registry. Step 1 of each upgrade test creates resources with this real binary;
// step 2 swaps to the in-process Plugin Framework build and asserts no drift.
const oldProviderVersion = "2.5.0"

// upgradeProviderEnabled reports whether the upgrade tests should run. They pull
// the old provider from the Terraform Registry, so they need network access in
// addition to the live Crucible APIs; gate them behind TF_TEST_UPGRADE=1.
func upgradeProviderEnabled(t *testing.T) {
	if os.Getenv("TF_TEST_UPGRADE") != "1" {
		t.Skip("TF_TEST_UPGRADE != 1; skipping old->new provider upgrade test")
	}
}

// externalOldProvider returns the ExternalProviders map pinning the old
// published provider for the create step.
func externalOldProvider() map[string]resource.ExternalProvider {
	return map[string]resource.ExternalProvider{
		"crucible": {
			Source:            "cmu-sei/crucible",
			VersionConstraint: oldProviderVersion,
		},
	}
}

// TestAccUpgradeView verifies that a crucible_player_view created by the old
// SDKv1 provider plans cleanly (empty plan) under the new Framework provider.
func TestAccUpgradeView(t *testing.T) {
	upgradeProviderEnabled(t)

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				// Create with the old published provider.
				ExternalProviders: externalOldProvider(),
				Config:            correctCreds + configViewEmpty,
			},
			{
				// Switch to the new dev provider; expect no planned changes.
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   correctCreds + configViewEmpty,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// TestAccUpgradeAppTemplate verifies the same empty-plan upgrade guarantee for
// crucible_player_application_template.
func TestAccUpgradeAppTemplate(t *testing.T) {
	upgradeProviderEnabled(t)

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: externalOldProvider(),
				Config:            correctCreds + configAppTemplate,
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   correctCreds + configAppTemplate,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}
