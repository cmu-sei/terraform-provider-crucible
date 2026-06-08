// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider_test

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/cmu-sei/terraform-provider-crucible/internal/api"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Fixed VM ids used by the VM tests. Shared so each test's CheckDestroy, pre-run
// sweep, and cleanup all reference the same id.
const (
	vmIDNormal    = "6a7ec409-d275-4b31-94d3-a51cb61d2519" // basic / update / move-teams
	vmIDFirst     = "1d0b5b53-e034-492d-95c6-714379a4f51e" // multiple-create #1
	vmIDSecond    = "3faebb23-d896-410b-9fcb-a17d9d37427d" // multiple-create #2
	vmIDBadUserID = "33605140-f28f-4722-b161-8540e97e6bab" // bad-user-id fail case
	vmDummyTeamID = "00000000-0000-4000-8000-000000000000" // a syntactically valid (but unused) team id for the fail case
	vmTestViewRes = "fix"                                  // local resource name of each test's inline view fixture
)

// Each VM test provisions its own crucible_player_view (named below) to supply
// real, freshly-created team ids — the tests no longer depend on any pre-seeded
// team GUIDs. Names are distinct per scenario so a sweep/cleanup of one can't
// disturb another.
const (
	vmNormalViewName     = "tf-acc-vm-normal"
	vmMultipleViewName   = "tf-acc-vm-multiple"
	vmMoveViewName       = "tf-acc-vm-move"
	vmBasicViewName      = "tf-acc-vm-basic"
	vmConsoleViewName    = "tf-acc-vm-console"
	vmProxmoxViewName    = "tf-acc-vm-proxmox"
	vmDefaultURLViewName = "tf-acc-vm-default-url"
	vmMultiTeamViewName  = "tf-acc-vm-multiteam"
)

// vmUserID is the user_id stamped on test VMs. Player's VM API stores user_id
// free-form (no Keycloak FK), so a fixed, overridable value round-trips cleanly
// and keeps ImportStateVerify deterministic. See TF_TEST_VM_USER_ID.
func vmUserID() string { return testEnv("TF_TEST_VM_USER_ID") }

// Test case for a normal creation/deployment of a VM. VM fields are set
// properly, as are API credentials.
//
// Expected behavior: resource is created, verified, and destroyed without error.
func TestAccVMBasicSuccessful(t *testing.T) {
	sweepVM(t, vmIDNormal)
	registerVMCleanup(t, vmIDNormal)
	sweepViewByName(t, vmNormalViewName)
	registerViewCleanupByName(t, vmNormalViewName)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccVMDestroyed(vmIDNormal),
		Steps: []resource.TestStep{
			{
				Config: tfConfig(vmViewVMConfig(vmNormalViewName, 1, "foo", 0)),
				Check: resource.ComposeTestCheckFunc(
					testAccVMVerifyLocal("crucible_player_virtual_machine.test", vmIDNormal,
						"foo", vmUserID(), 1),
					resource.TestCheckResourceAttrPair(
						"crucible_player_virtual_machine.test", "team_ids.0",
						"crucible_player_view.fix", "team.0.team_id"),
					testAccVMRemoteMatchesState("crucible_player_virtual_machine.test",
						"http://example.com", "foo", vmUserID()),
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
				Config: tfConfig(vmBadUserIDConfig()),
				// The config supplies a malformed user_id ("_"). The typed VM
				// client (internal/vmclient) parses ids into UUIDs before issuing
				// the request, so an invalid value is now rejected client-side
				// ("invalid UUID...") rather than by the API returning 400. Match
				// either so the test passes regardless of which layer rejects it.
				// No view fixture is needed: the create fails at the client-side
				// user_id parse before any team API call, so a syntactically-valid
				// dummy team id suffices.
				ExpectError: regexp.MustCompile("(?i)invalid UUID|status code 400"),
			},
		},
	})
}

// Test case for a VM that is created and then updated (name change).
func TestAccVMUpdate(t *testing.T) {
	sweepVM(t, vmIDNormal)
	registerVMCleanup(t, vmIDNormal)
	sweepViewByName(t, vmNormalViewName)
	registerViewCleanupByName(t, vmNormalViewName)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccVMDestroyed(vmIDNormal),
		Steps: []resource.TestStep{
			{
				Config: tfConfig(vmViewVMConfig(vmNormalViewName, 1, "foo", 0)),
				Check: resource.ComposeTestCheckFunc(
					testAccVMVerifyLocal("crucible_player_virtual_machine.test", vmIDNormal,
						"foo", vmUserID(), 1),
					testAccVMRemoteMatchesState("crucible_player_virtual_machine.test",
						"http://example.com", "foo", vmUserID()),
				),
			},
			{
				Config: tfConfig(vmViewVMConfig(vmNormalViewName, 1, "bar", 0)),
				Check: resource.ComposeTestCheckFunc(
					testAccVMVerifyLocal("crucible_player_virtual_machine.test", vmIDNormal,
						"bar", vmUserID(), 1),
					testAccVMRemoteMatchesState("crucible_player_virtual_machine.test",
						"http://example.com", "bar", vmUserID()),
				),
			},
		},
	})
}

// Test case for the creation of multiple VMs.
func TestAccVMMultipleCreate(t *testing.T) {
	sweepVM(t, vmIDFirst)
	sweepVM(t, vmIDSecond)
	registerVMCleanup(t, vmIDFirst)
	registerVMCleanup(t, vmIDSecond)
	sweepViewByName(t, vmMultipleViewName)
	registerViewCleanupByName(t, vmMultipleViewName)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccVMDestroyed(vmIDFirst),
			testAccVMDestroyed(vmIDSecond),
		),
		Steps: []resource.TestStep{
			{
				Config: tfConfig(vmMultipleConfig(vmMultipleViewName)),
				Check: resource.ComposeTestCheckFunc(
					testAccVMVerifyLocal("crucible_player_virtual_machine.first", vmIDFirst,
						"first", vmUserID(), 1),
					testAccVMRemoteMatchesState("crucible_player_virtual_machine.first",
						"http://example.com", "first", vmUserID()),
					testAccVMVerifyLocal("crucible_player_virtual_machine.second", vmIDSecond,
						"second", vmUserID(), 1),
					testAccVMRemoteMatchesState("crucible_player_virtual_machine.second",
						"http://example.com", "second", vmUserID()),
				),
			},
		},
	})
}

// Test case for moving a VM between teams. The inline view holds two teams the
// whole time; only the VM's membership toggles between one and both.
func TestAccVMMoveTeams(t *testing.T) {
	sweepVM(t, vmIDNormal)
	registerVMCleanup(t, vmIDNormal)
	sweepViewByName(t, vmMoveViewName)
	registerViewCleanupByName(t, vmMoveViewName)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccVMDestroyed(vmIDNormal),
		Steps: []resource.TestStep{
			{
				Config: tfConfig(vmViewVMConfig(vmMoveViewName, 2, "foo", 0, 1)),
				Check: resource.ComposeTestCheckFunc(
					testAccVMVerifyLocal("crucible_player_virtual_machine.test", vmIDNormal,
						"foo", vmUserID(), 2),
					testAccVMRemoteMatchesState("crucible_player_virtual_machine.test",
						"http://example.com", "foo", vmUserID()),
				),
			},
			{
				Config: tfConfig(vmViewVMConfig(vmMoveViewName, 2, "foo", 0)),
				Check: resource.ComposeTestCheckFunc(
					testAccVMVerifyLocal("crucible_player_virtual_machine.test", vmIDNormal,
						"foo", vmUserID(), 1),
					testAccVMRemoteMatchesState("crucible_player_virtual_machine.test",
						"http://example.com", "foo", vmUserID()),
				),
			},
			{
				Config: tfConfig(vmViewVMConfig(vmMoveViewName, 2, "foo", 0, 1)),
				Check: resource.ComposeTestCheckFunc(
					testAccVMVerifyLocal("crucible_player_virtual_machine.test", vmIDNormal,
						"foo", vmUserID(), 2),
					testAccVMRemoteMatchesState("crucible_player_virtual_machine.test",
						"http://example.com", "foo", vmUserID()),
				),
			},
		},
	})
}

// -------------------- helper functions --------------------

// vmTeamViewFixture emits a crucible_player_view (local resource name vmTestViewRes)
// named viewName with the requested number of teams. create_admin_team must be
// true: the VM is added to the view's teams using the provider's own credentials,
// and the VM API requires the caller to be a view admin (otherwise 403). Reference
// a created team's id as crucible_player_view.fix.team[i].team_id.
func vmTeamViewFixture(viewName string, teams int) string {
	var b strings.Builder
	fmt.Fprintf(&b, `resource "crucible_player_view" %q {
	name              = %q
	description       = "view fixture for VM acceptance tests"
	status            = "Active"
	create_admin_team = true
`, vmTestViewRes, viewName)
	for i := 0; i < teams; i++ {
		fmt.Fprintf(&b, "\n\tteam {\n\t\tname = \"vm-team-%d\"\n\t}\n", i)
	}
	b.WriteString("}\n\n")
	return b.String()
}

// vmViewVMConfig builds a view fixture (teamCount teams) plus a single VM that is
// a member of the teams at the given indexes (referencing their computed
// team_id). Used by the basic/update/move tests.
func vmViewVMConfig(viewName string, teamCount int, name string, teamIdxs ...int) string {
	// The VM resource name, vm_id, and url are identical across every caller, so
	// they are fixed here rather than passed as always-identical parameters; only
	// the view shape (teamCount/teamIdxs) and the VM name vary.
	const vmRes = "test"
	const vmID = vmIDNormal
	const url = "http://example.com"
	exprs := make([]string, len(teamIdxs))
	for i, idx := range teamIdxs {
		exprs[i] = fmt.Sprintf("crucible_player_view.%s.team[%d].team_id", vmTestViewRes, idx)
	}
	return vmTeamViewFixture(viewName, teamCount) + fmt.Sprintf(`resource "crucible_player_virtual_machine" %q {
	vm_id    = "%s"
	url      = "%s"
	name     = "%s"
	user_id  = "%s"
	team_ids = [%s]
}
`, vmRes, vmID, url, name, vmUserID(), strings.Join(exprs, ", "))
}

// vmMultipleConfig builds one view (single team) plus two VMs both in that team,
// for the multiple-create test.
func vmMultipleConfig(viewName string) string {
	teamExpr := fmt.Sprintf("crucible_player_view.%s.team[0].team_id", vmTestViewRes)
	vm := func(res, id, name string) string {
		return fmt.Sprintf(`resource "crucible_player_virtual_machine" %q {
	vm_id    = "%s"
	url      = "http://example.com"
	name     = "%s"
	user_id  = "%s"
	team_ids = [%s]
}
`, res, id, name, vmUserID(), teamExpr)
	}
	return vmTeamViewFixture(viewName, 1) + vm("first", vmIDFirst, "first") + vm("second", vmIDSecond, "second")
}

// vmBadUserIDConfig builds a VM with a malformed user_id and no view fixture. The
// create fails at the client-side user_id UUID parse before any team API call, so
// a syntactically-valid dummy team id is enough.
func vmBadUserIDConfig() string {
	return fmt.Sprintf(`resource "crucible_player_virtual_machine" "bad" {
	vm_id    = "%s"
	url      = "http://example.com"
	name     = "foo"
	user_id  = "_"
	team_ids = ["%s"]
}
`, vmIDBadUserID, vmDummyTeamID)
}

// testAccVMVerifyLocal verifies the Terraform state of a VM resource (the
// scalar fields against expected constants, and the team_ids count). The team ids
// themselves are computed (the fixture's freshly-created teams), so they are
// asserted by membership count here and by remote/round-trip checks elsewhere.
func testAccVMVerifyLocal(res, id, name, userID string, teamCount int) resource.TestCheckFunc {
	// Every caller uses the same VM url; fixed here rather than passed as an
	// always-identical parameter.
	const url = "http://example.com"
	return resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr(res, "vm_id", id),
		resource.TestCheckResourceAttr(res, "name", name),
		resource.TestCheckResourceAttr(res, "url", url),
		resource.TestCheckResourceAttr(res, "user_id", userID),
		resource.TestCheckResourceAttr(res, "team_ids.#", strconv.Itoa(teamCount)),
	)
}

// testAccVMRemoteMatchesState verifies the VM in the API matches the VM's own
// Terraform state: the scalar fields equal the expected constants, and the team
// membership reported by the API equals exactly the team_ids the provider stored
// in state. Reading team_ids from state (rather than a fixed list) decouples the
// check from the computed fixture ids and naturally covers the move-teams subset
// case.
func testAccVMRemoteMatchesState(res, wantURL, wantName, wantUserID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[res]
		if !ok {
			return fmt.Errorf("resource %s not found in state", res)
		}
		id := rs.Primary.Attributes["vm_id"]
		if id == "" {
			id = rs.Primary.ID
		}
		info, err := api.GetVMInfo(id, getMap())
		if err != nil {
			return err
		}
		if info.ID != id {
			return fmt.Errorf("expected id %s, got %s", id, info.ID)
		}
		if info.Name != wantName {
			return fmt.Errorf("expected name %s, got %s", wantName, info.Name)
		}
		if info.URL != wantURL {
			return fmt.Errorf("expected url %s, got %s", wantURL, info.URL)
		}
		if uid, ok := info.UserID.(string); !ok || uid != wantUserID {
			return fmt.Errorf("expected user_id %s, got %v", wantUserID, info.UserID)
		}

		n, _ := strconv.Atoi(rs.Primary.Attributes["team_ids.#"])
		want := make([]string, 0, n)
		for i := 0; i < n; i++ {
			want = append(want, rs.Primary.Attributes[fmt.Sprintf("team_ids.%d", i)])
		}
		got := append([]string{}, info.TeamIDs...)
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

// The plan-stability tests below build their HCL inline, following the
// viewOneUserConfig precedent in player_view_server_test.go. Each prepends an
// inline crucible_player_view to supply a real team id, like
// viewNetworkMultiTeamConfig does for view networks.

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
	registerVMCleanup(t, vmID)
	sweepViewByName(t, vmConsoleViewName)
	registerViewCleanupByName(t, vmConsoleViewName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccVMDestroyed(vmID),
		Steps: []resource.TestStep{
			{
				Config: tfConfig(vmConsoleConfig(vmID, baseURL)),
				Check: resource.ComposeTestCheckFunc(
					// url is the configured base, with no token appended.
					resource.TestCheckResourceAttr("crucible_player_virtual_machine.regconsole", "url", baseURL),
				),
			},
			{
				Config: tfConfig(vmConsoleConfig(vmID, baseURL)),
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
	registerVMCleanup(t, vmID)
	sweepViewByName(t, vmProxmoxViewName)
	registerViewCleanupByName(t, vmProxmoxViewName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccVMDestroyed(vmID),
		Steps: []resource.TestStep{
			{
				Config: tfConfig(vmProxmoxConfig(vmID, "https://example.com/proxmox", proxmoxID)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_player_virtual_machine.regproxmox", "proxmox_vm_info.0.id", proxmoxID),
					resource.TestCheckResourceAttr("crucible_player_virtual_machine.regproxmox", "proxmox_vm_info.0.type", "QEMU"),
				),
			},
			{
				Config: tfConfig(vmProxmoxConfig(vmID, "https://example.com/proxmox", proxmoxID)),
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
	registerVMCleanup(t, vmID)
	sweepViewByName(t, vmBasicViewName)
	registerViewCleanupByName(t, vmBasicViewName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccVMDestroyed(vmID),
		Steps: []resource.TestStep{
			{
				Config: tfConfig(vmBasicConfig(vmID, baseURL)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_player_virtual_machine.regbasic", "url", baseURL),
					resource.TestCheckResourceAttr("crucible_player_virtual_machine.regbasic", "default_url", "false"),
				),
			},
			{
				Config: tfConfig(vmBasicConfig(vmID, baseURL)),
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
	registerVMCleanup(t, vmID)
	sweepViewByName(t, vmDefaultURLViewName)
	registerViewCleanupByName(t, vmDefaultURLViewName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccVMDestroyed(vmID),
		Steps: []resource.TestStep{
			{
				Config: tfConfig(vmDefaultURLConfig(vmID)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_player_virtual_machine.regdefault", "default_url", "true"),
				),
			},
			{
				Config: tfConfig(vmDefaultURLConfig(vmID)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// TestAccVMMultiTeamStable guards multi-team ordering: a VM in two teams whose
// configured order is the reverse of their sorted order. read() must carry the
// configured order into state, not a sorted one, or apply fails with
// "inconsistent result after apply" on team_ids. A single-team test cannot catch
// this (sorting a one-element list is a no-op).
//
// The two team ids are computed (freshly created in the inline view), so we can't
// hardcode their relative order. Instead the config assigns them with
// reverse(sort([...])) — guaranteeing the VM's configured team_ids are in
// DESCENDING order regardless of the random ids (the same trick
// viewNetworkMultiTeamConfig uses). The check then asserts state preserves that
// descending order: a stray ascending sort in read() would flip team_ids.0 and
// team_ids.1 and fail. Step 2's empty plan confirms the order is stable.
func TestAccVMMultiTeamStable(t *testing.T) {
	const vmID = "b1f3c2a4-1111-4aaa-9bbb-000000000005"
	const baseURL = "https://example.com/multiteam"
	sweepVM(t, vmID)
	registerVMCleanup(t, vmID)
	sweepViewByName(t, vmMultiTeamViewName)
	registerViewCleanupByName(t, vmMultiTeamViewName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccVMDestroyed(vmID),
		Steps: []resource.TestStep{
			{
				Config: tfConfig(vmMultiTeamConfig(vmID, baseURL)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_player_virtual_machine.regmulti", "team_ids.#", "2"),
					// Config order is reverse(sort(...)) => descending. State must
					// preserve it; a re-sort in read() would make it ascending.
					testAccVMTeamIDsDescending("crucible_player_virtual_machine.regmulti"),
				),
			},
			{
				Config: tfConfig(vmMultiTeamConfig(vmID, baseURL)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// testAccVMTeamIDsDescending asserts the VM's first two team_ids are in
// descending order, proving read() did not re-sort them ascending.
func testAccVMTeamIDsDescending(res string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[res]
		if !ok {
			return fmt.Errorf("resource %s not found in state", res)
		}
		a := rs.Primary.Attributes["team_ids.0"]
		b := rs.Primary.Attributes["team_ids.1"]
		if a <= b {
			return fmt.Errorf("expected team_ids in descending order, got [%s, %s]", a, b)
		}
		return nil
	}
}

// --- inline config builders for the per-shape plan-stability tests ---
// Each prepends a crucible_player_view (resource name vmTestViewRes) so the VM
// has a real team to join. These builders are also used by upgrade_test.go.

func vmMultiTeamConfig(vmID, url string) string {
	return vmTeamViewFixture(vmMultiTeamViewName, 2) + fmt.Sprintf(`resource "crucible_player_virtual_machine" "regmulti" {
	vm_id    = "%s"
	url      = "%s"
	name     = "tf-acc-multiteam"
	user_id  = "%s"
	team_ids = reverse(sort([
		crucible_player_view.%s.team[0].team_id,
		crucible_player_view.%s.team[1].team_id,
	]))
}
`, vmID, url, vmUserID(), vmTestViewRes, vmTestViewRes)
}

func vmConsoleConfig(vmID, url string) string {
	return vmTeamViewFixture(vmConsoleViewName, 1) + fmt.Sprintf(`resource "crucible_player_virtual_machine" "regconsole" {
	vm_id    = "%s"
	url      = "%s"
	name     = "tf-acc-console"
	user_id  = "%s"
	team_ids = [crucible_player_view.%s.team[0].team_id]

	console_connection_info {
		hostname = "10.0.0.10"
		port     = "3389"
		protocol = "rdp"
		username = "administrator"
		password = "changeme"
	}
}
`, vmID, url, vmUserID(), vmTestViewRes)
}

func vmProxmoxConfig(vmID, url, proxmoxID string) string {
	return vmTeamViewFixture(vmProxmoxViewName, 1) + fmt.Sprintf(`resource "crucible_player_virtual_machine" "regproxmox" {
	vm_id    = "%s"
	url      = "%s"
	name     = "tf-acc-proxmox"
	user_id  = "%s"
	team_ids = [crucible_player_view.%s.team[0].team_id]

	proxmox_vm_info {
		id   = "%s"
		node = "pve1"
		type = "QEMU"
	}
}
`, vmID, url, vmUserID(), vmTestViewRes, proxmoxID)
}

func vmBasicConfig(vmID, url string) string {
	return vmTeamViewFixture(vmBasicViewName, 1) + fmt.Sprintf(`resource "crucible_player_virtual_machine" "regbasic" {
	vm_id    = "%s"
	url      = "%s"
	name     = "tf-acc-basic"
	user_id  = "%s"
	team_ids = [crucible_player_view.%s.team[0].team_id]
}
`, vmID, url, vmUserID(), vmTestViewRes)
}

func vmDefaultURLConfig(vmID string) string {
	return vmTeamViewFixture(vmDefaultURLViewName, 1) + fmt.Sprintf(`resource "crucible_player_virtual_machine" "regdefault" {
	vm_id    = "%s"
	name     = "tf-acc-default-url"
	user_id  = "%s"
	team_ids = [crucible_player_view.%s.team[0].team_id]

	console_connection_info {
		hostname = "10.0.0.20"
		port     = "3389"
		protocol = "rdp"
		username = "administrator"
		password = "changeme"
	}
}
`, vmID, vmUserID(), vmTestViewRes)
}
