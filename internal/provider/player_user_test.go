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

// TestAccPlayerUser covers create, role update, import, and destroy for the
// crucible_player_user resource.
//
// A crucible_player_user binds an existing identity-provider user (by GUID) into
// Player, so the test requires a real user id supplied via TF_TEST_USER_ID. The
// two roles default to the standard Player roles but can be overridden with
// TF_TEST_USER_ROLE / TF_TEST_USER_ROLE_UPDATED.
func TestAccPlayerUser(t *testing.T) {
	userID := os.Getenv("TF_TEST_USER_ID")
	if userID == "" {
		t.Skip("TF_TEST_USER_ID not set; skipping crucible_player_user acceptance test")
	}
	role := envOrDefault("TF_TEST_USER_ROLE", "Administrator")
	roleUpdated := envOrDefault("TF_TEST_USER_ROLE_UPDATED", "Content Developer")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccUserDestroyed(userID),
		Steps: []resource.TestStep{
			{
				Config: correctCreds + userConfig(userID, "acc-test-user", role),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_player_user.test", "user_id", userID),
					resource.TestCheckResourceAttr("crucible_player_user.test", "name", "acc-test-user"),
					resource.TestCheckResourceAttr("crucible_player_user.test", "role", role),
					testAccUserRemoteRole(userID, role),
				),
			},
			{
				Config: correctCreds + userConfig(userID, "acc-test-user", roleUpdated),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_player_user.test", "role", roleUpdated),
					testAccUserRemoteRole(userID, roleUpdated),
				),
			},
			{
				ResourceName:      "crucible_player_user.test",
				ImportState:       true,
				ImportStateVerify: true,
				// role is resolved from an id on read and name is not returned by
				// the user read endpoint, so only id/user_id round-trip cleanly.
				ImportStateVerifyIgnore: []string{"name", "role"},
			},
		},
	})
}

func userConfig(userID, name, role string) string {
	return fmt.Sprintf(`resource "crucible_player_user" "test" {
		user_id = "%s"
		name    = "%s"
		role    = "%s"
	}
	`, userID, name, role)
}

// testAccUserRemoteRole verifies the user's role resolves to the expected name
// via the Player API.
func testAccUserRemoteRole(userID, role string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		m := getMap()
		user, err := api.ReadUser(userID, m)
		if err != nil {
			return err
		}
		roleID, ok := user.Role.(string)
		if !ok || roleID == "" {
			return fmt.Errorf("expected role %q, but user has no role", role)
		}
		name, err := api.GetRoleByID(roleID, m)
		if err != nil {
			return err
		}
		if name != role {
			return fmt.Errorf("expected role %q, got %q", role, name)
		}
		return nil
	}
}

// testAccUserDestroyed asserts the user no longer exists in Player.
func testAccUserDestroyed(userID string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		exists, err := api.UserExists(userID, getMap())
		if err != nil {
			return fmt.Errorf("error checking destroyed user %s: %w", userID, err)
		}
		if exists {
			return fmt.Errorf("user %s still exists after destroy", userID)
		}
		return nil
	}
}

// envOrDefault returns the environment variable value or a fallback when unset.
func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
