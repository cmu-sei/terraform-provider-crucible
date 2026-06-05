// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

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
//
// Even when enabled, the old provider download can flake (registry/GitHub
// release unreachable). A flaky download must not fail the suite, so we probe
// once whether the old provider can be obtained and t.Skip if not — real drift
// is still tested whenever the provider is reachable.
func upgradeProviderEnabled(t *testing.T) {
	if os.Getenv("TF_TEST_UPGRADE") != "1" {
		t.Skip("TF_TEST_UPGRADE != 1; skipping old->new provider upgrade test")
	}
	if reason := oldProviderUnavailable(); reason != "" {
		t.Skipf("skipping upgrade test: old provider %s unavailable: %s", oldProviderVersion, reason)
	}
}

// oldProviderProbe memoizes a one-time check that the old published provider can
// be downloaded, so a registry/network blip skips the upgrade suite instead of
// failing it. Empty string means available.
var (
	oldProviderProbeOnce   sync.Once
	oldProviderProbeReason string
)

func oldProviderUnavailable() string {
	oldProviderProbeOnce.Do(func() {
		tf, err := exec.LookPath("terraform")
		if err != nil {
			oldProviderProbeReason = "terraform binary not found on PATH"
			return
		}
		dir, err := os.MkdirTemp("", "crucible-upgrade-probe-")
		if err != nil {
			oldProviderProbeReason = "could not create temp probe dir: " + err.Error()
			return
		}
		defer os.RemoveAll(dir)

		cfg := fmt.Sprintf(`terraform {
  required_providers {
    crucible = {
      source  = "cmu-sei/crucible"
      version = "%s"
    }
  }
}
`, oldProviderVersion)
		if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(cfg), 0o644); err != nil {
			oldProviderProbeReason = "could not write probe config: " + err.Error()
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		// -backend=false keeps this to just provider installation. TF_PLUGIN_CACHE_DIR
		// (set by `task testacc`) makes this a no-op once the provider is cached.
		cmd := exec.CommandContext(ctx, tf, "init", "-backend=false", "-input=false", "-no-color")
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			oldProviderProbeReason = fmt.Sprintf("terraform init could not install it: %v\n%s", err, out)
		}
	})
	return oldProviderProbeReason
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

// upgradeSteps builds the standard two-step upgrade test for a single config:
// step 1 creates the resource(s) with the old published SDKv1 provider, step 2
// switches to the in-process Framework build and asserts an empty plan. The
// config must be byte-identical and parse under both provider versions.
func upgradeSteps(config string) []resource.TestStep {
	return []resource.TestStep{
		{
			ExternalProviders: externalOldProvider(),
			Config:            config,
		},
		{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Config:                   config,
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectEmptyPlan(),
				},
			},
		},
	}
}

// TestAccUpgradeViewWithTeams enriches the view upgrade coverage beyond the bare
// configViewEmpty: it exercises team role/permissions and the nested
// app_instance/user blocks across the SDKv1->Framework migration.
func TestAccUpgradeViewWithTeams(t *testing.T) {
	upgradeProviderEnabled(t)
	sweepViewByName(t, "test")
	cleanupViewByName(t, "test")

	resource.Test(t, resource.TestCase{
		CheckDestroy: testAccViewDestroyed,
		Steps:        upgradeSteps(correctCreds + configViewTeams),
	})
}

// TestAccUpgradeVMConsole upgrades a VM with a console_connection_info block,
// exercising the url-token read path across provider versions.
func TestAccUpgradeVMConsole(t *testing.T) {
	upgradeProviderEnabled(t)

	const vmID = "c2e4d3a5-2222-4bbb-9ccc-000000000001"
	cleanupVM(t, vmID)
	sweepVM(t, vmID)
	resource.Test(t, resource.TestCase{
		CheckDestroy: testAccVMDestroyed(vmID),
		Steps:        upgradeSteps(correctCreds + vmConsoleConfig(vmID, "https://example.com/console")),
	})
}

// TestAccUpgradeVMProxmox upgrades a VM with a proxmox_vm_info block.
func TestAccUpgradeVMProxmox(t *testing.T) {
	upgradeProviderEnabled(t)

	const vmID = "c2e4d3a5-2222-4bbb-9ccc-000000000002"
	// Test-specific proxmox id: it is a unique server-side PK, so avoid colliding
	// with other fixtures / test/main.tf.
	sweepVM(t, vmID)
	cleanupVM(t, vmID)
	resource.Test(t, resource.TestCase{
		CheckDestroy: testAccVMDestroyed(vmID),
		Steps:        upgradeSteps(correctCreds + vmProxmoxConfig(vmID, "https://example.com/proxmox", "990010")),
	})
}

// TestAccUpgradeVMMultiTeam upgrades a VM in two teams, exercising team_ids
// ordering across provider versions.
func TestAccUpgradeVMMultiTeam(t *testing.T) {
	upgradeProviderEnabled(t)

	const vmID = "c2e4d3a5-2222-4bbb-9ccc-000000000003"
	sweepVM(t, vmID)
	cleanupVM(t, vmID)
	resource.Test(t, resource.TestCase{
		CheckDestroy: testAccVMDestroyed(vmID),
		Steps:        upgradeSteps(correctCreds + vmMultiTeamConfig(vmID, "https://example.com/multiteam")),
	})
}

// TestAccUpgradeUser upgrades a crucible_player_user with a role set.
func TestAccUpgradeUser(t *testing.T) {
	upgradeProviderEnabled(t)

	userID := testEnv("TF_TEST_USER_ID")
	role := envOrDefault("TF_TEST_USER_ROLE", "Administrator")
	sweepUser(t, userID)
	cleanupUser(t, userID)
	resource.Test(t, resource.TestCase{
		CheckDestroy: testAccUserDestroyed(userID),
		Steps:        upgradeSteps(correctCreds + userConfig(userID, "acc-upgrade-user", role)),
	})
}

// TestAccUpgradeVlan upgrades a crucible_vlan acquired from the default partition.
func TestAccUpgradeVlan(t *testing.T) {
	upgradeProviderEnabled(t)

	partitionID := os.Getenv("TF_TEST_PARTITION_ID") // "" => system default partition
	vlanID := envOrDefault("TF_TEST_VLAN_ID", "1000")
	num, _ := strconv.Atoi(vlanID)
	sweepVlanByNumber(t, num, partitionID)
	resource.Test(t, resource.TestCase{
		Steps: upgradeSteps(correctCreds + vlanConfig(partitionID, vlanID)),
	})
}

// TestAccUpgradeViewNetwork upgrades a crucible_player_view_network in two teams,
// also covering the inline multi-team parent view.
func TestAccUpgradeViewNetwork(t *testing.T) {
	upgradeProviderEnabled(t)

	providerType := envOrDefault("TF_TEST_NETWORK_PROVIDER_TYPE", "Unknown")
	instanceID := envOrDefault("TF_TEST_NETWORK_PROVIDER_INSTANCE_ID", "acc-test-instance")
	networkID := envOrDefault("TF_TEST_NETWORK_ID", "acc-test-network")
	sweepViewByName(t, "acc-test-net-multi-view")
	cleanupViewByName(t, "acc-test-net-multi-view")
	resource.Test(t, resource.TestCase{
		CheckDestroy: testAccViewNetworkDestroyed("crucible_player_view_network.multi"),
		Steps:        upgradeSteps(correctCreds + viewNetworkMultiTeamConfig(providerType, instanceID, networkID)),
	})
}
