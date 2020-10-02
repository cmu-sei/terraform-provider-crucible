package provider_test

import (
	"crucible_provider/internal/structs"
	"crucible_provider/internal/util"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

// This file will hold the global variables needed by the various test functions. It will also set up these globals
// by reading the config. Unfortunately Go does not support file scoped variables for some reason so other test files
// will be reading the globals from this file :(

// Configuration strings. These are the things normally found in a .tf file

// Config strings that map to the provider itself
var correctCreds string
var incorrectCreds string

// Config strings that map to a VM resource
var configVMNormal string
var configVMFirst string
var configVMSecond string
var configVMIncorrectUserID string
var configVMNormalUpdated string
var configVMFirstUpdated string
var configVMSecondUpdated string
var configVMMultiTeams string

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

	configVMNormal = getVMResource("configVMNormal", &asMap)
	configVMFirst = getVMResource("configVMFirst", &asMap)
	configVMSecond = getVMResource("configVMSecond", &asMap)
	configVMIncorrectUserID = getVMResource("configVMIncorrectUserID", &asMap)
	configVMNormalUpdated = getVMResource("configVMNormalUpdated", &asMap)
	configVMFirstUpdated = getVMResource("configVMFirstUpdated", &asMap)
	configVMSecondUpdated = getVMResource("configVMSecondUpdated", &asMap)
	configVMMultiTeams = getVMResource("configVMMultiTeams", &asMap)

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
				Role:        "TestRole",
				Permissions: []string{"19e7abe6-3a07-4a24-b86d-cf00ef7e7c2b", "f45e79da-7c9d-45d0-8e3a-0786eb8681bf"},
			},
			{
				Name:        "foo",
				Role:        nil,
				Permissions: []string{"19e7abe6-3a07-4a24-b86d-cf00ef7e7c2b"},
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
				Role:        "TestRole",
				Permissions: []string{"19e7abe6-3a07-4a24-b86d-cf00ef7e7c2b", "f45e79da-7c9d-45d0-8e3a-0786eb8681bf"},
			},
			{
				Name:        "fooUpdated",
				Role:        nil,
				Permissions: []string{"19e7abe6-3a07-4a24-b86d-cf00ef7e7c2b"},
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
				Role:        "TestRole",
				Permissions: []string{"19e7abe6-3a07-4a24-b86d-cf00ef7e7c2b", "f45e79da-7c9d-45d0-8e3a-0786eb8681bf"},
				Users: []structs.UserInfo{
					{
						ID:   "6fb5b293-668b-4eb6-b614-dfdd6b0e0acf",
						Role: "Super User",
					},
					{
						ID: "ad580e55-f5b3-4865-b3ed-acec087450a7",
					},
				},
			},
			{
				Name:        "foo",
				Role:        nil,
				Permissions: []string{"19e7abe6-3a07-4a24-b86d-cf00ef7e7c2b"},
				Users: []structs.UserInfo{
					{
						ID:   "6fb5b293-668b-4eb6-b614-dfdd6b0e0acf",
						Role: "Super User",
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
				Role:        "TestRole",
				Permissions: []string{"19e7abe6-3a07-4a24-b86d-cf00ef7e7c2b", "f45e79da-7c9d-45d0-8e3a-0786eb8681bf"},
				Users: []structs.UserInfo{
					{
						ID: "ad580e55-f5b3-4865-b3ed-acec087450a7",
					},
				},
			},
			{
				Name:        "foo",
				Role:        nil,
				Permissions: []string{"19e7abe6-3a07-4a24-b86d-cf00ef7e7c2b"},
				Users: []structs.UserInfo{
					{
						ID:   "6fb5b293-668b-4eb6-b614-dfdd6b0e0acf",
						Role: "Super User",
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
				Role:        "TestRole",
				Permissions: []string{"19e7abe6-3a07-4a24-b86d-cf00ef7e7c2b", "f45e79da-7c9d-45d0-8e3a-0786eb8681bf"},
				Users: []structs.UserInfo{
					{
						ID:   "6fb5b293-668b-4eb6-b614-dfdd6b0e0acf",
						Role: "ebf35fab-eaa6-435b-aa0c-566056a56fba",
					},
					{
						ID: "ad580e55-f5b3-4865-b3ed-acec087450a7",
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
				Role:        nil,
				Permissions: []string{"19e7abe6-3a07-4a24-b86d-cf00ef7e7c2b"},
				Users: []structs.UserInfo{
					{
						ID:   "6fb5b293-668b-4eb6-b614-dfdd6b0e0acf",
						Role: "Super User",
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
				Role:        "TestRole",
				Permissions: []string{"19e7abe6-3a07-4a24-b86d-cf00ef7e7c2b", "f45e79da-7c9d-45d0-8e3a-0786eb8681bf"},
				Users: []structs.UserInfo{
					{
						ID:   "6fb5b293-668b-4eb6-b614-dfdd6b0e0acf",
						Role: "Super User",
					},
					{
						ID: "ad580e55-f5b3-4865-b3ed-acec087450a7",
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
				Role:        nil,
				Permissions: []string{"19e7abe6-3a07-4a24-b86d-cf00ef7e7c2b"},
				Users: []structs.UserInfo{
					{
						ID:   "6fb5b293-668b-4eb6-b614-dfdd6b0e0acf",
						Role: "Super User",
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
	}
	
	`, os.Getenv("TF_PROV_NAME"), os.Getenv("TF_USERNAME"), os.Getenv("TF_PASSWORD"), os.Getenv("TF_AUTH_URL"),
		os.Getenv("TF_TOK_URL"), os.Getenv("TF_CLIENT_ID"), os.Getenv("TF_CLIENT_SECRET"), os.Getenv("TF_VM_API_URL"),
		os.Getenv("TF_PLAYER_API_URL"))

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
	
	`, os.Getenv("TF_PROV_NAME"), os.Getenv("TF_USERNAME"), "foobarboz", os.Getenv("TF_AUTH_URL"),
		os.Getenv("TF_TOK_URL"), os.Getenv("TF_CLIENT_ID"), os.Getenv("TF_CLIENT_SECRET"), os.Getenv("TF_VM_API_URL"),
		os.Getenv("TF_PLAYER_API_URL"))
	return ret
}

func getVMResource(key string, asMap *map[string]interface{}) string {
	subMap := (*asMap)[key].(map[string]interface{})
	if subMap == nil {
		panic("Key not found in file")
	}

	teamIDs := subMap["team_ids"].([]interface{})
	teamIDsStrSlice := util.ToStringSlice(&teamIDs)

	for i, team := range *teamIDsStrSlice {
		(*teamIDsStrSlice)[i] = "\"" + team + "\""
	}

	ret := fmt.Sprintf(`resource "%s" "%s" {
		vm_id = "%s"
		url = "%s"
		name = "%s"
		user_id = "%s"
		team_ids = [%v]
	}
	
	`, subMap["provider_name"].(string), subMap["resourceName"].(string), subMap["vm_id"].(string), subMap["url"].(string),
		subMap["name"].(string), subMap["user_id"].(string), strings.Join(*teamIDsStrSlice, ","))

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

func getMap() map[string]string {
	// Set up the authentication info
	m := make(map[string]string)
	m["username"] = os.Getenv("TF_USERNAME")
	m["password"] = os.Getenv("TF_PASSWORD")
	m["auth_url"] = os.Getenv("TF_AUTH_URL")
	m["token_url"] = os.Getenv("TF_TOK_URL")
	m["client_id"] = os.Getenv("TF_CLIENT_ID")
	m["client_secret"] = os.Getenv("TF_CLIENT_SECRET")
	m["vm_api_url"] = os.Getenv("TF_VM_API_URL")
	m["player_api_url"] = os.Getenv("TF_PLAYER_API_URL")

	return m
}
