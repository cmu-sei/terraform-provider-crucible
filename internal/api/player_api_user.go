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
)

// This file adapts the provider's user and team-membership operations onto the
// generated Player API client (internal/playerclient). Signatures unchanged.
// The private helpers here (addUser, findMembershipID, getMembership,
// getUsersInTeam, getRoleByName) are also used by player_api_team.go.

// RemoveUsers removes the specified users from the specified teams.
func RemoveUsers(teamsToUsers map[string][]string, m map[string]string) error {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return err
	}

	for team := range teamsToUsers {
		teamID, err := uuid.Parse(team)
		if err != nil {
			return err
		}
		for _, user := range teamsToUsers[team] {
			userID, err := uuid.Parse(user)
			if err != nil {
				return err
			}
			resp, err := client.RemoveUserFromTeamWithResponse(context.Background(), teamID, userID)
			if err != nil {
				return err
			}
			if resp.StatusCode() != http.StatusOK {
				return fmt.Errorf("player API returned with status code %d when removing user from team", resp.StatusCode())
			}
		}
	}
	return nil
}

// AddUsersToTeam adds the specified users to the specified team.
func AddUsersToTeam(users *[]string, team string, m map[string]string) error {
	for _, user := range *users {
		if err := addUser(user, team, m); err != nil {
			return err
		}
	}
	return nil
}

// SetUserRole sets a user's role within their team membership.
func SetUserRole(teamID, viewID string, user structs.UserInfo, m map[string]string) error {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return err
	}

	// Find the relevant TeamMembership id for this user/view/team.
	membershipID, err := findMembershipID(user.ID, viewID, teamID, m)
	if err != nil {
		return err
	}

	// Resolve the role name to its id.
	roleID, err := getRoleByName(roleString(user.Role), m)
	if err != nil {
		return err
	}
	rid, err := uuid.Parse(roleID)
	if err != nil {
		return err
	}
	mID, err := uuid.Parse(membershipID)
	if err != nil {
		return err
	}

	resp, err := client.UpdateTeamMembershipWithResponse(context.Background(), mID,
		playerclient.EditTeamMembershipCommand{RoleId: &rid})
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("player API returned with status code %d when setting user role", resp.StatusCode())
	}
	return nil
}

// CreateUser creates a new Player user.
func CreateUser(user structs.PlayerUser, m map[string]string) error {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return err
	}

	userID, err := uuid.Parse(user.ID)
	if err != nil {
		return err
	}

	body := playerclient.CreateUserCommand{
		Id:   &userID,
		Name: strPtr(user.Name),
	}
	if r := roleString(user.Role); r != "" {
		roleID, err := getRoleByName(r, m)
		if err != nil {
			return err
		}
		rid, err := uuid.Parse(roleID)
		if err != nil {
			return err
		}
		body.RoleId = &rid
	}

	resp, err := client.CreateUserWithResponse(context.Background(), body)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusCreated {
		return fmt.Errorf("Error creating user in Player. API returned status code %d", resp.StatusCode())
	}
	return nil
}

// ReadUser returns a struct representing a given user.
func ReadUser(id string, m map[string]string) (*structs.PlayerUser, error) {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	resp, err := client.GetUserWithResponse(context.Background(), userID)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("Error reading user in Player. API returned status code %d", resp.StatusCode())
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("Player API returned status 200 with an empty body when reading user %s", id)
	}

	user := &structs.PlayerUser{Name: derefStr(resp.JSON200.Name)}
	if resp.JSON200.Id != nil {
		user.ID = resp.JSON200.Id.String()
	}
	// Role carries the role id (interface{}), matching the prior behavior where
	// the resource resolves it to a name on read.
	if resp.JSON200.RoleId != nil {
		user.Role = resp.JSON200.RoleId.String()
	}
	return user, nil
}

// UserExists returns true if a user exists.
func UserExists(id string, m map[string]string) (bool, error) {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return false, err
	}

	userID, err := uuid.Parse(id)
	if err != nil {
		return false, err
	}

	resp, err := client.GetUserWithResponse(context.Background(), userID)
	if err != nil {
		return false, err
	}
	return resp.StatusCode() != http.StatusNotFound, nil
}

// UpdateUser updates a user in Player.
func UpdateUser(user structs.PlayerUser, m map[string]string) error {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return err
	}

	userID, err := uuid.Parse(user.ID)
	if err != nil {
		return err
	}

	body := playerclient.EditUserCommand{Name: strPtr(user.Name)}
	if r := roleString(user.Role); r != "" {
		roleID, err := getRoleByName(r, m)
		if err != nil {
			return err
		}
		rid, err := uuid.Parse(roleID)
		if err != nil {
			return err
		}
		body.RoleId = &rid
	}

	resp, err := client.UpdateUserWithResponse(context.Background(), userID, body)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("Error updating user in Player. API returned status code %d", resp.StatusCode())
	}
	return nil
}

// DeleteUser deletes a user.
func DeleteUser(id string, m map[string]string) error {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return err
	}

	userID, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	resp, err := client.DeleteUserWithResponse(context.Background(), userID)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusNoContent {
		return fmt.Errorf("Error deleting user in Player. API returned status code %d", resp.StatusCode())
	}
	return nil
}

// ---------------------- Private helpers (shared with player_api_team.go) ----------------------

// addUser adds a single user to a team.
func addUser(userID, teamID string, m map[string]string) error {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return err
	}

	uID, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	tID, err := uuid.Parse(teamID)
	if err != nil {
		return err
	}

	resp, err := client.AddUserToTeamWithResponse(context.Background(), tID, uID)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("player API returned with status code %d when adding user to team", resp.StatusCode())
	}
	return nil
}

// findMembershipID returns the team-membership id for the given user/view/team.
func findMembershipID(userID, viewID, teamID string, m map[string]string) (string, error) {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return "", err
	}

	uID, err := uuid.Parse(userID)
	if err != nil {
		return "", err
	}
	vID, err := uuid.Parse(viewID)
	if err != nil {
		return "", err
	}

	resp, err := client.GetTeamMembershipsWithResponse(context.Background(), uID, vID)
	if err != nil {
		return "", err
	}
	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("player API returned with status code %d when looking for teamMembership id", resp.StatusCode())
	}
	if resp.JSON200 == nil {
		return "", fmt.Errorf("no membership found for the given user and view")
	}
	for _, ms := range *resp.JSON200 {
		if ms.TeamId != nil && ms.TeamId.String() == teamID && ms.Id != nil {
			return ms.Id.String(), nil
		}
	}
	return "", fmt.Errorf("no membership found for the given user and view")
}

// getMembership returns the role name for the team-membership with the given id.
func getMembership(id string, m map[string]string) (string, error) {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return "", err
	}

	mID, err := uuid.Parse(id)
	if err != nil {
		return "", err
	}

	resp, err := client.GetTeamMembershipWithResponse(context.Background(), mID)
	if err != nil {
		return "", err
	}
	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("player API returned with status code %d when getting TeamMembership", resp.StatusCode())
	}
	if resp.JSON200 == nil {
		return "", nil
	}
	return derefStr(resp.JSON200.RoleName), nil
}

// getUsersInTeam returns all users in a team, each with its membership role name.
func getUsersInTeam(teamID, viewID string, m map[string]string) ([]structs.UserInfo, error) {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return nil, err
	}

	tID, err := uuid.Parse(teamID)
	if err != nil {
		return nil, err
	}

	resp, err := client.GetTeamUsersWithResponse(context.Background(), tID)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("player API returned with status code %d when getting users from team", resp.StatusCode())
	}

	out := []structs.UserInfo{}
	if resp.JSON200 == nil {
		return out, nil
	}
	for _, u := range *resp.JSON200 {
		if u.Id == nil {
			continue
		}
		userID := u.Id.String()
		membershipID, err := findMembershipID(userID, viewID, teamID, m)
		if err != nil {
			return nil, err
		}
		role, err := getMembership(membershipID, m)
		if err != nil {
			return nil, err
		}
		out = append(out, structs.UserInfo{ID: userID, Role: role})
	}
	return out, nil
}

// roleString extracts a string role from an interface{} field (string or nil).
func roleString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// getRoleByName returns the id of the (user) role with the given name. Shared
// with player_api_team.go's user-role handling.
func getRoleByName(role string, m map[string]string) (string, error) {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return "", err
	}

	resp, err := client.GetRoleByNameWithResponse(context.Background(), role)
	if err != nil {
		return "", err
	}
	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("player API returned with status code %d looking for role %v", resp.StatusCode(), role)
	}
	if resp.JSON200 == nil || resp.JSON200.Id == nil {
		return "", fmt.Errorf("player API returned no id for role %v", role)
	}
	return resp.JSON200.Id.String(), nil
}
