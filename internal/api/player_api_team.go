// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/cmu-sei/terraform-provider-crucible/internal/playerclient"
	"github.com/cmu-sei/terraform-provider-crucible/internal/structs"
	"github.com/cmu-sei/terraform-provider-crucible/internal/util"
)

// This file adapts the provider's team operations onto the generated Player API
// client (internal/playerclient). Signatures unchanged. Role-name lookups
// (getRoleByName) live in player_api_user.go; getTeamRoleByName and readTeams
// live here.

// CreateTeams creates teams in the specified view, then adds their users and
// application instances.
func CreateTeams(teams *[]*structs.TeamInfo, viewID string, m map[string]string) error {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return err
	}

	vID, err := uuid.Parse(viewID)
	if err != nil {
		return err
	}

	for i, team := range *teams {
		body := playerclient.CreateTeamCommand{Name: ifaceStrPtr(team.Name)}
		if role := roleString(team.Role); role != "" {
			roleID, err := getTeamRoleByName(role, m)
			if err != nil {
				return err
			}
			rid, err := uuid.Parse(roleID)
			if err != nil {
				return err
			}
			body.RoleId = &rid
		}

		resp, err := client.CreateTeamWithResponse(context.Background(), vID, body)
		if err != nil {
			return err
		}
		if resp.StatusCode() != http.StatusCreated {
			return fmt.Errorf("player API returned with status code %d when creating team. %d teams created before error", resp.StatusCode(), i)
		}
		if resp.JSON201 == nil || resp.JSON201.Id == nil {
			return fmt.Errorf("player API returned status 201 with no team id when creating team")
		}
		teamID := resp.JSON201.Id.String()
		(*teams)[i].ID = teamID

		// Add each user to the team, and set its role if configured.
		for _, user := range team.Users {
			if err := addUser(user.ID, teamID, m); err != nil {
				return err
			}
			if roleString(user.Role) != "" {
				if err := SetUserRole(teamID, viewID, user, m); err != nil {
					return err
				}
			}
		}

		// Add each application instance to the team.
		for j, app := range team.AppInstances {
			id, err := AddApplication(app.Parent, teamID, app.DisplayOrder, m)
			if err != nil {
				return err
			}
			app.ID = id
			team.AppInstances[j] = app
		}
	}
	return nil
}

// UpdateTeams updates the specified teams' name and role.
func UpdateTeams(teams *[]*structs.TeamInfo, m map[string]string) error {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return err
	}

	for i, team := range *teams {
		roleID, err := getTeamRoleByName(roleString(team.Role), m)
		if err != nil {
			return err
		}
		rid, err := uuid.Parse(roleID)
		if err != nil {
			return err
		}
		teamID, err := uuid.Parse(util.As[string](team.ID))
		if err != nil {
			return err
		}

		resp, err := client.UpdateTeamWithResponse(context.Background(), teamID,
			playerclient.EditTeamCommand{Name: ifaceStrPtr(team.Name), RoleId: &rid})
		if err != nil {
			return err
		}
		if resp.StatusCode() != http.StatusOK {
			return fmt.Errorf("player API returned with status code %d when updating team. %d teams updated before error", resp.StatusCode(), i)
		}
	}
	return nil
}

// DeleteTeams deletes the specified teams.
func DeleteTeams(ids *[]string, m map[string]string) error {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return err
	}

	for i, id := range *ids {
		teamID, err := uuid.Parse(id)
		if err != nil {
			return err
		}
		resp, err := client.DeleteTeamWithResponse(context.Background(), teamID)
		if err != nil {
			return err
		}
		if resp.StatusCode() != http.StatusNoContent {
			return fmt.Errorf("player API returned with status code %d when deleting team. %d teams deleted before error", resp.StatusCode(), i)
		}
	}
	return nil
}

// AddPermissionsToTeam adds each team's specified permissions to that team.
func AddPermissionsToTeam(teams *[]*structs.TeamInfo, m map[string]string) error {
	for _, team := range *teams {
		teamID := util.As[string](team.ID)
		for _, perm := range team.Permissions {
			if err := addTeamPermission(teamID, perm, m); err != nil {
				return err
			}
		}
	}
	return nil
}

// UpdateTeamPermissions adds and removes the specified permissions from teams.
func UpdateTeamPermissions(toAdd, toRemove map[string][]string, m map[string]string) error {
	for team := range toAdd {
		for _, perm := range toAdd[team] {
			if err := addTeamPermission(team, perm, m); err != nil {
				return err
			}
		}
	}
	for team := range toRemove {
		for _, perm := range toRemove[team] {
			if err := removeTeamPermission(team, perm, m); err != nil {
				return err
			}
		}
	}
	return nil
}

// GetRoleByID returns the name of the role with the given id.
func GetRoleByID(role string, m map[string]string) (string, error) {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return "", err
	}

	roleID, err := uuid.Parse(role)
	if err != nil {
		return "", err
	}

	resp, err := client.GetRoleWithResponse(context.Background(), roleID)
	if err != nil {
		return "", err
	}
	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("player API returned with status code %d looking for role %v", resp.StatusCode(), role)
	}
	if resp.JSON200 == nil {
		return "", fmt.Errorf("player API returned status 200 with an empty body for role %v", role)
	}
	return derefStr(resp.JSON200.Name), nil
}

// -------------------- Helper functions --------------------

// addTeamPermission adds a single permission to a team.
func addTeamPermission(teamID, permID string, m map[string]string) error {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return err
	}
	tID, err := uuid.Parse(teamID)
	if err != nil {
		return err
	}
	pID, err := uuid.Parse(permID)
	if err != nil {
		return err
	}
	resp, err := client.AddTeamPermissionToTeamWithResponse(context.Background(), tID, pID)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("player API returned with status code %d when adding permission to team", resp.StatusCode())
	}
	return nil
}

// removeTeamPermission removes a single permission from a team.
func removeTeamPermission(teamID, permID string, m map[string]string) error {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return err
	}
	tID, err := uuid.Parse(teamID)
	if err != nil {
		return err
	}
	pID, err := uuid.Parse(permID)
	if err != nil {
		return err
	}
	resp, err := client.RemoveTeamPermissionFromTeamWithResponse(context.Background(), tID, pID)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("player API returned with status code %d when removing permission from team", resp.StatusCode())
	}
	return nil
}

// readTeams reads all teams in a view, including users, app instances, and
// permissions for each.
func readTeams(viewID string, m map[string]string) (*[]structs.TeamInfo, error) {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return nil, err
	}

	vID, err := uuid.Parse(viewID)
	if err != nil {
		return nil, err
	}

	resp, err := client.GetViewTeamsWithResponse(context.Background(), vID)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("player API returned with status code %d when reading teams", resp.StatusCode())
	}

	teams := new([]structs.TeamInfo)
	if resp.JSON200 == nil {
		return teams, nil
	}

	for _, t := range *resp.JSON200 {
		// Leave permissions nil (not an empty slice) when there are none: the
		// resource distinguishes a null permissions list from an empty one, and
		// returning []string{} here causes "inconsistent result after apply".
		var permissions []string
		if t.Permissions != nil {
			for _, p := range *t.Permissions {
				if p.Id != nil {
					permissions = append(permissions, p.Id.String())
				}
			}
		}
		info := structs.TeamInfo{
			Name:        derefStr(t.Name),
			Role:        derefStr(t.RoleName),
			Permissions: permissions,
		}
		if t.Id != nil {
			info.ID = t.Id.String()
		}
		*teams = append(*teams, info)
	}

	// Read users and app instances for each team.
	for i, team := range *teams {
		id := util.As[string](team.ID)
		users, err := getUsersInTeam(id, viewID, m)
		if err != nil {
			return nil, err
		}
		team.Users = users

		instances, err := getTeamAppInstances(id, m)
		if err != nil {
			return nil, err
		}
		team.AppInstances = *instances
		(*teams)[i] = team
	}

	return teams, nil
}

// getTeamRoleByName returns the id of the team-role with the given name.
func getTeamRoleByName(roleName string, m map[string]string) (string, error) {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return "", err
	}

	resp, err := client.GetTeamRolesWithResponse(context.Background())
	if err != nil {
		return "", err
	}
	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("player API returned with status code %d looking for role %v", resp.StatusCode(), roleName)
	}
	if resp.JSON200 != nil {
		for _, role := range *resp.JSON200 {
			if role.Name != nil && *role.Name == roleName && role.Id != nil {
				return role.Id.String(), nil
			}
		}
	}
	return "", fmt.Errorf("role %q not found in returned list", roleName)
}
