// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider_test

import (
	"fmt"
	"testing"

	"github.com/cmu-sei/terraform-provider-crucible/internal/api"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccViewNetwork covers create, name update, and destroy for the
// crucible_player_view_network resource. A view is created inline to supply the
// required view_id, so the test only needs the standard Player credentials.
//
// provider_instance_id / network_id reference real infrastructure objects; they
// default to placeholder values and can be overridden with
// TF_TEST_NETWORK_PROVIDER_INSTANCE_ID / TF_TEST_NETWORK_ID. provider_type
// defaults to "Unknown" (override with TF_TEST_NETWORK_PROVIDER_TYPE).
func TestAccViewNetwork(t *testing.T) {
	providerType := envOrDefault("TF_TEST_NETWORK_PROVIDER_TYPE", "Unknown")
	instanceID := envOrDefault("TF_TEST_NETWORK_PROVIDER_INSTANCE_ID", "00000000-0000-0000-0000-000000000001")
	networkID := envOrDefault("TF_TEST_NETWORK_ID", "00000000-0000-0000-0000-000000000002")

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

func viewNetworkConfig(providerType, instanceID, networkID, name string) string {
	return fmt.Sprintf(`resource "crucible_player_view" "net_parent" {
		name              = "acc-test-net-view"
		description       = "view for view_network acceptance test"
		status            = "Active"
		create_admin_team = false
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
