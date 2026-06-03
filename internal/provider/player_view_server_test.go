// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider_test

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/cmu-sei/terraform-provider-crucible/internal/api"
	"github.com/cmu-sei/terraform-provider-crucible/internal/structs"
	"github.com/cmu-sei/terraform-provider-crucible/internal/util"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccViewDestroyed,
		Steps: []resource.TestStep{
			{
				// Create resource and check
				Config: correctCreds + configViewEmpty,
				Check: resource.ComposeTestCheckFunc(
					testAccVerifyLocalView("crucible_player_view.empty", emptyViewExpected),
					testAccVerifyRemoteView(emptyViewExpected),
				),
			},
			{
				// Update resource and check
				Config: correctCreds + configViewEmptyUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccVerifyLocalView("crucible_player_view.empty", emptyViewExpectedUpdated),
					testAccVerifyRemoteView(&structs.ViewInfo{
						Name:        "test",
						Description: "test empty view updated",
						Status:      "Active",
					}),
				),
			},
			{
				// Import an empty view by id and confirm it succeeds. Nested
				// blocks carry computed ids, so a full ImportStateVerify is left
				// to per-attribute checks above; this exercises the import path.
				ResourceName:      "crucible_player_view.empty",
				ImportState:       true,
				ImportStateVerify: false,
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
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccViewDestroyed,
		Steps: []resource.TestStep{
			{
				Config: correctCreds + configViewApps,
				Check: resource.ComposeTestCheckFunc(
					testAccVerifyLocalView("crucible_player_view.apps", appsViewExpected),
					testAccVerifyRemoteView(appsViewExpected),
				),
			},
			{
				Config: correctCreds + configViewAppsUpdated,
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
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccViewDestroyed,
		Steps: []resource.TestStep{
			{
				Config: correctCreds + configViewTeams,
				Check: resource.ComposeTestCheckFunc(
					testAccVerifyLocalView("crucible_player_view.teams", teamViewExpected),
					testAccVerifyRemoteView(teamViewExpected)),
			},
			{
				Config: correctCreds + configViewTeamsUpdated,
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
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccViewDestroyed,
		Steps: []resource.TestStep{
			{
				Config: correctCreds + configViewUsers,
				Check: resource.ComposeTestCheckFunc(
					testAccVerifyLocalView("crucible_player_view.users", userViewExpected),
					testAccVerifyRemoteView(userViewExpected),
				),
			},
			{
				Config: correctCreds + configViewUsersUpdated,
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
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccViewDestroyed,
		Steps: []resource.TestStep{
			// View with 2 app instances
			{
				Config: correctCreds + configViewInstances,
				Check: resource.ComposeTestCheckFunc(
					testAccVerifyLocalView("crucible_player_view.instances", instanceViewExpected),
					testAccVerifyRemoteView(instanceViewExpected),
				),
			},
			// Remove the app instances (same resource, updated in place).
			{
				Config: correctCreds + configViewInstancesNoInst,
				Check: resource.ComposeTestCheckFunc(
					testAccVerifyLocalView("crucible_player_view.instances", userViewExpected),
					testAccVerifyRemoteView(userViewExpected),
				),
			},
			// Add them back
			{
				Config: correctCreds + configViewInstances,
				Check: resource.ComposeTestCheckFunc(
					testAccVerifyLocalView("crucible_player_view.instances", instanceViewExpected),
					testAccVerifyRemoteView(instanceViewExpected),
				),
			},
			// Update the instances
			{
				Config: correctCreds + configViewInstancesUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccVerifyLocalView("crucible_player_view.instances", instanceViewExpectedUpdated),
					testAccVerifyRemoteView(instanceViewExpectedUpdated),
				),
			},
		},
	})
}

// -------------------- Helper functions --------------------

// testAccViewDestroyed asserts that every crucible_player_view resource left in
// state no longer exists in the Player API after destroy.
func testAccViewDestroyed(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "crucible_player_view" {
			continue
		}
		exists, err := api.ViewExists(rs.Primary.ID, getMap())
		if err != nil {
			return fmt.Errorf("error checking destroyed view %s: %w", rs.Primary.ID, err)
		}
		if exists {
			return fmt.Errorf("view %s still exists after destroy", rs.Primary.ID)
		}
	}
	return nil
}

// ifaceToStateString renders an application's string-or-bool attribute the way
// it appears in Terraform state: "null" when absent, the string verbatim, or a
// "true"/"false" rendering of a bool.
func ifaceToStateString(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return "null"
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	default:
		return fmt.Sprintf("%v", t)
	}
}

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

				// embeddable / load_in_background are string-typed attributes on
				// the application block, so the fixtures carry string values.
				// Accept either a string or a bool to be safe.
				appEmbed := ifaceToStateString(app.Embeddable)
				appLoad := ifaceToStateString(app.LoadInBackground)

				var appTemplate string
				if app.AppTemplateID == nil {
					appTemplate = "null"
				} else {
					appTemplate = app.AppTemplateID.(string)
				}

				// app_id and the view id are server-assigned and unknowable at
				// compile time, so they are not compared against a fixed expected
				// value; we only assert app_id is populated in state.
				if localView[id] == "" || localView[id] == "null" {
					return fmt.Errorf("local state of application %d has no app_id", i)
				}
				if localView[templateId] != appTemplate ||
					localView[icon] != appIcon || localView[load] != appLoad || localView[name] != appName || localView[url] != appURL ||
					localView[embed] != appEmbed {
					return fmt.Errorf("local state of application %d does not match expected:\n  template: got %q want %q\n  icon: got %q want %q\n  load: got %q want %q\n  name: got %q want %q\n  url: got %q want %q\n  embed: got %q want %q",
						i, localView[templateId], appTemplate, localView[icon], appIcon,
						localView[load], appLoad, localView[name], appName, localView[url], appURL, localView[embed], appEmbed)
				}

			}
			return nil
		},
		// Check state of teams
		func(s *terraform.State) error {
			local := getLocalState(s)

			for i, team := range view.Teams {
				name := "team." + strconv.Itoa(i) + ".name"
				role := "team." + strconv.Itoa(i) + ".role"
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
					locRole := "team." + strconv.Itoa(i) + ".user." + strconv.Itoa(j) + ".role"

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

		// Check equality on app fields. The app ID is server-assigned, so it is
		// not compared. embeddable/load_in_background may come back as a bool from
		// the API while the fixtures use strings, so normalize both sides.
		for i, app := range view.Applications {
			if remote.Applications[i].Name != app.Name ||
				remote.Applications[i].URL != app.URL || remote.Applications[i].Icon != app.Icon ||
				ifaceToStateString(remote.Applications[i].Embeddable) != ifaceToStateString(app.Embeddable) ||
				ifaceToStateString(remote.Applications[i].LoadInBackground) != ifaceToStateString(app.LoadInBackground) {
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
			// Permissions come back from the API in arbitrary order; compare as
			// sets. The team Role is resolved to a name by ReadView.
			if ifaceString(team.Name) != ifaceString(view.Teams[i].Name) || ifaceString(team.Role) != ifaceString(view.Teams[i].Role) ||
				!sameStringSet(team.Permissions, view.Teams[i].Permissions) {
				return fmt.Errorf("expected does not equal actual remote state for team %d", i)
			}

			// Check user equality. Order the remote users to match the expected
			// (configured) order by id so the comparison is position-independent.
			expectedUsers := view.Teams[i].Users
			remoteUsers := append([]structs.UserInfo{}, team.Users...)
			expectRank := map[string]int{}
			for k, u := range expectedUsers {
				expectRank[u.ID] = k
			}
			sort.SliceStable(remoteUsers, func(a, b int) bool {
				return expectRank[remoteUsers[a].ID] < expectRank[remoteUsers[b].ID]
			})
			for j, user := range remoteUsers {
				if user.ID != expectedUsers[j].ID || ifaceString(user.Role) != ifaceString(expectedUsers[j].Role) {
					return fmt.Errorf("expected does not equal actual remote state for user %d of team %d", j, i)
				}
			}

			// Check instance equality (order-independent by name).
			expectedInsts := view.Teams[i].AppInstances
			remoteInsts := append([]structs.AppInstance{}, team.AppInstances...)
			instRank := map[string]int{}
			for k, in := range expectedInsts {
				instRank[in.Name] = k
			}
			sort.SliceStable(remoteInsts, func(a, b int) bool {
				return instRank[remoteInsts[a].Name] < instRank[remoteInsts[b].Name]
			})
			for j, inst := range remoteInsts {
				if inst.Name != expectedInsts[j].Name || inst.DisplayOrder != expectedInsts[j].DisplayOrder {
					return fmt.Errorf("expected does not equal actual remote state for app instance %d of team %d", j, i)
				}
			}
		}
		return nil
	}
}

// ifaceString coerces a nil-or-string interface into a string.
func ifaceString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// sameStringSet reports whether two string slices contain the same elements,
// ignoring order.
func sameStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	counts := map[string]int{}
	for _, s := range a {
		counts[s]++
	}
	for _, s := range b {
		counts[s]--
	}
	for _, c := range counts {
		if c != 0 {
			return false
		}
	}
	return true
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
