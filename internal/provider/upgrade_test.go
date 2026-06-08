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
	sweepViewByName(t, "test")
	registerViewCleanupByName(t, "test")

	resource.Test(t, resource.TestCase{
		CheckDestroy: testAccViewDestroyed,
		Steps: upgradeStepsWithUpdate(
			correctCreds+configViewEmpty,
			correctCreds+configViewEmptyUpdated,
			testAccVerifyRemoteView(emptyViewExpected),
			testAccVerifyRemoteView(emptyViewExpectedUpdated),
		),
	})
}

// TestAccUpgradeAppTemplate verifies the same empty-plan upgrade guarantee for
// crucible_player_application_template.
func TestAccUpgradeAppTemplate(t *testing.T) {
	upgradeProviderEnabled(t)

	resource.Test(t, resource.TestCase{
		Steps: upgradeStepsWithUpdate(
			correctCreds+configAppTemplate,
			correctCreds+configAppTemplateUpdated,
			verifyRemoteTemplate("TestTemplate", "http://example.com",
				"https://upload.wikimedia.org/wikipedia/en/thumb/9/9e/Buffalo_Sabres_Logo.svg/1200px-Buffalo_Sabres_Logo.svg.png",
				"false", "false"),
			verifyRemoteTemplate("TestTemplateUpdated", "http://example.com",
				"https://upload.wikimedia.org/wikipedia/en/thumb/9/9e/Buffalo_Sabres_Logo.svg/1200px-Buffalo_Sabres_Logo.svg.png",
				"false", "false"),
		),
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

// upgradeStepsWithUpdate extends upgradeSteps with two things the empty-plan-only
// flow can't prove: that the migrated resource's content is actually correct after
// the switch, and that the new (Framework) provider can MUTATE a resource that was
// created by the old SDKv1 provider. The latter is what a user does immediately
// after upgrading, so it is the real upgrade-safety guarantee.
//
//   - Step 1 creates with the old published provider.
//   - Step 2 switches to the in-process Framework build, asserts an empty plan, and
//     runs postSwitchCheck (content assertions on the migrated state).
//   - Step 3 applies updatedConfig under the Framework provider and runs
//     postUpdateCheck, exercising Update against old-provider-born state.
//
// createConfig must parse byte-identically under both provider versions; updatedConfig
// only needs to parse under the new provider.
func upgradeStepsWithUpdate(createConfig, updatedConfig string, postSwitchCheck, postUpdateCheck resource.TestCheckFunc) []resource.TestStep {
	return []resource.TestStep{
		{
			ExternalProviders: externalOldProvider(),
			Config:            createConfig,
		},
		{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Config:                   createConfig,
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectEmptyPlan(),
				},
			},
			Check: postSwitchCheck,
		},
		{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Config:                   updatedConfig,
			Check:                    postUpdateCheck,
		},
	}
}

// TestAccUpgradeViewWithTeams enriches the view upgrade coverage beyond the bare
// configViewEmpty: it exercises team role/permissions and the nested
// app_instance/user blocks across the SDKv1->Framework migration.
func TestAccUpgradeViewWithTeams(t *testing.T) {
	upgradeProviderEnabled(t)
	sweepViewByName(t, "test")
	registerViewCleanupByName(t, "test")

	resource.Test(t, resource.TestCase{
		CheckDestroy: testAccViewDestroyed,
		Steps: upgradeStepsWithUpdate(
			correctCreds+configViewTeams,
			correctCreds+configViewTeamsUpdated,
			testAccVerifyRemoteView(teamViewExpected),
			testAccVerifyRemoteView(teamViewExpectedUpdated),
		),
	})
}

// TestAccUpgradeVMConsole upgrades a VM with a console_connection_info block,
// exercising the url-token read path across provider versions.
func TestAccUpgradeVMConsole(t *testing.T) {
	upgradeProviderEnabled(t)

	const vmID = "c2e4d3a5-2222-4bbb-9ccc-000000000001"
	const baseURL = "https://example.com/console"
	const updatedURL = "https://example.com/console-updated"
	registerVMCleanup(t, vmID)
	sweepVM(t, vmID)
	// vmConsoleConfig emits an inline view fixture (for a real team id); clean it up.
	sweepViewByName(t, vmConsoleViewName)
	registerViewCleanupByName(t, vmConsoleViewName)
	resource.Test(t, resource.TestCase{
		CheckDestroy: testAccVMDestroyed(vmID),
		// The console VM's url is asserted against state (not the remote API): the
		// VM API appends a Guacamole "/#/client/<token>" fragment to the url, which
		// the provider strips on read, so state holds the base url but the remote
		// value differs. testAccVMRemoteMatchesState would mismatch here.
		Steps: upgradeStepsWithUpdate(
			correctCreds+vmConsoleConfig(vmID, baseURL),
			correctCreds+vmConsoleConfig(vmID, updatedURL),
			resource.TestCheckResourceAttr("crucible_player_virtual_machine.regconsole", "url", baseURL),
			resource.TestCheckResourceAttr("crucible_player_virtual_machine.regconsole", "url", updatedURL),
		),
	})
}

// TestAccUpgradeVMProxmox upgrades a VM with a proxmox_vm_info block.
func TestAccUpgradeVMProxmox(t *testing.T) {
	upgradeProviderEnabled(t)

	const vmID = "c2e4d3a5-2222-4bbb-9ccc-000000000002"
	// Test-specific proxmox id: it is a unique server-side PK, so avoid colliding
	// with other fixtures / test/main.tf.
	const proxmoxID = "990010"
	const baseURL = "https://example.com/proxmox"
	const updatedURL = "https://example.com/proxmox-updated"
	sweepVM(t, vmID)
	registerVMCleanup(t, vmID)
	sweepViewByName(t, vmProxmoxViewName)
	registerViewCleanupByName(t, vmProxmoxViewName)
	resource.Test(t, resource.TestCase{
		CheckDestroy: testAccVMDestroyed(vmID),
		Steps: upgradeStepsWithUpdate(
			correctCreds+vmProxmoxConfig(vmID, baseURL, proxmoxID),
			correctCreds+vmProxmoxConfig(vmID, updatedURL, proxmoxID),
			testAccVMRemoteMatchesState("crucible_player_virtual_machine.regproxmox", baseURL, "tf-acc-proxmox", vmUserID()),
			testAccVMRemoteMatchesState("crucible_player_virtual_machine.regproxmox", updatedURL, "tf-acc-proxmox", vmUserID()),
		),
	})
}

// TestAccUpgradeVMMultiTeam upgrades a VM in two teams, exercising team_ids
// ordering across provider versions.
func TestAccUpgradeVMMultiTeam(t *testing.T) {
	upgradeProviderEnabled(t)

	const vmID = "c2e4d3a5-2222-4bbb-9ccc-000000000003"
	const baseURL = "https://example.com/multiteam"
	const updatedURL = "https://example.com/multiteam-updated"
	sweepVM(t, vmID)
	registerVMCleanup(t, vmID)
	sweepViewByName(t, vmMultiTeamViewName)
	registerViewCleanupByName(t, vmMultiTeamViewName)
	resource.Test(t, resource.TestCase{
		CheckDestroy: testAccVMDestroyed(vmID),
		Steps: upgradeStepsWithUpdate(
			correctCreds+vmMultiTeamConfig(vmID, baseURL),
			correctCreds+vmMultiTeamConfig(vmID, updatedURL),
			resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("crucible_player_virtual_machine.regmulti", "team_ids.#", "2"),
				testAccVMRemoteMatchesState("crucible_player_virtual_machine.regmulti", baseURL, "tf-acc-multiteam", vmUserID()),
			),
			resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("crucible_player_virtual_machine.regmulti", "team_ids.#", "2"),
				testAccVMRemoteMatchesState("crucible_player_virtual_machine.regmulti", updatedURL, "tf-acc-multiteam", vmUserID()),
			),
		),
	})
}

// TestAccUpgradeUser upgrades a crucible_player_user with a role set.
func TestAccUpgradeUser(t *testing.T) {
	upgradeProviderEnabled(t)

	userID := testEnv("TF_TEST_USER_ID")
	role := envOrDefault("TF_TEST_USER_ROLE", "Administrator")
	roleUpdated := envOrDefault("TF_TEST_USER_ROLE_UPDATED", "Content Developer")
	sweepUser(t, userID)
	registerUserCleanup(t, userID)
	resource.Test(t, resource.TestCase{
		CheckDestroy: testAccUserDestroyed(userID),
		Steps: upgradeStepsWithUpdate(
			correctCreds+userConfig(userID, "acc-upgrade-user", role),
			correctCreds+userConfig(userID, "acc-upgrade-user", roleUpdated),
			testAccUserRemoteRole(userID, role),
			testAccUserRemoteRole(userID, roleUpdated),
		),
	})
}

// TestAccUpgradeVlan upgrades a crucible_vlan acquired from the default partition.
func TestAccUpgradeVlan(t *testing.T) {
	upgradeProviderEnabled(t)

	partitionID := os.Getenv("TF_TEST_PARTITION_ID") // "" => system default partition
	vlanID := envOrDefault("TF_TEST_VLAN_ID", "1000")
	num, _ := strconv.Atoi(vlanID)
	sweepVlanByNumber(t, num, partitionID)
	// Stays on the empty-plan-only upgradeSteps: every crucible_vlan attribute is
	// RequiresReplace, so there is no in-place Update to exercise across the
	// boundary — a config change is a destroy+recreate. The no-drift guarantee is
	// the meaningful one for this resource.
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
	registerViewCleanupByName(t, "acc-test-net-multi-view")
	resource.Test(t, resource.TestCase{
		CheckDestroy: testAccViewNetworkDestroyed("crucible_player_view_network.multi"),
		Steps: upgradeStepsWithUpdate(
			correctCreds+viewNetworkMultiTeamConfig(providerType, instanceID, networkID, "acc-test-net-multi"),
			correctCreds+viewNetworkMultiTeamConfig(providerType, instanceID, networkID, "acc-test-net-multi-updated"),
			testAccViewNetworkRemoteMatches("crucible_player_view_network.multi",
				providerType, instanceID, networkID, "acc-test-net-multi"),
			testAccViewNetworkRemoteMatches("crucible_player_view_network.multi",
				providerType, instanceID, networkID, "acc-test-net-multi-updated"),
		),
	})
}
