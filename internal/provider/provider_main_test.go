// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/cmu-sei/terraform-provider-crucible/internal/provider"
	"github.com/cmu-sei/terraform-provider-crucible/internal/structs"
	"github.com/cmu-sei/terraform-provider-crucible/internal/util"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories provides the Plugin Framework provider to the
// acceptance test harness via the v6 protocol.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"crucible": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// This file will hold the global variables needed by the various test functions. It will also set up these globals
// by reading the config. Unfortunately Go does not support file scoped variables for some reason so other test files
// will be reading the globals from this file :(

// Configuration strings. These are the things normally found in a .tf file

// Config strings that map to the provider itself
var correctCreds string
var incorrectCreds string

// VM resource configs are built inline per-test (see
// player_virtual_machine_server_test.go), each provisioning its own
// crucible_player_view to supply real team ids, so there are no VM config globals.

// Config strings for views
var configViewEmpty string
var configViewEmptyUpdated string
var configViewApps string
var configViewAppsUpdated string
var configViewTeams string
var configViewTeamsUpdated string
var configViewUsers string
var configViewUsersUpdated string
var configViewInstances string
var configViewInstancesUpdated string
var configViewInstancesNoInst string

// Structs representing expected state for views
var emptyViewExpected *structs.ViewInfo
var emptyViewExpectedUpdated *structs.ViewInfo
var appsViewExpected *structs.ViewInfo
var appsViewExpectedUpdated *structs.ViewInfo
var teamViewExpected *structs.ViewInfo
var teamViewExpectedUpdated *structs.ViewInfo
var userViewExpected *structs.ViewInfo
var userViewExpectedUpdated *structs.ViewInfo
var instanceViewExpected *structs.ViewInfo
var instanceViewExpectedUpdated *structs.ViewInfo

// Config strings for app templates
var configAppTemplate string
var configAppTemplateUpdated string

// Set up the globals
func init() {
	fp, err := os.Open("../../configs/testConfigs.json")
	if err != nil {
		panic(err)
	}
	defer fp.Close()

	bytes, err := ioutil.ReadAll(fp)
	if err != nil {
		panic(err)
	}

	var asMap map[string]interface{}
	json.Unmarshal([]byte(bytes), &asMap)

	correctCreds = getCreds()
	incorrectCreds = getIncorrectCreds()

	configViewEmpty = getViewResource("configViewEmpty", &asMap)
	configViewEmptyUpdated = getViewResource("configViewEmptyUpdated", &asMap)
	configViewApps = getViewResource("configViewApps", &asMap)
	configViewAppsUpdated = getViewResource("configViewAppsUpdated", &asMap)
	configViewTeams = getViewResource("configViewTeams", &asMap)
	configViewTeamsUpdated = getViewResource("configViewTeamsUpdated", &asMap)
	configViewUsers = getViewResource("configViewUsers", &asMap)
	configViewUsersUpdated = getViewResource("configViewUsersUpdated", &asMap)
	configViewInstances = getViewResource("configViewInstances", &asMap)
	configViewInstancesUpdated = getViewResource("configViewInstancesUpdated", &asMap)
	configViewInstancesNoInst = getViewResource("configViewInstancesNoInst", &asMap)

	configAppTemplate = getTemplateResource("configAppTemplate", &asMap)
	configAppTemplateUpdated = getTemplateResource("configAppTemplateUpdated", &asMap)

	emptyViewExpected = &structs.ViewInfo{
		Name:        "test",
		Description: "test empty view",
		Status:      "Active",
	}

	emptyViewExpectedUpdated = &structs.ViewInfo{
		Name:        "test",
		Description: "test empty view updated",
		Status:      "Active",
	}

	appsViewExpected = &structs.ViewInfo{
		Name:        "test",
		Description: "test view with apps",
		Status:      "Active",
		Applications: []structs.AppInfo{
			{
				Name:             "testApp",
				URL:              nil,
				Icon:             nil,
				Embeddable:       "true",
				LoadInBackground: "true",
				AppTemplateID:    nil,
			},
			{
				Name:             "testApp2",
				URL:              nil,
				Icon:             nil,
				Embeddable:       "false",
				LoadInBackground: "false",
				AppTemplateID:    nil,
			},
		},
	}

	appsViewExpectedUpdated = &structs.ViewInfo{
		Name:        "test",
		Description: "test view with apps",
		Status:      "Active",
		Applications: []structs.AppInfo{
			{
				Name:             "testAppUpdated",
				URL:              nil,
				Icon:             nil,
				Embeddable:       "true",
				LoadInBackground: "true",
				AppTemplateID:    nil,
			},
			{
				Name:             "testApp2Updated",
				URL:              nil,
				Icon:             nil,
				Embeddable:       "false",
				LoadInBackground: "false",
				AppTemplateID:    nil,
			},
		},
	}

	teamViewExpected = &structs.ViewInfo{
		Name:        "test",
		Description: "test view with teams",
		Status:      "Active",
		Applications: []structs.AppInfo{
			{
				Name:             "testApp",
				URL:              nil,
				Icon:             nil,
				Embeddable:       "true",
				LoadInBackground: "true",
				AppTemplateID:    nil,
			},
			{
				Name:             "testApp2",
				URL:              nil,
				Icon:             nil,
				Embeddable:       "false",
				LoadInBackground: "false",
				AppTemplateID:    nil,
			},
		},
		Teams: []structs.TeamInfo{
			{
				Name:        "bar",
				Role:        "View Member",
				Permissions: []string{"3b135496-c7d9-4bef-b60c-fbcfa1af9c1b", "7be07cd5-104e-4770-800b-80ac26cda6d5"},
			},
			{
				Name:        "foo",
				Role:        "View Member",
				Permissions: []string{"3b135496-c7d9-4bef-b60c-fbcfa1af9c1b"},
			},
		},
	}

	teamViewExpectedUpdated = &structs.ViewInfo{
		Name:        "test",
		Description: "test view with teams",
		Status:      "Active",
		Applications: []structs.AppInfo{
			{
				Name:             "testApp",
				URL:              nil,
				Icon:             nil,
				Embeddable:       "true",
				LoadInBackground: "true",
				AppTemplateID:    nil,
			},
			{
				Name:             "testApp2",
				URL:              nil,
				Icon:             nil,
				Embeddable:       "false",
				LoadInBackground: "false",
				AppTemplateID:    nil,
			},
		},
		Teams: []structs.TeamInfo{
			{
				Name:        "barUpdated",
				Role:        "View Member",
				Permissions: []string{"3b135496-c7d9-4bef-b60c-fbcfa1af9c1b", "7be07cd5-104e-4770-800b-80ac26cda6d5"},
			},
			{
				Name:        "fooUpdated",
				Role:        "View Member",
				Permissions: []string{"3b135496-c7d9-4bef-b60c-fbcfa1af9c1b"},
			},
		},
	}

	userViewExpected = &structs.ViewInfo{
		Name:        "test",
		Description: "test view with users",
		Status:      "Active",
		Applications: []structs.AppInfo{
			{
				Name:             "testApp",
				URL:              nil,
				Icon:             nil,
				Embeddable:       "true",
				LoadInBackground: "true",
				AppTemplateID:    nil,
			},
			{
				Name:             "testApp2",
				URL:              nil,
				Icon:             nil,
				Embeddable:       "false",
				LoadInBackground: "false",
				AppTemplateID:    nil,
			},
		},
		Teams: []structs.TeamInfo{
			{
				Name:        "bar",
				Role:        "View Member",
				Permissions: []string{"3b135496-c7d9-4bef-b60c-fbcfa1af9c1b", "7be07cd5-104e-4770-800b-80ac26cda6d5"},
				Users: []structs.UserInfo{
					{
						ID:   "9b3b331c-10c1-448b-8114-21b2586d8e38",
						Role: nil,
					},
					{
						ID: "3d8332fa-2e35-4823-b83f-732eb9483690",
					},
				},
			},
			{
				Name:        "foo",
				Role:        "View Member",
				Permissions: []string{"3b135496-c7d9-4bef-b60c-fbcfa1af9c1b"},
				Users: []structs.UserInfo{
					{
						ID:   "9b3b331c-10c1-448b-8114-21b2586d8e38",
						Role: nil,
					},
				},
			},
		},
	}

	userViewExpectedUpdated = &structs.ViewInfo{
		Name:        "test",
		Description: "test view with users",
		Status:      "Active",
		Applications: []structs.AppInfo{
			{
				Name:             "testApp",
				URL:              nil,
				Icon:             nil,
				Embeddable:       "true",
				LoadInBackground: "true",
				AppTemplateID:    nil,
			},
			{
				Name:             "testApp2",
				URL:              nil,
				Icon:             nil,
				Embeddable:       "false",
				LoadInBackground: "false",
				AppTemplateID:    nil,
			},
		},
		Teams: []structs.TeamInfo{
			{
				Name:        "bar",
				Role:        "View Member",
				Permissions: []string{"3b135496-c7d9-4bef-b60c-fbcfa1af9c1b", "7be07cd5-104e-4770-800b-80ac26cda6d5"},
				Users: []structs.UserInfo{
					{
						ID: "3d8332fa-2e35-4823-b83f-732eb9483690",
					},
				},
			},
			{
				Name:        "foo",
				Role:        "View Member",
				Permissions: []string{"3b135496-c7d9-4bef-b60c-fbcfa1af9c1b"},
				Users: []structs.UserInfo{
					{
						ID:   "9b3b331c-10c1-448b-8114-21b2586d8e38",
						Role: nil,
					},
				},
			},
		},
	}

	instanceViewExpected = &structs.ViewInfo{
		Name:        "test",
		Description: "test view with instances",
		Status:      "Active",
		Applications: []structs.AppInfo{
			{
				Name:             "testApp",
				URL:              nil,
				Icon:             nil,
				Embeddable:       "true",
				LoadInBackground: "true",
				AppTemplateID:    nil,
			},
			{
				Name:             "testApp2",
				URL:              nil,
				Icon:             nil,
				Embeddable:       "false",
				LoadInBackground: "false",
				AppTemplateID:    nil,
			},
		},
		Teams: []structs.TeamInfo{
			{
				Name:        "bar",
				Role:        "View Member",
				Permissions: []string{"3b135496-c7d9-4bef-b60c-fbcfa1af9c1b", "7be07cd5-104e-4770-800b-80ac26cda6d5"},
				Users: []structs.UserInfo{
					{
						ID:   "9b3b331c-10c1-448b-8114-21b2586d8e38",
						Role: nil,
					},
					{
						ID: "3d8332fa-2e35-4823-b83f-732eb9483690",
					},
				},
				AppInstances: []structs.AppInstance{
					{
						Name:         "testApp",
						DisplayOrder: 1,
					},
					{
						Name:         "testApp2",
						DisplayOrder: 0,
					},
				},
			},
			{
				Name:        "foo",
				Role:        "View Member",
				Permissions: []string{"3b135496-c7d9-4bef-b60c-fbcfa1af9c1b"},
				Users: []structs.UserInfo{
					{
						ID:   "9b3b331c-10c1-448b-8114-21b2586d8e38",
						Role: nil,
					},
				},
			},
		},
	}

	instanceViewExpectedUpdated = &structs.ViewInfo{
		Name:        "test",
		Description: "test view with instances",
		Status:      "Active",
		Applications: []structs.AppInfo{
			{
				Name:             "testApp",
				URL:              nil,
				Icon:             nil,
				Embeddable:       "true",
				LoadInBackground: "true",
				AppTemplateID:    nil,
			},
			{
				Name:             "testApp2",
				URL:              nil,
				Icon:             nil,
				Embeddable:       "false",
				LoadInBackground: "false",
				AppTemplateID:    nil,
			},
		},
		Teams: []structs.TeamInfo{
			{
				Name:        "bar",
				Role:        "View Member",
				Permissions: []string{"3b135496-c7d9-4bef-b60c-fbcfa1af9c1b", "7be07cd5-104e-4770-800b-80ac26cda6d5"},
				Users: []structs.UserInfo{
					{
						ID:   "9b3b331c-10c1-448b-8114-21b2586d8e38",
						Role: nil,
					},
					{
						ID: "3d8332fa-2e35-4823-b83f-732eb9483690",
					},
				},
				AppInstances: []structs.AppInstance{
					{
						Name:         "testApp",
						DisplayOrder: 0,
					},
					{
						Name:         "testApp2",
						DisplayOrder: 1,
					},
				},
			},
			{
				Name:        "foo",
				Role:        "View Member",
				Permissions: []string{"3b135496-c7d9-4bef-b60c-fbcfa1af9c1b"},
				Users: []structs.UserInfo{
					{
						ID:   "9b3b331c-10c1-448b-8114-21b2586d8e38",
						Role: nil,
					},
				},
			},
		},
	}
}

// Helper functions for setting up configs

func getCreds() string {
	ret := fmt.Sprintf(`provider "%s" {
		username = "%s"
		password = "%s"
		auth_url = "%s"
		token_url = "%s"
		client_id = "%s"
		client_secret = "%s"
		vm_api_url = "%s"
		player_api_url = "%s"
		caster_api_url = "%s"
	}

	`, testEnv("TF_PROV_NAME"), testEnv("TF_USERNAME"), testEnv("TF_PASSWORD"), testEnv("TF_AUTH_URL"),
		testEnv("TF_TOK_URL"), testEnv("TF_CLIENT_ID"), testEnv("TF_CLIENT_SECRET"), testEnv("TF_VM_API_URL"),
		testEnv("TF_PLAYER_API_URL"), testEnv("TF_CASTER_API_URL"))

	return ret
}

func getIncorrectCreds() string {
	ret := fmt.Sprintf(`provider "%s" {
		username = "%s"
		password = "%s"
		auth_url = "%s"
		token_url = "%s"
		client_id = "%s"
		client_secret = "%s"
		vm_api_url = "%s"
		player_api_url = "%s"
	}
	
	`, testEnv("TF_PROV_NAME"), testEnv("TF_USERNAME"), "foobarboz", testEnv("TF_AUTH_URL"),
		testEnv("TF_TOK_URL"), testEnv("TF_CLIENT_ID"), testEnv("TF_CLIENT_SECRET"), testEnv("TF_VM_API_URL"),
		testEnv("TF_PLAYER_API_URL"))
	return ret
}

func getViewResource(key string, file *map[string]interface{}) string {
	resource := (*file)[key].(map[string]interface{})
	if resource == nil {
		panic("key not found in file")
	}

	view := fmt.Sprintf(`resource "%s" "%s" {
		name = "%s"
		description = "%s"
		status = "%s"
		`, resource["provider_name"].(string), resource["resourceName"].(string), resource["name"].(string),
		resource["description"].(string), resource["status"].(string))

	// Consider applications
	if apps, ok := resource["applications"]; ok {
		appList := apps.([]interface{})

		for _, app := range appList {
			asMap := app.(map[string]interface{})

			// Handle optional arguments
			var url string
			if asMap["url"] != nil {
				url = "\"" + asMap["url"].(string) + "\""
			} else {
				url = "null"
			}

			var icon string
			if asMap["icon"] != nil {
				icon = "\"" + asMap["icon"].(string) + "\""
			} else {
				icon = "null"
			}

			var embeddable string
			if asMap["embeddable"] != nil {
				embeddable = "\"" + asMap["embeddable"].(string) + "\""
			} else {
				embeddable = "null"
			}

			var load_in_background string
			if asMap["load_in_background"] != nil {
				load_in_background = "\"" + asMap["load_in_background"].(string) + "\""
			} else {
				load_in_background = "null"
			}

			curr := fmt.Sprintf(`
			application {
				name = "%s"
				url = %s
				icon = %s
				embeddable = %s
				load_in_background = %s
				}
				`, asMap["name"].(string), url, icon, embeddable, load_in_background)
			view += curr
		}
		// Consider teams
		if teams, ok := resource["teams"]; ok {
			teamList := teams.([]interface{})
			for _, team := range teamList {
				asMap := team.(map[string]interface{})

				var name string
				if asMap["name"] != nil {
					name = "\"" + asMap["name"].(string) + "\""
				} else {
					name = "null"
				}

				var role string
				if asMap["role"] != nil {
					role = "\"" + asMap["role"].(string) + "\""
				} else {
					role = "null"
				}

				permissions := asMap["permissions"].([]interface{})
				permissionsStr := util.ToStringSlice(&permissions)
				for i, entry := range *permissionsStr {
					(*permissionsStr)[i] = "\"" + entry + "\""
				}

				curr := fmt.Sprintf(`
				team {
					name = %v
					role = %v
					permissions = [%v]
					`, name, role, strings.Join(*permissionsStr, ","))

				// Handle users
				if asMap["users"] != nil {
					users := asMap["users"].([]interface{})
					for _, user := range users {
						userMap := user.(map[string]interface{})

						var userRole string
						if userMap["role"] != nil {
							userRole = "\"" + userMap["role"].(string) + "\""
						} else {
							userRole = "null"
						}

						usr := fmt.Sprintf(`
						user {
							user_id = "%v"
							role = %v
						}
						`, userMap["user_id"], userRole)
						curr += usr
					}
				}

				// Handle app instances
				if asMap["app_instances"] != nil {
					instances := asMap["app_instances"].([]interface{})
					for _, inst := range instances {
						instMap := inst.(map[string]interface{})

						var display string
						if instMap["display_order"] != nil {
							display = strconv.FormatFloat(instMap["display_order"].(float64), 'f', 0, 64)
						} else {
							display = "null"
						}

						currInst := fmt.Sprintf(`
						app_instance {
							name = "%v"
							display_order = %v
						}
						`, instMap["name"], display)
						curr += currInst
					}
				}

				view += curr + "\n}"
			}
		}
	}
	return view + "\n}"
}

func getTemplateResource(key string, file *map[string]interface{}) string {
	resource := (*file)[key].(map[string]interface{})

	return fmt.Sprintf(`resource "%s" "%s" {
		name = "%s"
		url = "%s"
		icon = "%s"
		embeddable = %v
		load_in_background = %v
	}
	`, resource["provider_name"], resource["resourceName"], resource["name"], resource["url"], resource["icon"],
		resource["embeddable"], resource["load_in_background"])
}

// testEnvDefaults maps each TF_* test variable to the value used by the default
// crucible-development Aspire stack. Tests read these through testEnv(), so the
// acceptance suite runs out of the box against a local stack while any value can
// still be overridden by exporting the corresponding environment variable.
//
// TF_TEST_PROJECT_ID (the Caster project for crucible_vlan) has no stable dev
// default and is intentionally absent — that test skips until it is supplied.
var testEnvDefaults = map[string]string{
	"TF_PROV_NAME":      "crucible",
	"TF_USERNAME":       "admin",
	"TF_PASSWORD":       "admin",
	"TF_AUTH_URL":       "https://localhost:8443/realms/crucible/protocol/openid-connect/auth",
	"TF_TOK_URL":        "https://localhost:8443/realms/crucible/protocol/openid-connect/token",
	"TF_CLIENT_ID":      "crucible.provider",
	"TF_CLIENT_SECRET":  "", // crucible.provider is a public client
	"TF_PLAYER_API_URL": "http://localhost:4300/api",
	"TF_VM_API_URL":     "http://localhost:4302/api",
	"TF_CASTER_API_URL": "http://localhost:4309/api",
	// A dedicated, test-only user GUID. Player does not validate the id against
	// Keycloak, so the acceptance test can create and destroy it freely. (We do
	// NOT reuse the seeded admin GUID here — creating it 500s because it already
	// exists, and destroying it would remove the seeded admin.)
	"TF_TEST_USER_ID": "f1a9b2c3-0000-4d5e-8f60-acc7e57e0001",
	// The user_id stamped on the VM acceptance tests' VMs. Player's VM API stores
	// user_id free-form (no FK validation against Keycloak), so any fixed UUID
	// round-trips; a stable value keeps ImportStateVerify deterministic. Override
	// only to target a real user.
	"TF_TEST_VM_USER_ID": "8694c78c-1c49-421b-8ed8-689b46834878",
}

// testEnv returns the exported environment variable when set, otherwise the
// crucible-development default from testEnvDefaults (or "" if there is none).
func testEnv(key string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return testEnvDefaults[key]
}

// envOrDefault returns the environment variable value or the provided fallback
// when unset. Used for test values that have no entry in testEnvDefaults.
func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getMap() map[string]string {
	// Set up the authentication info
	m := make(map[string]string)
	m["username"] = testEnv("TF_USERNAME")
	m["password"] = testEnv("TF_PASSWORD")
	m["auth_url"] = testEnv("TF_AUTH_URL")
	// The OAuth2 config reads the token URL under the "player_token_url" key.
	m["player_token_url"] = testEnv("TF_TOK_URL")
	m["client_id"] = testEnv("TF_CLIENT_ID")
	m["client_secret"] = testEnv("TF_CLIENT_SECRET")
	m["vm_api_url"] = testEnv("TF_VM_API_URL")
	m["player_api_url"] = testEnv("TF_PLAYER_API_URL")
	m["caster_api_url"] = testEnv("TF_CASTER_API_URL")

	return m
}
