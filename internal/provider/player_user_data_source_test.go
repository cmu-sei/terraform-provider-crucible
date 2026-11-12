// Copyright 2026 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccPlayerUserDataSourceByName creates a user and looks it up by name,
// exactly and without regard to case.
func TestAccPlayerUserDataSourceByName(t *testing.T) {
	userID := testEnv("TF_TEST_LOOKUP_USER_ID")
	const name = "acc-test-user-lookup"
	sweepUser(t, userID)
	registerUserCleanup(t, userID)

	config := fmt.Sprintf(`
resource "crucible_player_user" "lookup" {
  user_id = "%s"
  name    = "%s"
  role    = "Content Developer"
}

data "crucible_player_user" "exact" {
  name       = crucible_player_user.lookup.name
  depends_on = [crucible_player_user.lookup]
}

data "crucible_player_user" "case_insensitive" {
  name             = upper(crucible_player_user.lookup.name)
  case_insensitive = true
  depends_on       = [crucible_player_user.lookup]
}
`, userID, name)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccUserDestroyed(userID),
		Steps: []resource.TestStep{{
			Config: tfConfig(config),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.crucible_player_user.exact", "id", userID),
				resource.TestCheckResourceAttr("data.crucible_player_user.exact", "name", name),
				resource.TestCheckResourceAttr("data.crucible_player_user.exact", "role_id", "7fd6aa3e-a765-47b8-a77e-f58eae53a82f"),
				resource.TestCheckResourceAttr("data.crucible_player_user.exact", "role_name", "Content Developer"),
				resource.TestCheckResourceAttr("data.crucible_player_user.exact", "attributes.%", "0"),
				resource.TestCheckResourceAttr("data.crucible_player_user.case_insensitive", "id", userID),
				resource.TestCheckResourceAttr("data.crucible_player_user.case_insensitive", "name", name),
			),
		}},
	})
}

// TestAccPlayerUserDataSourceByIdentityAttribute looks a user up by an identity
// attribute. Player stores identity attributes from the user's own token, so
// the test cannot create one; it skips unless TF_TEST_USER_ATTRIBUTE_KEY,
// TF_TEST_USER_ATTRIBUTE_VALUE and TF_TEST_USER_ATTRIBUTE_USER_ID name a user
// who has used a Player configured with that attribute.
func TestAccPlayerUserDataSourceByIdentityAttribute(t *testing.T) {
	key := os.Getenv("TF_TEST_USER_ATTRIBUTE_KEY")
	value := os.Getenv("TF_TEST_USER_ATTRIBUTE_VALUE")
	userID := os.Getenv("TF_TEST_USER_ATTRIBUTE_USER_ID")
	if key == "" || value == "" || userID == "" {
		t.Skip("TF_TEST_USER_ATTRIBUTE_KEY, TF_TEST_USER_ATTRIBUTE_VALUE and TF_TEST_USER_ATTRIBUTE_USER_ID not set; skipping identity attribute lookup test")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: tfConfig(fmt.Sprintf(`
data "crucible_player_user" "by_attribute" {
  identity_attributes = {
    %q = %q
  }
}
`, key, value)),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.crucible_player_user.by_attribute", "id", userID),
				resource.TestCheckResourceAttr("data.crucible_player_user.by_attribute", "attributes."+key, value),
			),
		}},
	})
}

func TestAccPlayerUserDataSourceNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: tfConfig(`
data "crucible_player_user" "missing" {
  name = "terraform-user-does-not-exist"
}
`),
			ExpectError: regexp.MustCompile(`resolved to 0 users`),
		}},
	})
}

func TestAccPlayerUserDataSourceUnknownAttributeKey(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: tfConfig(`
data "crucible_player_user" "missing" {
  identity_attributes = {
    terraform-attribute-does-not-exist = "value"
  }
}
`),
			ExpectError: regexp.MustCompile(`no identity attribute with key`),
		}},
	})
}

func TestAccPlayerUserDataSourceRequiresFilter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: tfConfig(`
data "crucible_player_user" "none" {}
`),
			ExpectError: regexp.MustCompile(`At least one of these attributes must be configured`),
		}},
	})
}
