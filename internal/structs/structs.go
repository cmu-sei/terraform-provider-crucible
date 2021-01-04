// Copyright 2021 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package structs

import (
	"crucible_provider/internal/util"
	"fmt"
	"log"
	"strconv"
	"strings"
)

// Structs used throughout provider

// VMInfo used as the payload for VM creation and as the return value for VM retrieval
type VMInfo struct {
	ID         string
	URL        string
	Name       string
	TeamIDs    []string
	UserID     interface{}
	Connection ConsoleConnection `json:"consoleConnectionInfo"`
}

// ConsoleConnection represents a console connection info block
type ConsoleConnection struct {
	Hostname string
	Port     string
	Protocol string
	Username string
	Password string
}

// ConnectionFromMap creates a ConsoleConnection object from an equivalent map
func ConnectionFromMap(m map[string]interface{}) ConsoleConnection {
	return ConsoleConnection{
		Hostname: m["hostname"].(string),
		Port:     m["port"].(string),
		Protocol: m["protocol"].(string),
		Username: m["username"].(string),
		Password: m["password"].(string),
	}
}

// ToMap turns a ConsoleConnection into an equivalent map.
func (conn ConsoleConnection) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"hostname": conn.Hostname,
		"port":     conn.Port,
		"protocol": conn.Protocol,
		"username": conn.Username,
		"password": conn.Password,
	}
}

// ViewInfo used as payload for view creation and return value for view retrieval
type ViewInfo struct {
	Name            string
	Description     string
	Status          string
	CreateAdminTeam bool
	Applications    []AppInfo  `json:"-"`
	Teams           []TeamInfo `json:"-"`
}

// ToMap converts a ViewInfo struct to a map
//
// Ignore the apps and teams field here b/c it's only used to call API
func (view *ViewInfo) ToMap() map[string]interface{} {
	ret := make(map[string]interface{}, 1)
	ret["name"] = view.Name
	ret["description"] = view.Description
	ret["status"] = view.Status
	ret["create_admin_team"] = view.CreateAdminTeam
	return ret
}

// AppInfo used as payload for application creation and return value for application retrieval
//
// Fields who's type is interface{} are optional. Interface{} is nullable but string is not, so we need to use
// interface{} to pass null values to the API
type AppInfo struct {
	ID               string
	Name             interface{}
	URL              interface{}
	Icon             interface{}
	Embeddable       interface{}
	LoadInBackground interface{}
	ViewID           string
	AppTemplateID    interface{} `json:"applicationTemplateId"`
}

// AppInfoFromMap returns an AppInfo struct read from a map object
func AppInfoFromMap(asMap map[string]interface{}) *AppInfo {
	return &AppInfo{
		// Required fields
		ID:     asMap["app_id"].(string),
		ViewID: asMap["v_id"].(string),
		// Everything else is optional and can thus be nil
		Name:             util.Ternary(asMap["name"].(string) == "", nil, asMap["name"]),
		URL:              util.Ternary(asMap["url"].(string) == "", nil, asMap["url"]),
		Icon:             util.Ternary(asMap["icon"].(string) == "", nil, asMap["icon"]),
		Embeddable:       util.Ternary(asMap["embeddable"].(string) == "", nil, strings.ReplaceAll(asMap["embeddable"].(string), `"`, "")),
		LoadInBackground: util.Ternary(asMap["load_in_background"].(string) == "", nil, strings.ReplaceAll(asMap["load_in_background"].(string), `"`, "")),
		AppTemplateID:    util.Ternary(asMap["app_template_id"].(string) == "", nil, asMap["app_template_id"]),
	}
}

// ToMap converts an appInfo struct to an equivalent map
func (app *AppInfo) ToMap() map[string]interface{} {
	ret := make(map[string]interface{})

	log.Printf("! App to make into a map: %+v", app)

	t := fmt.Sprintf("%T", app.Embeddable)
	// Can't do this with the ternary function b/c it isn't lazily evaluated
	var embed string
	if app.Embeddable == nil {
		embed = ""
	} else if t == "string" {
		embed = app.Embeddable.(string)
	} else {
		embed = strconv.FormatBool(app.Embeddable.(bool))
	}

	t = fmt.Sprintf("%T", app.LoadInBackground)
	var load string
	if app.LoadInBackground == nil {
		load = ""
	} else if t == "string" {
		load = app.LoadInBackground.(string)
	} else {
		load = strconv.FormatBool(app.LoadInBackground.(bool))
	}

	ret["app_id"] = app.ID
	ret["name"] = app.Name
	ret["url"] = app.URL
	ret["icon"] = app.Icon
	ret["embeddable"] = embed
	ret["load_in_background"] = load
	ret["v_id"] = app.ViewID
	ret["app_template_id"] = app.AppTemplateID

	log.Printf("! App as map: %+v", ret)
	return ret
}

// TeamInfo holds information about a team within a view. Used to create, read, and update teams within views
type TeamInfo struct {
	ID           interface{}
	Name         interface{}
	Role         interface{}
	Permissions  []string
	Users        []UserInfo
	AppInstances []AppInstance
}

// TeamInfoFromMap returns a TeamInfo struct from a map object
func TeamInfoFromMap(asMap map[string]interface{}) *TeamInfo {
	users := userInfoFromMap(asMap)
	if len(users) == 0 {
		users = nil
	}

	apps := AppInstanceFromMap(asMap)
	if len(apps) == 0 {
		apps = nil
	}

	permissions := asMap["permissions"].([]interface{})
	strPermissions := util.ToStringSlice(&permissions)

	return &TeamInfo{
		ID:           asMap["team_id"],
		Name:         asMap["name"],
		Role:         asMap["role"],
		Permissions:  *strPermissions,
		Users:        users,
		AppInstances: apps,
	}
}

// ToMap converts a TeamInfo struct into an equivalent map
func (team *TeamInfo) ToMap() map[string]interface{} {
	ret := make(map[string]interface{})

	userSlice := new([]map[string]interface{})
	if len(team.Users) == 0 {
		*userSlice = nil
	} else {
		for _, user := range team.Users {
			*userSlice = append(*userSlice, user.ToMap())
		}

	}
	instances := new([]map[string]interface{})
	if len(team.AppInstances) == 0 {
		*instances = nil
	} else {
		for _, inst := range team.AppInstances {
			*instances = append(*instances, inst.ToMap())
		}
	}

	ret["team_id"] = team.ID
	ret["name"] = team.Name
	ret["role"] = team.Role
	ret["user"] = *userSlice
	ret["permissions"] = team.Permissions
	ret["app_instance"] = *instances
	return ret
}

// UserInfo holds information about a user within a team. See PlayerUser for the representation of a
// user in general.
//
// ID is required, RoleID is optional
type UserInfo struct {
	ID   string
	Role interface{}
}

// Returns the list of structs representing the users in a team
func userInfoFromMap(asMap map[string]interface{}) []UserInfo {
	if asMap["user"] == nil {
		return nil
	}
	list := asMap["user"].([]interface{})
	ret := new([]UserInfo)

	for _, user := range list {
		userMap := user.(map[string]interface{})

		*ret = append(*ret, UserInfo{
			ID:   userMap["user_id"].(string),
			Role: userMap["role"],
		})
	}
	return *ret
}

// ToMap returns a map created from a userInfo struct
func (user *UserInfo) ToMap() map[string]interface{} {
	ret := make(map[string]interface{})
	ret["user_id"] = user.ID
	ret["role"] = user.Role
	return ret
}

// UserHasID takes a slice of userInfo structs and returns true if any of them
// have the specified ID. We can't just make a array contains function
// because Go doesn't have generics
func UserHasID(arr []UserInfo, id string) bool {
	for _, user := range arr {
		if user.ID == id {
			return true
		}
	}
	return false
}

// AppTemplate holds the information needed for CRUD operations on an ApplicationTemplate resource
type AppTemplate struct {
	Name             string
	URL              string
	Icon             string
	Embeddable       bool
	LoadInBackground bool
}

// AppInstance holds the info needed to manage application instances
type AppInstance struct {
	Name         string
	ID           string
	DisplayOrder float64 `json:"displayOrder"`
	Parent       string  `json:"applicationId"`
}

// AppInstanceFromMap creates an app instance object from a map
func AppInstanceFromMap(asMap map[string]interface{}) []AppInstance {
	if asMap["app_instance"] == nil {
		return nil
	}

	list := asMap["app_instance"].([]interface{})
	ret := new([]AppInstance)

	for _, app := range list {
		m := app.(map[string]interface{})

		var order float64
		if _, ok := m["display_order"]; ok {
			order = m["display_order"].(float64)
		} else {
			order = 0
		}

		log.Printf("! Reading app instance from map. Display order is %v", order)

		*ret = append(*ret, AppInstance{
			Name:         m["name"].(string),
			DisplayOrder: order,
			ID:           m["id"].(string),
		})
	}
	return *ret
}

// ToMap returns a map representation of an AppInstance struct
func (instance *AppInstance) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"name":          instance.Name,
		"id":            instance.ID,
		"display_order": instance.DisplayOrder,
	}
}

// InstanceHasID takes an array of AppInstances and returns whether one of them has the given ID.
func InstanceHasID(arr *[]AppInstance, id string) bool {
	for _, inst := range *arr {
		if inst.ID == id {
			return true
		}
	}
	return false
}

// PlayerUser represents a user outside of a team. IE one that simply exists within player. Used for the
// user resource type.
type PlayerUser struct {
	ID            string
	Name          string
	Role          interface{} `json:"roleId"`
	IsSystemAdmin bool
}

