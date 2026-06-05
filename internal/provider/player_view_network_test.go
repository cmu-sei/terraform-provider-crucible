// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider_test

import (
	"fmt"
	"testing"

	"github.com/cmu-sei/terraform-provider-crucible/internal/api"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccViewNetwork covers create, name update, and destroy for the
// crucible_player_view_network resource. A view is created inline to supply the
// required view_id, so the test only needs the standard Player credentials.
//
// provider_instance_id / network_id are free-form strings identifying the
// backing provider instance and network. With provider_type "Unknown" the dev
// stack accepts arbitrary placeholder strings, so the test runs by default;
// override with TF_TEST_NETWORK_PROVIDER_INSTANCE_ID / TF_TEST_NETWORK_ID /
// TF_TEST_NETWORK_PROVIDER_TYPE for a real provider instance.
func TestAccViewNetwork(t *testing.T) {
	providerType := envOrDefault("TF_TEST_NETWORK_PROVIDER_TYPE", "Unknown")
	instanceID := envOrDefault("TF_TEST_NETWORK_PROVIDER_INSTANCE_ID", "acc-test-instance")
	networkID := envOrDefault("TF_TEST_NETWORK_ID", "acc-test-network")
	// The view network lives under an inline parent view; sweeping/cleaning that
	// view by name cascades the network.
	sweepViewByName(t, "acc-test-net-view")
	registerViewCleanupByName(t, "acc-test-net-view")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccViewNetworkDestroyed("crucible_player_view_network.test"),
		Steps: []resource.TestStep{
			{
				Config: correctCreds + viewNetworkConfig(providerType, instanceID, networkID, "acc-test-net"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_player_view_network.test", "provider_type", providerType),
					resource.TestCheckResourceAttr("crucible_player_view_network.test", "name", "acc-test-net"),
					resource.TestCheckResourceAttrSet("crucible_player_view_network.test", "view_id"),
					testAccViewNetworkExists("crucible_player_view_network.test"),
				),
			},
			{
				Config: correctCreds + viewNetworkConfig(providerType, instanceID, networkID, "acc-test-net-updated"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_player_view_network.test", "name", "acc-test-net-updated"),
					testAccViewNetworkExists("crucible_player_view_network.test"),
				),
			},
			// NOTE: no ImportState step — ImportState passes through only `id`,
			// but Read needs view_id as well, so import is unsupported for this
			// resource as currently implemented.
		},
	})
}

// TestAccViewNetworkMultiTeamStable is the regression test for the view_network
// team_ids ordering bug (same class as the VM team_ids bug): the API returns
// team ids in arbitrary order, so read() must preserve the configured order or
// apply fails with "inconsistent result after apply". The config creates a view
// with two teams and assigns the network to both in a fixed order; step 2
// asserts the second plan is empty.
func TestAccViewNetworkMultiTeamStable(t *testing.T) {
	providerType := envOrDefault("TF_TEST_NETWORK_PROVIDER_TYPE", "Unknown")
	instanceID := envOrDefault("TF_TEST_NETWORK_PROVIDER_INSTANCE_ID", "acc-test-instance")
	networkID := envOrDefault("TF_TEST_NETWORK_ID", "acc-test-network")
	sweepViewByName(t, "acc-test-net-multi-view")
	registerViewCleanupByName(t, "acc-test-net-multi-view")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccViewNetworkDestroyed("crucible_player_view_network.multi"),
		Steps: []resource.TestStep{
			{
				Config: correctCreds + viewNetworkMultiTeamConfig(providerType, instanceID, networkID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_player_view_network.multi", "team_ids.#", "2"),
					testAccViewNetworkExists("crucible_player_view_network.multi"),
				),
			},
			{
				Config: correctCreds + viewNetworkMultiTeamConfig(providerType, instanceID, networkID),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func viewNetworkConfig(providerType, instanceID, networkID, name string) string {
	// create_admin_team must be true: the view network is created with the
	// provider's own credentials, and the VM API requires the caller to be a
	// view admin (otherwise it returns 403 Insufficient Permissions).
	return fmt.Sprintf(`resource "crucible_player_view" "net_parent" {
		name              = "acc-test-net-view"
		description       = "view for view_network acceptance test"
		status            = "Active"
		create_admin_team = true
	}

	resource "crucible_player_view_network" "test" {
		view_id              = crucible_player_view.net_parent.id
		provider_type        = "%s"
		provider_instance_id = "%s"
		network_id           = "%s"
		name                 = "%s"
	}
	`, providerType, instanceID, networkID, name)
}

func viewNetworkMultiTeamConfig(providerType, instanceID, networkID string) string {
	// Two teams; assign the network to both in reverse-sorted order so the
	// configured order is guaranteed to differ from ascending sort order. The
	// old read() sorted team_ids ascending, which would then mismatch this
	// descending config order and fail with "inconsistent result after apply".
	return fmt.Sprintf(`resource "crucible_player_view" "net_multi_parent" {
		name              = "acc-test-net-multi-view"
		description       = "view for view_network multi-team test"
		status            = "Active"
		create_admin_team = true

		team {
			name = "net-team-a"
		}
		team {
			name = "net-team-b"
		}
	}

	resource "crucible_player_view_network" "multi" {
		view_id              = crucible_player_view.net_multi_parent.id
		provider_type        = "%s"
		provider_instance_id = "%s"
		network_id           = "%s"
		name                 = "acc-test-net-multi"
		team_ids = reverse(sort([
			crucible_player_view.net_multi_parent.team[0].team_id,
			crucible_player_view.net_multi_parent.team[1].team_id,
		]))
	}
	`, providerType, instanceID, networkID)
}

// testAccViewNetworkExists verifies the view network exists in the API.
func testAccViewNetworkExists(res string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		viewID, id, err := viewNetworkIDs(s, res)
		if err != nil {
			return err
		}
		exists, err := api.ViewNetworkExists(viewID, id, getMap())
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("view network %s (view %s) does not exist", id, viewID)
		}
		return nil
	}
}

// testAccViewNetworkDestroyed asserts the view network no longer exists. The
// parent view is destroyed in the same operation; a missing view also means the
// network is gone.
func testAccViewNetworkDestroyed(res string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[res]
		if !ok {
			// Resource already removed from state; nothing to verify.
			return nil
		}
		viewID := rs.Primary.Attributes["view_id"]
		id := rs.Primary.ID

		exists, err := api.ViewNetworkExists(viewID, id, getMap())
		if err != nil {
			// The parent view may already be deleted, which surfaces as an
			// error here; treat that as successfully destroyed.
			return nil
		}
		if exists {
			return fmt.Errorf("view network %s still exists after destroy", id)
		}
		return nil
	}
}

func viewNetworkIDs(s *terraform.State, res string) (string, string, error) {
	rs, ok := s.RootModule().Resources[res]
	if !ok {
		return "", "", fmt.Errorf("resource %s not found in state", res)
	}
	return rs.Primary.Attributes["view_id"], rs.Primary.ID, nil
}
