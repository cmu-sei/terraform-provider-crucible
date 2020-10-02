/*
Crucible
Copyright 2020 Carnegie Mellon University.
NO WARRANTY. THIS CARNEGIE MELLON UNIVERSITY AND SOFTWARE ENGINEERING INSTITUTE MATERIAL IS FURNISHED ON AN "AS-IS" BASIS. CARNEGIE MELLON UNIVERSITY MAKES NO WARRANTIES OF ANY KIND, EITHER EXPRESSED OR IMPLIED, AS TO ANY MATTER INCLUDING, BUT NOT LIMITED TO, WARRANTY OF FITNESS FOR PURPOSE OR MERCHANTABILITY, EXCLUSIVITY, OR RESULTS OBTAINED FROM USE OF THE MATERIAL. CARNEGIE MELLON UNIVERSITY DOES NOT MAKE ANY WARRANTY OF ANY KIND WITH RESPECT TO FREEDOM FROM PATENT, TRADEMARK, OR COPYRIGHT INFRINGEMENT.
Released under a MIT (SEI)-style license, please see license.txt or contact permission@sei.cmu.edu for full terms.
[DISTRIBUTION STATEMENT A] This material has been approved for public release and unlimited distribution.  Please see Copyright notice for non-US Government use and distribution.
Carnegie Mellon(R) and CERT(R) are registered in the U.S. Patent and Trademark Office by Carnegie Mellon University.
DM20-0181
*/

package provider_test

import (
	"crucible_provider/internal/api"
	"crucible_provider/internal/provider"
	"crucible_provider/internal/structs"
	"crucible_provider/internal/util"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

// Test case for creation and updating of an empty view. That is, one without any teams or applications inside of it

// Execution steps:
// 1. Terraform automatically calls apply
// 2. Ensure local and remote states are set
// 3. Terraform calls apply with the second config
// 4. Ensure local and remote states updated properly
// 5. Terraform destroys the resource

// Expected behavior:
// The resource is created, updated, and destroyed without error
func TestAccEmptyView(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: map[string]terraform.ResourceProvider{
			"crucible": provider.Provider(),
		},
		Steps: []resource.TestStep{
			{
				// Create resource and check
				Config: configViewEmpty,
				Check: resource.ComposeTestCheckFunc(
					testAccVerifyLocalView("crucible_player_view.empty", emptyViewExpected),
					testAccVerifyRemoteView(emptyViewExpected),
				),
			},
			{
				// Update resource and check
				Config: configViewEmptyUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccVerifyLocalView("crucible_player_view.empty", emptyViewExpectedUpdated),
					testAccVerifyRemoteView(&structs.ViewInfo{
						Name:        "test",
						Description: "test empty view updated",
						Status:      "Active",
					}),
				),
			},
		},
	})
}

// Test case for creating and updating a view with applications inside of it.
//
// Execution steps: Same as above, there's just applications inside the view now
//
// Expected behavior:
// The resource is created, updated, and destroyed without error
func TestAccViewWithApps(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: map[string]terraform.ResourceProvider{
			"crucible": provider.Provider(),
		},
		Steps: []resource.TestStep{
			{
				Config: configViewApps,
				Check: resource.ComposeTestCheckFunc(
					testAccVerifyLocalView("crucible_player_view.apps", appsViewExpected),
					testAccVerifyRemoteView(appsViewExpected),
				),
			},
			{
				Config: configViewAppsUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccVerifyLocalView("crucible_player_view.apps", appsViewExpectedUpdated),
					testAccVerifyRemoteView(appsViewExpectedUpdated),
				),
			},
		},
	})
}

// Test case for creating a view with teams inside of it. Same as above except now there are also teams.
//
// Execution steps: Same as above
//
// Expected behavior:
// The resource is created, updated, and destroyed without error
func TestAccViewWithTeams(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: map[string]terraform.ResourceProvider{
			"crucible": provider.Provider(),
		},
		Steps: []resource.TestStep{
			{
				Config: configViewTeams,
				Check: resource.ComposeTestCheckFunc(
					testAccVerifyLocalView("crucible_player_view.teams", teamViewExpected),
					testAccVerifyRemoteView(teamViewExpected)),
			},
			{
				Config: configViewTeamsUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccVerifyLocalView("crucible_player_view.teams", teamViewExpectedUpdated),
					testAccVerifyRemoteView(teamViewExpectedUpdated)),
			},
		},
	})
}

// Test case for a view with teams and users inside those teams
//
// Execution steps: Same as before
//
// Expected behavior:
// Resource is created, updated, and destroyed without error
func TestAccViewWithUsers(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: map[string]terraform.ResourceProvider{
			"crucible": provider.Provider(),
		},
		Steps: []resource.TestStep{
			{
				Config: configViewUsers,
				Check: resource.ComposeTestCheckFunc(
					testAccVerifyLocalView("crucible_player_view.users", userViewExpected),
					testAccVerifyRemoteView(userViewExpected),
				),
			},
			{
				Config: configViewUsersUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccVerifyLocalView("crucible_player_view.users", userViewExpectedUpdated),
					testAccVerifyRemoteView(userViewExpectedUpdated),
				),
			},
		},
	})
}

// Test case for a view whose teams have app instances.
//
// Execution steps: Same as before, but also test that instances can be removed and added
//
// Expected behavior:
// View can be created and updated without error
func TestAccViewInstances(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: map[string]terraform.ResourceProvider{
			"crucible": provider.Provider(),
		},
		Steps: []resource.TestStep{
			// View with 2 app instances
			{
				Config: configViewInstances,
				Check: resource.ComposeTestCheckFunc(
					testAccVerifyLocalView("crucible_player_view.instances", instanceViewExpected),
					testAccVerifyRemoteView(instanceViewExpected),
				),
			},
			// Remove the app instances
			{
				Config: configViewUsers,
				Check: resource.ComposeTestCheckFunc(
					testAccVerifyLocalView("crucible_player_view.users", userViewExpected),
					testAccVerifyRemoteView(userViewExpected),
				),
			},
			// Add them back
			{
				Config: configViewInstances,
				Check: resource.ComposeTestCheckFunc(
					testAccVerifyLocalView("crucible_player_view.instances", instanceViewExpected),
					testAccVerifyRemoteView(instanceViewExpected),
				),
			},
			// Update the instances
			{
				Config: configViewInstancesUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccVerifyLocalView("crucible_player_view.instances", instanceViewExpectedUpdated),
					testAccVerifyRemoteView(instanceViewExpectedUpdated),
				),
			},
		},
	})
}

// -------------------- Helper functions --------------------

func testAccVerifyLocalView(viewName string, view *structs.ViewInfo) resource.TestCheckFunc {
	return resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr(viewName, "name", view.Name),
		resource.TestCheckResourceAttr(viewName, "description", view.Description),
		resource.TestCheckResourceAttr(viewName, "status", view.Status),
		// Verify local state of applications
		func(s *terraform.State) error {
			localView := getLocalState(s)
			// fmt.Printf("Local state: %+v\n\n", localView)

			for i, app := range view.Applications {
				// fmt.Printf("Expected App: %+v\n\n", app)

				id := "application." + strconv.Itoa(i) + ".app_id"
				templateId := "application." + strconv.Itoa(i) + ".app_template_id"
				icon := "application." + strconv.Itoa(i) + ".icon"
				load := "application." + strconv.Itoa(i) + ".load_in_background"
				name := "application." + strconv.Itoa(i) + ".name"
				url := "application." + strconv.Itoa(i) + ".url"
				embed := "application." + strconv.Itoa(i) + ".embeddable"

				var appName string
				if app.Name == nil {
					appName = "null"
				} else {
					appName = app.Name.(string)
				}

				var appURL string
				if app.URL == nil {
					appURL = "null"
				} else {
					appURL = app.URL.(string)
				}

				var appIcon string
				if app.Icon == nil {
					appIcon = "null"
				} else {
					appIcon = app.Icon.(string)
				}

				var appEmbed string
				if app.Embeddable == nil {
					appEmbed = "null"
				} else {
					appEmbed = strconv.FormatBool(app.Embeddable.(bool))
				}

				var appLoad string
				if app.LoadInBackground == nil {
					appLoad = "null"
				} else {
					appLoad = strconv.FormatBool(app.LoadInBackground.(bool))
				}

				var appTemplate string
				if app.AppTemplateID == nil {
					appTemplate = "null"
				} else {
					appTemplate = app.AppTemplateID.(string)
				}

				// View ID field is not checked b/c it is impossible to know at compile time, so we cannot provide an expected value
				if localView[id] != app.ID || localView[templateId] != appTemplate ||
					localView[icon] != appIcon || localView[load] != appLoad || localView[name] != appName || localView[url] != appURL ||
					localView[embed] != appEmbed {
					return fmt.Errorf("local state of application %d does not match expected", i)
				}

			}
			return nil
		},
		// Check state of teams
		func(s *terraform.State) error {
			local := getLocalState(s)

			for i, team := range view.Teams {
				name := "team." + strconv.Itoa(i) + ".name"
				role := "team." + strconv.Itoa(i) + ".role_id"
				// Handle permissions
				for j, perm := range team.Permissions {
					locPerm := "team." + strconv.Itoa(i) + ".permissions." + strconv.Itoa(j)
					if local[locPerm] != perm {
						return fmt.Errorf("local permission %s did not match expected permission %s for team %d", local[locPerm], perm, i)
					}
				}

				expectedName := util.Ternary(team.Name == nil, "null", team.Name)
				if local[name] != expectedName {
					return fmt.Errorf("local name %v does not equal expected %v for team %d", local[name], expectedName, i)
				}

				expectedRole := util.Ternary(team.Role == nil, "null", team.Role)
				if local[role] != expectedRole {
					return fmt.Errorf("local role id %v does not equal expected %v for team %d", local[role], expectedRole, i)
				}

				// Handle users
				for j, user := range team.Users {
					locID := "team." + strconv.Itoa(i) + ".user." + strconv.Itoa(j) + ".user_id"
					locRole := "team." + strconv.Itoa(i) + ".user." + strconv.Itoa(j) + ".role_id"

					userRoleExpected := util.Ternary(user.Role == nil, "null", user.Role)

					if local[locID] != user.ID {
						return fmt.Errorf("local user id %v does not equal expected %v for user %d in team %d", local[locID], user.ID, j, i)
					}

					if local[locRole] != userRoleExpected {
						return fmt.Errorf("local role id %v does not equal expected %v for user %d in team %d", local[locRole], user.Role, j, i)
					}
				}

				// Handle app instances
				for j, inst := range team.AppInstances {
					// fmt.Printf("Expected instance: %+v\n\n", inst)

					name := "team." + strconv.Itoa(i) + ".app_instance." + strconv.Itoa(j) + ".name"
					display := "team." + strconv.Itoa(i) + ".app_instance." + strconv.Itoa(j) + ".display_order"

					if local[name] != inst.Name {
						return fmt.Errorf("local app instance app_id %v does not equal expected %v for instance %d in team %d", local[name], inst.Name, j, i)
					}

					if local[display] != strconv.FormatFloat(inst.DisplayOrder, 'f', 0, 64) {
						return fmt.Errorf("local display order %v does not equal expected %v for instance %d in team %d", local[display], inst.DisplayOrder, j, i)
					}
				}
			}
			return nil
		},
	)
}

func testAccVerifyRemoteView(view *structs.ViewInfo) resource.TestCheckFunc {
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

		// Get remote state of view
		remote, err := api.ReadView(id, getMap())
		if err != nil {
			return err
		}

		// Check for equality on fields directly in view
		if remote.Name != view.Name || remote.Status != view.Status || remote.Description != view.Description {
			return fmt.Errorf("remote state does not equal expected for view itself")
		}

		// Check equality on app fields
		for i, app := range view.Applications {
			if remote.Applications[i].ID != app.ID || remote.Applications[i].Name != app.Name ||
				remote.Applications[i].URL != app.URL || remote.Applications[i].Icon != app.Icon ||
				remote.Applications[i].Embeddable != app.Embeddable || remote.Applications[i].LoadInBackground != app.LoadInBackground {
				return fmt.Errorf("expected does not equal actual remote state for application %d", i)
			}
		}

		teams := remote.Teams
		sort.Slice(teams, func(i, j int) bool {
			return teams[i].Name.(string) < teams[j].Name.(string)
		})
		// Get rid of admin team
		teams = teams[1:]
		remote.Teams = teams

		for _, team := range remote.Teams {
			sort.Slice(team.Users, func(i, j int) bool {
				return team.Users[i].ID < team.Users[j].ID
			})
			sort.Slice(team.AppInstances, func(i, j int) bool {
				return team.AppInstances[i].Name < team.AppInstances[j].Name
			})
		}

		for i, team := range remote.Teams {
			if team.Name != view.Teams[i].Name || team.Role != view.Teams[i].Role ||
				!reflect.DeepEqual(team.Permissions, view.Teams[i].Permissions) {
				return fmt.Errorf("expected does not equal actual remote state for team %d", i)
			}

			// Check user equality
			for j, user := range team.Users {
				expectedRole := util.Ternary(view.Teams[i].Users[j].Role == nil, "", view.Teams[i].Users[j].Role)
				if user.ID != view.Teams[i].Users[j].ID || user.Role != expectedRole {
					return fmt.Errorf("expected does not equal actual remote state for user %d of team %d", j, i)
				}
			}

			// Check instance equality
			for j, inst := range team.AppInstances {
				if inst.Name != view.Teams[i].AppInstances[j].Name || inst.DisplayOrder != view.Teams[i].AppInstances[j].DisplayOrder {
					return fmt.Errorf("expected does not equal actual remote state for app instance %d of team %d", j, i)
				}
			}
		}
		return nil
	}
}

func getLocalState(s *terraform.State) map[string]string {
	// Need to change this line if we test multiple views
	mod := s.Modules[0]
	str := fmt.Sprintf("%+v", mod)
	str = strings.TrimSpace(str)

	lines := strings.Split(str, "\n")
	lines = lines[1:]
	localView := make(map[string]string)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		split := strings.Split(line, "=")
		// If a property is null, there will be nothing to the right of the =
		if split[1] == "" {
			localView[strings.TrimSpace(split[0])] = "null"
		} else {
			localView[strings.TrimSpace(split[0])] = strings.TrimSpace(split[1])
		}
	}
	return localView

}

