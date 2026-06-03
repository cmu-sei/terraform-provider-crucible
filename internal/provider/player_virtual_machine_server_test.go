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
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Test case for a normal creation/deployment of a VM. VM fields are set
// properly, as are API credentials.
//
// Expected behavior: resource is created, verified, and destroyed without error.
func TestAccVMBasicSuccessful(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccVMDestroyed("6a7ec409-d275-4b31-94d3-a51cb61d2519"),
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
				Config:      correctCreds + configVMIncorrectUserID,
				ExpectError: regexp.MustCompile("status code 400"),
			},
		},
	})
}

// Test case for a VM that is created and then updated (name change).
func TestAccVMUpdate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccVMDestroyed("6a7ec409-d275-4b31-94d3-a51cb61d2519"),
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
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccVMDestroyed("1d0b5b53-e034-492d-95c6-714379a4f51e"),
			testAccVMDestroyed("3faebb23-d896-410b-9fcb-a17d9d37427d"),
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
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccVMDestroyed("6a7ec409-d275-4b31-94d3-a51cb61d2519"),
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
