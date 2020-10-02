/*
Crucible
Copyright 2020 Carnegie Mellon University.
NO WARRANTY. THIS CARNEGIE MELLON UNIVERSITY AND SOFTWARE ENGINEERING INSTITUTE MATERIAL IS FURNISHED ON AN "AS-IS" BASIS. CARNEGIE MELLON UNIVERSITY MAKES NO WARRANTIES OF ANY KIND, EITHER EXPRESSED OR IMPLIED, AS TO ANY MATTER INCLUDING, BUT NOT LIMITED TO, WARRANTY OF FITNESS FOR PURPOSE OR MERCHANTABILITY, EXCLUSIVITY, OR RESULTS OBTAINED FROM USE OF THE MATERIAL. CARNEGIE MELLON UNIVERSITY DOES NOT MAKE ANY WARRANTY OF ANY KIND WITH RESPECT TO FREEDOM FROM PATENT, TRADEMARK, OR COPYRIGHT INFRINGEMENT.
Released under a MIT (SEI)-style license, please see license.txt or contact permission@sei.cmu.edu for full terms.
[DISTRIBUTION STATEMENT A] This material has been approved for public release and unlimited distribution.  Please see Copyright notice for non-US Government use and distribution.
Carnegie Mellon(R) and CERT(R) are registered in the U.S. Patent and Trademark Office by Carnegie Mellon University.
DM20-0181
*/

package api

import (
	"bytes"
	"crucible_provider/internal/structs"
	"crucible_provider/internal/util"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// CreateTeams creates teams in the specified view
//
// param teams the teams to create
//
// param viewID: the view to create the teams within
//
// param m map: containing provider config info
//
// Returns some error on failure or nil on success
func CreateTeams(teams *[]*structs.TeamInfo, viewID string, m map[string]string) error {
	log.Printf("! At top of API wrapper to create teams")

	auth, err := util.GetAuth(m)
	if err != nil {
		return err
	}

	// Create a new team for each entry in the slice of structs
	for i, team := range *teams {
		// We don't want the ID field in the request, so make struct into map and remove that key
		asMap := team.ToMap()
		delete(asMap, "id")

		// Need to look up the role by name
		role := asMap["role"]
		delete(asMap, "role")

		log.Printf("! Team's role: %v", role)
		if role.(string) != "" {
			roleID, err := getRoleByName(role.(string), auth, m)
			if err != nil {
				return err
			}

			// API wasn't seeing role_id, rename to roleId
			asMap["roleId"] = roleID
		}

		asJSON, err := json.Marshal(asMap)
		if err != nil {
			return err
		}

		log.Printf("! Team being created: %+v", asMap)

		url := m["player_api_url"] + "views/" + viewID + "/teams"
		request, err := http.NewRequest("POST", url, bytes.NewBuffer(asJSON))
		if err != nil {
			return err
		}
		request.Header.Add("Authorization", "Bearer "+auth)
		request.Header.Set("Content-Type", "application/json")
		client := &http.Client{}

		response, err := client.Do(request)
		if err != nil {
			return err
		}

		status := response.StatusCode
		if status != http.StatusCreated {
			return fmt.Errorf("player API returned with status code %d when creating team. %d teams created before error", status, i)
		}

		// Get the id of the team from the response
		body := make(map[string]interface{})
		err = json.NewDecoder(response.Body).Decode(&body)
		if err != nil {
			return err
		}
		teamID := body["id"].(string)
		(*teams)[i].ID = teamID

		log.Printf("! Team creation response body: %+v", body)
		// Add each user to this team
		for _, user := range team.Users {
			err := addUser(user.ID, teamID, m)
			if err != nil {
				return err
			}
			log.Printf("! User's role: %v", user.Role)
			if user.Role.(string) != "" {
				err = SetUserRole(teamID, viewID, user, m)
				if err != nil {
					return err
				}
			}
		}

		// Add each application to this team
		for i, app := range team.AppInstances {
			id, err := AddApplication(app.Parent, teamID, app.DisplayOrder, m)
			if err != nil {
				return err
			}
			app.ID = id
			team.AppInstances[i] = app
		}
	}
	return nil
}

// UpdateTeams updates the specified teams.
//
// Param teams: the teams to update.
//
// param m map: containing provider config info.
//
// Returns some error on failure or nil on success.
func UpdateTeams(teams *[]*structs.TeamInfo, m map[string]string) error {
	log.Printf("! At top of API wrapper for updating team")
	auth, err := util.GetAuth(m)
	if err != nil {
		return err
	}

	// Update each team
	for i, team := range *teams {
		// Set up payload for PUT request
		roleID, err := getRoleByName(team.Role.(string), auth, m)
		if err != nil {
			return err
		}

		asJSON, err := json.Marshal(map[string]interface{}{
			"id":     team.ID,
			"name":   team.Name,
			"roleId": roleID,
		})
		if err != nil {
			return err
		}

		url := m["player_api_url"] + "teams/" + team.ID.(string)
		log.Printf("! Updating team. URL: %v", url)
		log.Printf("! Updating team. Payload: %+v", team)
		request, err := http.NewRequest("PUT", url, bytes.NewBuffer(asJSON))
		if err != nil {
			return err
		}
		request.Header.Add("Authorization", "Bearer "+auth)
		request.Header.Set("Content-Type", "application/json")
		client := &http.Client{}

		response, err := client.Do(request)
		if err != nil {
			return err
		}

		status := response.StatusCode
		if status != http.StatusOK {
			return fmt.Errorf("player API returned with status code %d when updating team. %d teams updated before error", status, i)
		}
	}
	return nil
}

// DeleteTeams deletes the teams specified.
//
// param ids: the IDs of the teams to delete
//
// param m map: containing provider config info
//
// Returns some error on failure or nil on success
func DeleteTeams(ids *[]string, m map[string]string) error {
	auth, err := util.GetAuth(m)
	if err != nil {
		return err
	}

	for i, id := range *ids {
		url := m["player_api_url"] + "teams/" + id
		request, err := http.NewRequest("DELETE", url, nil)
		if err != nil {
			return err
		}
		request.Header.Add("Authorization", "Bearer "+auth)
		client := &http.Client{}

		response, err := client.Do(request)
		if err != nil {
			return err
		}

		status := response.StatusCode
		if status != http.StatusNoContent {
			return fmt.Errorf("player API returned with status code %d when deleting team. %d teams deleted before error", status, i)
		}
	}
	return nil
}

// AddPermissionsToTeam adds each team's specified permissions to that team
//
// param teams: A slice of structs representing the teams
//
// param m map: containing provider config info
//
// Returns some error on failure or nil on success
func AddPermissionsToTeam(teams *[]*structs.TeamInfo, m map[string]string) error {
	log.Printf("! At top of API wrapper to add permissions to team")
	auth, err := util.GetAuth(m)
	if err != nil {
		return err
	}

	for _, team := range *teams {
		log.Printf("! Adding permission to team %+v", team)
		for _, perm := range team.Permissions {
			url := m["player_api_url"] + "teams/" + team.ID.(string) + "/permissions/" + perm
			request, err := http.NewRequest("POST", url, nil)
			if err != nil {
				return err
			}
			request.Header.Add("Authorization", "Bearer "+auth)

			client := &http.Client{}
			response, err := client.Do(request)
			if err != nil {
				return err
			}

			status := response.StatusCode
			if status != http.StatusOK {
				return fmt.Errorf("player API returned with status code %d when adding permission to team", status)
			}

		}
	}

	return nil
}

// UpdateTeamPermissions adds and removes the permissions specified from the teams specified
//
// param toAdd: map corresponding teams with lists of permissions to add
//
// param toRemove: map corresponding teams with lists of permissions to remove
//
// param m map: containing provider config info
//
// Returns some error on failure or nil on success
func UpdateTeamPermissions(toAdd, toRemove map[string][]string, m map[string]string) error {
	log.Printf("! At top of API wrapper to update a team's permissions")
	auth, err := util.GetAuth(m)
	if err != nil {
		return err
	}

	// Add permissions
	for team := range toAdd {
		for _, perm := range toAdd[team] {
			url := m["player_api_url"] + "teams/" + team + "/permissions/" + perm
			request, err := http.NewRequest("POST", url, nil)
			if err != nil {
				return err
			}

			request.Header.Add("Authorization", "Bearer "+auth)

			client := &http.Client{}
			response, err := client.Do(request)
			if err != nil {
				return err
			}

			status := response.StatusCode
			if status != http.StatusOK {
				return fmt.Errorf("player API returned with status code %d when adding permission to team", status)
			}
		}
	}

	// Remove permissions
	for team := range toRemove {
		for _, perm := range toRemove[team] {
			url := m["player_api_url"] + "teams/" + team + "/permissions/" + perm
			request, err := http.NewRequest("DELETE", url, nil)
			if err != nil {
				return err
			}

			request.Header.Add("Authorization", "Bearer "+auth)

			client := &http.Client{}
			response, err := client.Do(request)
			if err != nil {
				return err
			}

			status := response.StatusCode
			if status != http.StatusOK {
				return fmt.Errorf("player API returned with status code %d when adding permission to team", status)
			}
		}
	}

	return nil
}

// GetRoleByID returns the name of the role with the given ID
func GetRoleByID(role string, m map[string]string) (string, error) {
	auth, err := util.GetAuth(m)
	if err != nil {
		return "", err
	}

	url := m["player_api_url"] + "roles/" + role
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	request.Header.Add("Authorization", "Bearer "+auth)

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}

	status := response.StatusCode
	if status != http.StatusOK {
		return "", fmt.Errorf("player API returned with status code %d looking for role %v", status, role)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)
	asStr := buf.String()
	defer response.Body.Close()

	asMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(asStr), &asMap)
	if err != nil {
		return "", err
	}

	return asMap["name"].(string), nil
}

// -------------------- Helper functions --------------------

// Reads information for all teams in a view.
//
// param viewID: the view to look under
//
// Returns a list of teamInfo structs and an error value
func readTeams(viewID string, m map[string]string) (*[]structs.TeamInfo, error) {
	log.Printf("! At top of API wrapper to read teams")

	auth, err := util.GetAuth(m)
	if err != nil {
		return nil, err
	}

	url := m["player_api_url"] + "views/" + viewID + "/teams"
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Add("Authorization", "Bearer "+auth)

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	status := response.StatusCode
	if status != http.StatusOK {
		return nil, fmt.Errorf("player API returned with status code %d when reading teams", status)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)
	asStr := buf.String()
	defer response.Body.Close()

	asMap := new([]map[string]interface{})
	teams := new([]structs.TeamInfo)

	// Unmarshal into a map instead of team struct so we can handle the permissions field
	err = json.Unmarshal([]byte(asStr), asMap)
	if err != nil {
		return nil, err
	}

	log.Printf("! Remote team state as map: %+v", asMap)
	for _, team := range *asMap {
		permissions := new([]string)
		permissionsMaps := team["permissions"].([]interface{})
		for _, perm := range permissionsMaps {
			permMap := perm.(map[string]interface{})
			*permissions = append(*permissions, permMap["id"].(string))
		}

		*teams = append(*teams, structs.TeamInfo{
			ID:          team["id"],
			Name:        team["name"],
			Role:        team["roleName"],
			Permissions: *permissions,
		})
	}

	if err != nil {
		return nil, err
	}

	// Read the users for each team
	for i, team := range *teams {
		id := team.ID.(string)
		users, err := getUsersInTeam(id, viewID, m)
		if err != nil {
			return nil, err
		}
		team.Users = users
		(*teams)[i] = team
	}

	// Read the app instances for each team
	for i, team := range *teams {
		id := team.ID.(string)
		instances, err := getTeamAppInstances(id, m)
		if err != nil {
			return nil, err
		}
		team.AppInstances = *instances
		(*teams)[i] = team
	}

	log.Printf("! Returning from api, team structs are: %+v", teams)
	return teams, nil
}

// Returns the ID of the role with the given name
func getRoleByName(role, auth string, m map[string]string) (string, error) {
	url := m["player_api_url"] + "roles/name/" + role
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	request.Header.Add("Authorization", "Bearer "+auth)

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}

	status := response.StatusCode
	if status != http.StatusOK {
		return "", fmt.Errorf("player API returned with status code %d looking for role %v", status, role)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)
	asStr := buf.String()
	defer response.Body.Close()

	asMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(asStr), &asMap)
	if err != nil {
		return "", err
	}

	return asMap["id"].(string), nil
}

