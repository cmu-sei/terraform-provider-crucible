package provider_test

import (
	"crucible_provider/internal/api"
	"crucible_provider/internal/provider"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

// Test case for the creation and updating of an application template resource
//
// Execution steps
// 1. Terraform calls apply
// 2. Verify local and remote states
// 3. Terraform calls apply again to update resource
// 4. Verify state
// 5. Terraform destroys resource
//
// Expected behavior:
// Resource is created, updated, and destroyed without error
func TestAccAppTemplate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: map[string]terraform.ResourceProvider{
			"crucible": provider.Provider(),
		},
		Steps: []resource.TestStep{
			{
				Config: configAppTemplate,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_player_application_template.test", "name", "TestTemplate"),
					resource.TestCheckResourceAttr("crucible_player_application_template.test", "url", "http://example.com"),
					resource.TestCheckResourceAttr("crucible_player_application_template.test", "icon", "https://upload.wikimedia.org/wikipedia/en/thumb/9/9e/Buffalo_Sabres_Logo.svg/1200px-Buffalo_Sabres_Logo.svg.png"),
					resource.TestCheckResourceAttr("crucible_player_application_template.test", "embeddable", "false"),
					resource.TestCheckResourceAttr("crucible_player_application_template.test", "load_in_background", "false"),
					verifyRemoteTemplate("TestTemplate", "http://example.com", "https://upload.wikimedia.org/wikipedia/en/thumb/9/9e/Buffalo_Sabres_Logo.svg/1200px-Buffalo_Sabres_Logo.svg.png",
						"false", "false"),
				),
			},
			{
				Config: configAppTemplateUpdated,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("crucible_player_application_template.test", "name", "TestTemplateUpdated"),
					resource.TestCheckResourceAttr("crucible_player_application_template.test", "url", "http://example.com"),
					resource.TestCheckResourceAttr("crucible_player_application_template.test", "icon", "https://upload.wikimedia.org/wikipedia/en/thumb/9/9e/Buffalo_Sabres_Logo.svg/1200px-Buffalo_Sabres_Logo.svg.png"),
					resource.TestCheckResourceAttr("crucible_player_application_template.test", "embeddable", "false"),
					resource.TestCheckResourceAttr("crucible_player_application_template.test", "load_in_background", "false"),
					verifyRemoteTemplate("TestTemplateUpdated", "http://example.com", "https://upload.wikimedia.org/wikipedia/en/thumb/9/9e/Buffalo_Sabres_Logo.svg/1200px-Buffalo_Sabres_Logo.svg.png",
						"false", "false"),
				),
			},
		},
	})
}

func verifyRemoteTemplate(name, url, icon, embeddable, load string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		mod := s.Modules[0]
		str := fmt.Sprintf("%+v", mod)
		str = strings.TrimSpace(str)

		lines := strings.Split(str, "\n")
		lines = lines[1:]
		var id string

		for _, line := range lines {
			line = strings.TrimSpace(line)

			split := strings.Split(line, " = ")

			if split[0] == "ID" {
				id = split[1]
			}
		}

		remote, err := api.AppTemplateRead(id, getMap())
		if err != nil {
			return err
		}

		if remote.Name != name {
			return fmt.Errorf("for app template, remote name %s does not equal expected name %s", remote.Name, name)
		}

		if remote.URL != url {
			return fmt.Errorf("for app template, remote url %s does not equal expected url %s", remote.URL, url)
		}

		if remote.Icon != icon {
			return fmt.Errorf("for app template, remote icon %s does not equal expected icon %s", remote.Icon, icon)
		}

		if strconv.FormatBool(remote.Embeddable) != embeddable {
			return fmt.Errorf("for app template, remote value for embeddable %v does not equal expected value for embeddable %v", remote.Embeddable, embeddable)
		}

		if strconv.FormatBool(remote.LoadInBackground) != load {
			return fmt.Errorf("for app template, remote value for load_in_background %v does not equal expected value for load_in_background %v", remote.LoadInBackground, load)
		}
		return nil
	}
}
