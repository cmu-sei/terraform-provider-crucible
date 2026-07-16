// Copyright 2026 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package api

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"github.com/cmu-sei/terraform-provider-crucible/internal/playerclient"
	"github.com/cmu-sei/terraform-provider-crucible/internal/structs"
	"github.com/google/uuid"
)

type PlayerApplication struct {
	ID                    string
	ViewID                string
	Name                  string
	URL                   *string
	Icon                  *string
	Embeddable            *bool
	LoadInBackground      *bool
	ApplicationTemplateID *string
}

type PlayerTeam struct {
	ID            string
	ViewID        string
	Name          string
	Role          string
	Default       bool
	Permissions   []string
	ScopedTeamIDs []string
}

type PlayerTeamUser struct {
	ID     string
	TeamID string
	UserID string
	Role   *string
}

type PlayerApplicationInstance struct {
	ID            string
	TeamID        string
	ApplicationID string
	DisplayOrder  float64
}

func CreatePlayerApplication(app *PlayerApplication, m map[string]string) error {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return err
	}
	viewID, err := uuid.Parse(app.ViewID)
	if err != nil {
		return err
	}
	body := playerclient.CreateApplicationCommand{
		ViewId:           viewID,
		Name:             &app.Name,
		Url:              app.URL,
		Icon:             app.Icon,
		Embeddable:       app.Embeddable,
		LoadInBackground: app.LoadInBackground,
	}
	if app.ApplicationTemplateID != nil {
		id, err := uuid.Parse(*app.ApplicationTemplateID)
		if err != nil {
			return err
		}
		body.ApplicationTemplateId = &id
	}
	resp, err := client.CreateApplicationWithResponse(context.Background(), viewID, body)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusCreated || resp.JSON201 == nil || resp.JSON201.Id == nil {
		return fmt.Errorf("player API returned status %d without a created application id", resp.StatusCode())
	}
	app.ID = resp.JSON201.Id.String()
	return nil
}

func ReadPlayerApplication(id string, m map[string]string) (*PlayerApplication, bool, error) {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return nil, false, err
	}
	appID, err := uuid.Parse(id)
	if err != nil {
		return nil, false, err
	}
	resp, err := client.GetApplicationWithResponse(context.Background(), appID)
	if err != nil {
		return nil, false, err
	}
	if resp.StatusCode() == http.StatusNotFound || (resp.StatusCode() == http.StatusOK && resp.JSON200 == nil) {
		return nil, false, nil
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, false, fmt.Errorf("player API returned status %d when reading application %s", resp.StatusCode(), id)
	}
	a := resp.JSON200
	out := &PlayerApplication{ID: id, ViewID: a.ViewId.String(), Name: derefStr(a.Name), URL: a.Url, Icon: a.Icon, Embeddable: a.Embeddable, LoadInBackground: a.LoadInBackground}
	if a.Id != nil {
		out.ID = a.Id.String()
	}
	if a.ApplicationTemplateId != nil {
		v := a.ApplicationTemplateId.String()
		out.ApplicationTemplateID = &v
	}
	return out, true, nil
}

func UpdatePlayerApplication(app *PlayerApplication, m map[string]string) error {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return err
	}
	appID, err := uuid.Parse(app.ID)
	if err != nil {
		return err
	}
	viewID, err := uuid.Parse(app.ViewID)
	if err != nil {
		return err
	}
	body := playerclient.EditApplicationCommand{ViewId: viewID, Name: &app.Name, Url: app.URL, Icon: app.Icon, Embeddable: app.Embeddable, LoadInBackground: app.LoadInBackground}
	if app.ApplicationTemplateID != nil {
		id, err := uuid.Parse(*app.ApplicationTemplateID)
		if err != nil {
			return err
		}
		body.ApplicationTemplateId = &id
	}
	resp, err := client.UpdateApplicationWithResponse(context.Background(), appID, body)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("player API returned status %d when updating application %s", resp.StatusCode(), app.ID)
	}
	return nil
}

func DeletePlayerApplication(id string, m map[string]string) error {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return err
	}
	appID, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	resp, err := client.DeleteApplicationWithResponse(context.Background(), appID)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusNoContent && resp.StatusCode() != http.StatusNotFound {
		return fmt.Errorf("player API returned status %d when deleting application %s", resp.StatusCode(), id)
	}
	return nil
}

func CreatePlayerTeam(team *PlayerTeam, m map[string]string) error {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return err
	}
	viewID, err := uuid.Parse(team.ViewID)
	if err != nil {
		return err
	}
	roleID, err := getTeamRoleByName(team.Role, m)
	if err != nil {
		return err
	}
	rid, err := uuid.Parse(roleID)
	if err != nil {
		return err
	}
	resp, err := client.CreateTeamWithResponse(context.Background(), viewID, playerclient.CreateTeamCommand{Name: &team.Name, RoleId: &rid})
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusCreated || resp.JSON201 == nil || resp.JSON201.Id == nil {
		return fmt.Errorf("player API returned status %d without a created team id", resp.StatusCode())
	}
	team.ID = resp.JSON201.Id.String()
	if err := ReconcilePlayerTeamRelationships(team.ID, team.Permissions, team.ScopedTeamIDs, m); err != nil {
		return err
	}
	return ReconcileDefaultTeam(team.ViewID, team.ID, team.Default, m)
}

func ReadPlayerTeam(id string, m map[string]string) (*PlayerTeam, bool, error) {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return nil, false, err
	}
	teamID, err := uuid.Parse(id)
	if err != nil {
		return nil, false, err
	}
	resp, err := client.GetTeamWithResponse(context.Background(), teamID)
	if err != nil {
		return nil, false, err
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, false, nil
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, false, fmt.Errorf("player API returned status %d when reading team %s", resp.StatusCode(), id)
	}
	t := resp.JSON200
	if t.ViewId == nil {
		return nil, false, fmt.Errorf("player API returned team %s without a view id", id)
	}
	out := &PlayerTeam{ID: id, ViewID: t.ViewId.String(), Name: derefStr(t.Name), Role: derefStr(t.RoleName)}
	if t.Id != nil {
		out.ID = t.Id.String()
	}
	if t.Permissions != nil {
		for _, permission := range *t.Permissions {
			if permission.Id != nil {
				out.Permissions = append(out.Permissions, permission.Id.String())
			}
		}
	}
	if t.ScopedTeamIds != nil {
		for _, scopedID := range *t.ScopedTeamIds {
			out.ScopedTeamIDs = append(out.ScopedTeamIDs, scopedID.String())
		}
	}
	sort.Strings(out.Permissions)
	sort.Strings(out.ScopedTeamIDs)
	view, err := ReadViewTopLevel(out.ViewID, m)
	if err != nil {
		return nil, false, err
	}
	out.Default = view.DefaultTeamID == out.ID
	return out, true, nil
}

func UpdatePlayerTeam(team *PlayerTeam, m map[string]string) error {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return err
	}
	teamID, err := uuid.Parse(team.ID)
	if err != nil {
		return err
	}
	roleID, err := getTeamRoleByName(team.Role, m)
	if err != nil {
		return err
	}
	rid, err := uuid.Parse(roleID)
	if err != nil {
		return err
	}
	resp, err := client.UpdateTeamWithResponse(context.Background(), teamID, playerclient.EditTeamCommand{Name: &team.Name, RoleId: &rid})
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("player API returned status %d when updating team %s", resp.StatusCode(), team.ID)
	}
	if err := ReconcilePlayerTeamRelationships(team.ID, team.Permissions, team.ScopedTeamIDs, m); err != nil {
		return err
	}
	return ReconcileDefaultTeam(team.ViewID, team.ID, team.Default, m)
}

func ReconcilePlayerTeamRelationships(teamID string, desiredPermissions, desiredScopes []string, m map[string]string) error {
	current, exists, err := ReadPlayerTeam(teamID, m)
	if err != nil || !exists {
		if err != nil {
			return err
		}
		return fmt.Errorf("team %s disappeared while reconciling relationships", teamID)
	}
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return err
	}
	tid, _ := uuid.Parse(teamID)
	currentPermissions := stringSet(current.Permissions)
	wantedPermissions := stringSet(desiredPermissions)
	for _, id := range setDifference(wantedPermissions, currentPermissions) {
		if err := addTeamPermission(teamID, id, m); err != nil {
			return err
		}
	}
	for _, id := range setDifference(currentPermissions, wantedPermissions) {
		if err := removeTeamPermission(teamID, id, m); err != nil {
			return err
		}
	}
	currentScopes := stringSet(current.ScopedTeamIDs)
	wantedScopes := stringSet(desiredScopes)
	for _, id := range setDifference(wantedScopes, currentScopes) {
		target, err := uuid.Parse(id)
		if err != nil {
			return err
		}
		resp, err := client.AddTeamPermissionScopeWithResponse(context.Background(), tid, target)
		if err != nil {
			return err
		}
		if resp.StatusCode() != http.StatusOK {
			return fmt.Errorf("player API returned status %d when adding team scope", resp.StatusCode())
		}
	}
	for _, id := range setDifference(currentScopes, wantedScopes) {
		target, err := uuid.Parse(id)
		if err != nil {
			return err
		}
		resp, err := client.RemoveTeamPermissionScopeWithResponse(context.Background(), tid, target)
		if err != nil {
			return err
		}
		if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNotFound {
			return fmt.Errorf("player API returned status %d when removing team scope", resp.StatusCode())
		}
	}
	return nil
}

func DeletePlayerTeam(id string, m map[string]string) error {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return err
	}
	teamID, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	resp, err := client.DeleteTeamWithResponse(context.Background(), teamID)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusNoContent && resp.StatusCode() != http.StatusNotFound {
		return fmt.Errorf("player API returned status %d when deleting team %s", resp.StatusCode(), id)
	}
	return nil
}

func CreatePlayerTeamUser(membership *PlayerTeamUser, m map[string]string) error {
	if err := addUser(membership.UserID, membership.TeamID, m); err != nil {
		return err
	}
	id, err := findMembershipIDByTeam(membership.UserID, membership.TeamID, m)
	if err != nil {
		return err
	}
	membership.ID = id
	if membership.Role != nil {
		return UpdatePlayerTeamUser(membership, m)
	}
	return nil
}

func ReadPlayerTeamUser(id string, m map[string]string) (*PlayerTeamUser, bool, error) {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return nil, false, err
	}
	membershipID, err := uuid.Parse(id)
	if err != nil {
		return nil, false, err
	}
	resp, err := client.GetTeamMembershipWithResponse(context.Background(), membershipID)
	if err != nil {
		return nil, false, err
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, false, nil
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, false, fmt.Errorf("player API returned status %d when reading team membership %s", resp.StatusCode(), id)
	}
	value := resp.JSON200
	out := &PlayerTeamUser{ID: id, TeamID: value.TeamId.String(), UserID: value.UserId.String()}
	if value.Id != nil {
		out.ID = value.Id.String()
	}
	if value.RoleName != nil {
		role := *value.RoleName
		out.Role = &role
	}
	return out, true, nil
}

func UpdatePlayerTeamUser(membership *PlayerTeamUser, m map[string]string) error {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return err
	}
	membershipID, err := uuid.Parse(membership.ID)
	if err != nil {
		return err
	}
	var roleID *uuid.UUID
	if membership.Role != nil {
		id, err := getTeamRoleByName(*membership.Role, m)
		if err != nil {
			return err
		}
		parsed, err := uuid.Parse(id)
		if err != nil {
			return err
		}
		roleID = &parsed
	}
	resp, err := client.UpdateTeamMembershipWithResponse(context.Background(), membershipID, playerclient.EditTeamMembershipCommand{RoleId: roleID})
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("player API returned status %d when updating team membership %s", resp.StatusCode(), membership.ID)
	}
	return nil
}

func DeletePlayerTeamUser(teamID, userID string, m map[string]string) error {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return err
	}
	tid, err := uuid.Parse(teamID)
	if err != nil {
		return err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	resp, err := client.RemoveUserFromTeamWithResponse(context.Background(), tid, uid)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNotFound {
		return fmt.Errorf("player API returned status %d when deleting team membership", resp.StatusCode())
	}
	return nil
}

func CreatePlayerApplicationInstance(instance *PlayerApplicationInstance, m map[string]string) error {
	id, err := AddApplication(instance.ApplicationID, instance.TeamID, instance.DisplayOrder, m)
	if err != nil {
		return err
	}
	instance.ID = id
	return nil
}

func ReadPlayerApplicationInstance(id, teamID string, m map[string]string) (*PlayerApplicationInstance, bool, error) {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return nil, false, err
	}
	instanceID, err := uuid.Parse(id)
	if err != nil {
		return nil, false, err
	}
	resp, err := client.GetApplicationInstanceWithResponse(context.Background(), instanceID)
	if err != nil {
		return nil, false, err
	}
	if resp.StatusCode() == http.StatusNotFound || (resp.StatusCode() == http.StatusOK && resp.JSON200 == nil) {
		return nil, false, nil
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, false, fmt.Errorf("player API returned status %d when reading application instance %s", resp.StatusCode(), id)
	}
	tid, err := uuid.Parse(teamID)
	if err != nil {
		return nil, false, err
	}
	teamResp, err := client.GetTeamApplicationInstancesWithResponse(context.Background(), tid)
	if err != nil {
		return nil, false, err
	}
	if teamResp.StatusCode() == http.StatusNotFound {
		return nil, false, nil
	}
	if teamResp.StatusCode() != http.StatusOK || teamResp.JSON200 == nil {
		return nil, false, fmt.Errorf("player API returned status %d when verifying application instance team", teamResp.StatusCode())
	}
	found := false
	for _, candidate := range *teamResp.JSON200 {
		if candidate.Id != nil && candidate.Id.String() == id {
			found = true
			break
		}
	}
	if !found {
		return nil, false, nil
	}
	value := resp.JSON200
	out := &PlayerApplicationInstance{ID: id, TeamID: teamID, DisplayOrder: float64(derefFloat32(value.DisplayOrder))}
	if value.Id != nil {
		out.ID = value.Id.String()
	}
	if value.ApplicationId != nil {
		out.ApplicationID = value.ApplicationId.String()
	}
	return out, true, nil
}

func UpdatePlayerApplicationInstance(instance *PlayerApplicationInstance, m map[string]string) error {
	return UpdateAppInstance(structs.AppInstance{
		ID:           instance.ID,
		Parent:       instance.ApplicationID,
		DisplayOrder: instance.DisplayOrder,
	}, instance.TeamID, m)
}

func DeletePlayerApplicationInstance(id string, m map[string]string) error {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return err
	}
	instanceID, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	resp, err := client.DeleteApplicationInstanceWithResponse(context.Background(), instanceID)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusNoContent && resp.StatusCode() != http.StatusNotFound {
		return fmt.Errorf("player API returned status %d when deleting application instance %s", resp.StatusCode(), id)
	}
	return nil
}

func stringSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}

func setDifference(left, right map[string]struct{}) []string {
	out := make([]string, 0)
	for value := range left {
		if _, exists := right[value]; !exists {
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}
