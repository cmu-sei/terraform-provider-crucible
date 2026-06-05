// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider_test

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/cmu-sei/terraform-provider-crucible/internal/api"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Fixed VM ids used by the testConfigs.json-backed VM tests. Shared so each
// test's CheckDestroy, pre-run sweep, and cleanup all reference the same id.
const (
	vmIDNormal = "6a7ec409-d275-4b31-94d3-a51cb61d2519" // configVMNormal / Updated / MultiTeams
	vmIDFirst  = "1d0b5b53-e034-492d-95c6-714379a4f51e" // configVMFirst
	vmIDSecond = "3faebb23-d896-410b-9fcb-a17d9d37427d" // configVMSecond
)

// Test case for a normal creation/deployment of a VM. VM fields are set
// properly, as are API credentials.
//
// Expected behavior: resource is created, verified, and destroyed without error.
func TestAccVMBasicSuccessful(t *testing.T) {
	sweepVM(t, vmIDNormal)
	cleanupVM(t, vmIDNormal)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccVMDestroyed(vmIDNormal),
		Steps: []resource.TestStep{
			{
				Config: correctCreds + configVMNormal,
				Check: resource.ComposeTestCheckFunc(
					testAccVMVerifyLocal("crucible_player_virtual_machine.test", "6a7ec409-d275-4b31-94d3-a51cb61d2519",
						"http://example.com", "foo", "8694c78c-1c49-421b-8ed8-689b46834878",
						[]string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),
					testAccVMRemoteEquals("6a7ec409-d275-4b31-94d3-a51cb61d2519", "http://example.com", "foo",
						"8694c78c-1c49-421b-8ed8-689b46834878", []string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),
				),
			},
			{
				ResourceName:      "crucible_player_virtual_machine.test",
				ImportState:       true,
				ImportStateVerify: true,
				// user_id is echoed by the API and may differ from config form;
				// the remaining attributes round-trip cleanly.
				ImportStateVerifyIgnore: []string{"user_id"},
			},
		},
	})
}

// Test case for a misconfigured VM (bad user id). Creation should fail with a
// 400 and leave no remote state.
func TestAccVMBasicFail(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: correctCreds + configVMIncorrectUserID,
				// The config supplies a malformed user_id ("_"). The typed VM
				// client (internal/vmclient) parses ids into UUIDs before issuing
				// the request, so an invalid value is now rejected client-side
				// ("invalid UUID...") rather than by the API returning 400. Match
				// either so the test passes regardless of which layer rejects it.
				ExpectError: regexp.MustCompile("(?i)invalid UUID|status code 400"),
			},
		},
	})
}

// Test case for a VM that is created and then updated (name change).
func TestAccVMUpdate(t *testing.T) {
	sweepVM(t, vmIDNormal)
	cleanupVM(t, vmIDNormal)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccVMDestroyed(vmIDNormal),
		Steps: []resource.TestStep{
			{
				Config: correctCreds + configVMNormal,
				Check: resource.ComposeTestCheckFunc(
					testAccVMVerifyLocal("crucible_player_virtual_machine.test", "6a7ec409-d275-4b31-94d3-a51cb61d2519",
						"http://example.com", "foo", "8694c78c-1c49-421b-8ed8-689b46834878",
						[]string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),
					testAccVMRemoteEquals("6a7ec409-d275-4b31-94d3-a51cb61d2519", "http://example.com", "foo",
						"8694c78c-1c49-421b-8ed8-689b46834878", []string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),
				),
			},
			{
				Config: correctCreds + configVMNormalUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccVMVerifyLocal("crucible_player_virtual_machine.test", "6a7ec409-d275-4b31-94d3-a51cb61d2519",
						"http://example.com", "bar", "8694c78c-1c49-421b-8ed8-689b46834878",
						[]string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),
					testAccVMRemoteEquals("6a7ec409-d275-4b31-94d3-a51cb61d2519", "http://example.com", "bar",
						"8694c78c-1c49-421b-8ed8-689b46834878", []string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),
				),
			},
		},
	})
}

// Test case for the creation of multiple VMs.
func TestAccVMMultipleCreate(t *testing.T) {
	sweepVM(t, vmIDFirst)
	sweepVM(t, vmIDSecond)
	cleanupVM(t, vmIDFirst)
	cleanupVM(t, vmIDSecond)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccVMDestroyed(vmIDFirst),
			testAccVMDestroyed(vmIDSecond),
		),
		Steps: []resource.TestStep{
			{
				Config: correctCreds + configVMFirst + configVMSecond,
				Check: resource.ComposeTestCheckFunc(
					testAccVMVerifyLocal("crucible_player_virtual_machine.first", "1d0b5b53-e034-492d-95c6-714379a4f51e",
						"http://example.com", "first", "8694c78c-1c49-421b-8ed8-689b46834878",
						[]string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),
					testAccVMRemoteEquals("1d0b5b53-e034-492d-95c6-714379a4f51e", "http://example.com", "first",
						"8694c78c-1c49-421b-8ed8-689b46834878", []string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),
					testAccVMVerifyLocal("crucible_player_virtual_machine.second", "3faebb23-d896-410b-9fcb-a17d9d37427d",
						"http://example.com", "second", "8694c78c-1c49-421b-8ed8-689b46834878",
						[]string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),
					testAccVMRemoteEquals("3faebb23-d896-410b-9fcb-a17d9d37427d", "http://example.com", "second",
						"8694c78c-1c49-421b-8ed8-689b46834878", []string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),
				),
			},
		},
	})
}

// Test case for moving a VM between teams.
func TestAccVMMoveTeams(t *testing.T) {
	sweepVM(t, vmIDNormal)
	cleanupVM(t, vmIDNormal)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccVMDestroyed(vmIDNormal),
		Steps: []resource.TestStep{
			{
				Config: correctCreds + configVMMultiTeams,
				Check: testAccVMRemoteEquals("6a7ec409-d275-4b31-94d3-a51cb61d2519", "http://example.com", "foo",
					"8694c78c-1c49-421b-8ed8-689b46834878",
					[]string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761", "8efdcbd3-daa5-4cb4-b62b-338fe7bf3351"}),
			},
			{
				Config: correctCreds + configVMNormal,
				Check: testAccVMRemoteEquals("6a7ec409-d275-4b31-94d3-a51cb61d2519", "http://example.com", "foo",
					"8694c78c-1c49-421b-8ed8-689b46834878", []string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),
			},
			{
				Config: correctCreds + configVMMultiTeams,
				Check: testAccVMRemoteEquals("6a7ec409-d275-4b31-94d3-a51cb61d2519", "http://example.com", "foo",
					"8694c78c-1c49-421b-8ed8-689b46834878",
					[]string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761", "8efdcbd3-daa5-4cb4-b62b-338fe7bf3351"}),
			},
		},
	})
}

// -------------------- helper functions --------------------

// testAccVMVerifyLocal verifies the Terraform state of a VM resource.
func testAccVMVerifyLocal(res, id, url, name, userID string, teamIDs []string) resource.TestCheckFunc {
	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr(res, "vm_id", id),
		resource.TestCheckResourceAttr(res, "name", name),
		resource.TestCheckResourceAttr(res, "url", url),
		resource.TestCheckResourceAttr(res, "user_id", userID),
		resource.TestCheckResourceAttr(res, "team_ids.#", fmt.Sprintf("%d", len(teamIDs))),
	}
	for _, tid := range teamIDs {
		checks = append(checks, resource.TestCheckTypeSetElemAttr(res, "team_ids.*", tid))
	}
	return resource.ComposeTestCheckFunc(checks...)
}

// testAccVMRemoteEquals verifies the VM with the given ID exists in the API and
// matches the expected fields.
func testAccVMRemoteEquals(id, url, name, userID string, teamIDs []string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		info, err := api.GetVMInfo(id, getMap())
		if err != nil {
			return err
		}
		if info.ID != id {
			return fmt.Errorf("expected id %s, got %s", id, info.ID)
		}
		if info.Name != name {
			return fmt.Errorf("expected name %s, got %s", name, info.Name)
		}
		if info.URL != url {
			return fmt.Errorf("expected url %s, got %s", url, info.URL)
		}
		if uid, ok := info.UserID.(string); !ok || uid != userID {
			return fmt.Errorf("expected user_id %s, got %v", userID, info.UserID)
		}

		got := append([]string{}, info.TeamIDs...)
		want := append([]string{}, teamIDs...)
		sort.Strings(got)
		sort.Strings(want)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			return fmt.Errorf("expected team_ids %v, got %v", want, got)
		}
		return nil
	}
}

// testAccVMDestroyed asserts that the VM with the given ID no longer exists.
func testAccVMDestroyed(id string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		exists, err := api.VMExists(id, getMap())
		if err != nil {
			return fmt.Errorf("error checking destroyed VM %s: %w", id, err)
		}
		if exists {
			return fmt.Errorf("VM %s still exists after destroy", id)
		}
		return nil
	}
}

// vmRegressionTeamID is the team id the existing VM acceptance tests use; the
// per-shape plan-stability tests below reuse it.
const vmRegressionTeamID = "c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"

// The plan-stability tests below build their HCL inline (rather than via the
// configs/testConfigs.json + getVMResource path) because that helper does not
// render the console_connection_info / proxmox_vm_info nested blocks, and these
// tests need them. They follow the viewOneUserConfig precedent in
// player_view_server_test.go.

// TestAccVMConsoleURLStable is the regression test for the reported bug: a VM
// with console_connection_info and a base url. The VM API appends a Guacamole
// "/#/client/<token>" fragment to the url on read; without stripping it (see
// consoleURLBase) the first apply fails with "Provider produced inconsistent
// result after apply". Step 1 reproduces that; step 2 asserts the second plan is
// empty.
func TestAccVMConsoleURLStable(t *testing.T) {
	const vmID = "b1f3c2a4-1111-4aaa-9bbb-000000000001"
	const baseURL = "https://example.com/console"
	sweepVM(t, vmID)
	cleanupVM(t, vmID)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccVMDestroyed(vmID),
		Steps: []resource.TestStep{
			{
				Config: correctCreds + vmConsoleConfig(vmID, baseURL),
				Check: resource.ComposeTestCheckFunc(
					// url is the configured base, with no token appended.
					resource.TestCheckResourceAttr("crucible_player_virtual_machine.regconsole", "url", baseURL),
				),
			},
			{
				Config: correctCreds + vmConsoleConfig(vmID, baseURL),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// TestAccVMProxmoxStable covers the proxmox_vm_info shape and guards the
// proxmox id/type plan modifiers against re-planning on an unchanged config.
func TestAccVMProxmoxStable(t *testing.T) {
	const vmID = "b1f3c2a4-1111-4aaa-9bbb-000000000002"
	// The proxmox id has a unique primary key server-side, so a VM holding it
	// cannot coexist with any other (test fixtures, test/main.tf, leftover
	// orphans). Use a high, test-specific id that real configs are unlikely to
	// use to avoid 500 "duplicate key" collisions.
	const proxmoxID = "990002"
	sweepVM(t, vmID)
	cleanupVM(t, vmID)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccVMDestroyed(vmID),
		Steps: []resource.TestStep{
			{
				Config: correctCreds + vmProxmoxConfig(vmID, "https://example.com/proxmox", proxmoxID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_player_virtual_machine.regproxmox", "proxmox_vm_info.0.id", proxmoxID),
					resource.TestCheckResourceAttr("crucible_player_virtual_machine.regproxmox", "proxmox_vm_info.0.type", "QEMU"),
				),
			},
			{
				Config: correctCreds + vmProxmoxConfig(vmID, "https://example.com/proxmox", proxmoxID),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// TestAccVMBasicStable covers the simplest shape (a base url, no nested blocks)
// and asserts the second plan is empty.
func TestAccVMBasicStable(t *testing.T) {
	const vmID = "b1f3c2a4-1111-4aaa-9bbb-000000000003"
	const baseURL = "https://example.com/basic"
	sweepVM(t, vmID)
	cleanupVM(t, vmID)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccVMDestroyed(vmID),
		Steps: []resource.TestStep{
			{
				Config: correctCreds + vmBasicConfig(vmID, baseURL),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_player_virtual_machine.regbasic", "url", baseURL),
					resource.TestCheckResourceAttr("crucible_player_virtual_machine.regbasic", "default_url", "false"),
				),
			},
			{
				Config: correctCreds + vmBasicConfig(vmID, baseURL),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// TestAccVMDefaultURLStable covers the empty-url path: with no url the API sets
// default_url=true and computes the url itself (no token appended). This guards
// the empty-url UseStateForUnknown behavior the old SDKv1 DiffSuppressFunc
// covered. This is the shape test/main.tf's console VM uses (url commented out).
func TestAccVMDefaultURLStable(t *testing.T) {
	const vmID = "b1f3c2a4-1111-4aaa-9bbb-000000000004"
	sweepVM(t, vmID)
	cleanupVM(t, vmID)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccVMDestroyed(vmID),
		Steps: []resource.TestStep{
			{
				Config: correctCreds + vmDefaultURLConfig(vmID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_player_virtual_machine.regdefault", "default_url", "true"),
				),
			},
			{
				Config: correctCreds + vmDefaultURLConfig(vmID),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// secondVMRegressionTeamID is a second real team id (the one TestAccVMMoveTeams
// uses) for multi-team coverage. It sorts BEFORE vmRegressionTeamID
// ("8efd..." < "c0a1..."), so listing them in the opposite order in config makes
// the test fail if read() ever reorders team_ids (e.g. by sorting) relative to
// config — the bug behind the reported "inconsistent result ... team_ids[n]".
const secondVMRegressionTeamID = "8efdcbd3-daa5-4cb4-b62b-338fe7bf3351"

// TestAccVMMultiTeamStable guards multi-team ordering: a VM in two teams whose
// configured order is the reverse of their sorted order. read() must carry the
// configured order into state, not a sorted one, or apply fails with
// "inconsistent result after apply" on team_ids. A single-team test cannot catch
// this (sorting a one-element list is a no-op).
func TestAccVMMultiTeamStable(t *testing.T) {
	const vmID = "b1f3c2a4-1111-4aaa-9bbb-000000000005"
	const baseURL = "https://example.com/multiteam"
	sweepVM(t, vmID)
	cleanupVM(t, vmID)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccVMDestroyed(vmID),
		Steps: []resource.TestStep{
			{
				Config: correctCreds + vmMultiTeamConfig(vmID, baseURL),
				Check: resource.ComposeTestCheckFunc(
					// Config order is [vmRegressionTeamID, secondVMRegressionTeamID]
					// (the reverse of sorted order); state must preserve it.
					resource.TestCheckResourceAttr("crucible_player_virtual_machine.regmulti", "team_ids.0", vmRegressionTeamID),
					resource.TestCheckResourceAttr("crucible_player_virtual_machine.regmulti", "team_ids.1", secondVMRegressionTeamID),
				),
			},
			{
				Config: correctCreds + vmMultiTeamConfig(vmID, baseURL),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// --- inline config builders for the per-shape plan-stability tests ---

func vmMultiTeamConfig(vmID, url string) string {
	return fmt.Sprintf(`resource "crucible_player_virtual_machine" "regmulti" {
	vm_id    = "%s"
	url      = "%s"
	name     = "tf-acc-multiteam"
	team_ids = ["%s", "%s"]
}
`, vmID, url, vmRegressionTeamID, secondVMRegressionTeamID)
}

func vmConsoleConfig(vmID, url string) string {
	return fmt.Sprintf(`resource "crucible_player_virtual_machine" "regconsole" {
	vm_id    = "%s"
	url      = "%s"
	name     = "tf-acc-console"
	team_ids = ["%s"]

	console_connection_info {
		hostname = "10.0.0.10"
		port     = "3389"
		protocol = "rdp"
		username = "administrator"
		password = "changeme"
	}
}
`, vmID, url, vmRegressionTeamID)
}

func vmProxmoxConfig(vmID, url, proxmoxID string) string {
	return fmt.Sprintf(`resource "crucible_player_virtual_machine" "regproxmox" {
	vm_id    = "%s"
	url      = "%s"
	name     = "tf-acc-proxmox"
	team_ids = ["%s"]

	proxmox_vm_info {
		id   = "%s"
		node = "pve1"
		type = "QEMU"
	}
}
`, vmID, url, vmRegressionTeamID, proxmoxID)
}

func vmBasicConfig(vmID, url string) string {
	return fmt.Sprintf(`resource "crucible_player_virtual_machine" "regbasic" {
	vm_id    = "%s"
	url      = "%s"
	name     = "tf-acc-basic"
	team_ids = ["%s"]
}
`, vmID, url, vmRegressionTeamID)
}

func vmDefaultURLConfig(vmID string) string {
	return fmt.Sprintf(`resource "crucible_player_virtual_machine" "regdefault" {
	vm_id    = "%s"
	name     = "tf-acc-default-url"
	team_ids = ["%s"]

	console_connection_info {
		hostname = "10.0.0.20"
		port     = "3389"
		protocol = "rdp"
		username = "administrator"
		password = "changeme"
	}
}
`, vmID, vmRegressionTeamID)
}
