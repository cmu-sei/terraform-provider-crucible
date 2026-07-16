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

// This file adapts the provider's view operations onto the oapi-codegen-generated
// Player API client (internal/playerclient). Exported function signatures are
// unchanged; only the bodies and model marshaling go through the typed client.
// structs.ViewInfo remains the provider-facing model. ReadView still delegates to
// readApps/readTeams (sibling files) for the nested application/team data.

// CreateView acquires a new view via the Player API and returns its id.
func CreateView(view *structs.ViewInfo, m map[string]string) (string, error) {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return "", err
	}

	body := playerclient.CreateViewCommand{
		Name:            strPtr(view.Name),
		CreateAdminTeam: boolPtr(view.CreateAdminTeam),
	}
	if view.Description != "" {
		body.Description = strPtr(view.Description)
	}
	status := view.Status
	if status == "" {
		status = "Active"
	}
	st := playerclient.ViewStatus(status)
	body.Status = &st

	resp, err := client.CreateViewWithResponse(context.Background(), body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode() != http.StatusCreated {
		return "", fmt.Errorf("player API returned with status code %d when creating view", resp.StatusCode())
	}
	if resp.JSON201 == nil || resp.JSON201.Id == nil {
		return "", fmt.Errorf("player API returned status 201 with no view id when creating view")
	}
	return resp.JSON201.Id.String(), nil
}

// ReadView reads a view's fields plus its applications and teams.
func ReadView(id string, m map[string]string) (*structs.ViewInfo, error) {
	view, err := ReadViewTopLevel(id, m)
	if err != nil {
		return nil, err
	}

	apps, err := readApps(id, m)
	if err != nil {
		return nil, err
	}
	teams, err := readTeams(id, m)
	if err != nil {
		return nil, err
	}

	view.Applications = *apps
	view.Teams = *teams
	return view, nil
}

// ReadViewTopLevel reads only fields owned by the view itself.
func ReadViewTopLevel(id string, m map[string]string) (*structs.ViewInfo, error) {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return nil, err
	}

	viewID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	resp, err := client.GetViewWithResponse(context.Background(), viewID)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("player API returned with status code %d when reading view", resp.StatusCode())
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("player API returned status 200 with an empty body when reading view %s", id)
	}

	view := &structs.ViewInfo{
		Name:        derefStr(resp.JSON200.Name),
		Description: derefStr(resp.JSON200.Description),
	}
	if resp.JSON200.DefaultTeamId != nil {
		view.DefaultTeamID = resp.JSON200.DefaultTeamId.String()
	}
	view.IsTemplate = derefBool(resp.JSON200.IsTemplate, false)
	if resp.JSON200.Status != nil {
		view.Status = string(*resp.JSON200.Status)
	}
	return view, nil
}

// UpdateView updates a view's top-level fields.
func UpdateView(view *structs.ViewInfo, m map[string]string, id string) error {
	current, err := ReadViewTopLevel(id, m)
	if err != nil {
		return err
	}
	view.DefaultTeamID = current.DefaultTeamID
	return updateView(view, m, id)
}

// ReconcileDefaultTeam assigns teamID as the view's default when desired is
// true. When desired is false, it only clears the setting if teamID currently
// owns it, so updating a non-default team cannot unset another team.
func ReconcileDefaultTeam(viewID, teamID string, desired bool, m map[string]string) error {
	view, err := ReadViewTopLevel(viewID, m)
	if err != nil {
		return err
	}
	if desired {
		if view.DefaultTeamID == teamID {
			return nil
		}
		view.DefaultTeamID = teamID
	} else {
		if view.DefaultTeamID != teamID {
			return nil
		}
		view.DefaultTeamID = ""
	}
	return updateView(view, m, viewID)
}

func updateView(view *structs.ViewInfo, m map[string]string, id string) error {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return err
	}

	viewID, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	body := playerclient.EditViewCommand{
		Name:        strPtr(view.Name),
		Description: strPtr(view.Description),
		IsTemplate:  boolPtr(view.IsTemplate),
	}
	if view.DefaultTeamID != "" {
		defaultTeamID, err := uuid.Parse(view.DefaultTeamID)
		if err != nil {
			return err
		}
		body.DefaultTeamId = &defaultTeamID
	}
	if view.Status != "" {
		st := playerclient.ViewStatus(view.Status)
		body.Status = &st
	}

	resp, err := client.UpdateViewWithResponse(context.Background(), viewID, body)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("player API returned with status code %d when updating view", resp.StatusCode())
	}
	return nil
}

// DeleteView deletes a view.
func DeleteView(id string, m map[string]string) error {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return err
	}

	viewID, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	resp, err := client.DeleteViewWithResponse(context.Background(), viewID)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusNoContent {
		return fmt.Errorf("player API returned with status code %d when deleting view", resp.StatusCode())
	}
	return nil
}

// FindViewByName returns the id of the first view whose name matches, or an
// empty string if none is found. Used by test cleanup to locate a leaked view
// (whose computed id wasn't captured) by its known fixed name.
func FindViewByName(name string, m map[string]string) (string, error) {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return "", err
	}

	resp, err := client.GetViewsWithResponse(context.Background())
	if err != nil {
		return "", err
	}
	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("player API returned with status code %d when listing views", resp.StatusCode())
	}
	if resp.JSON200 == nil {
		return "", nil
	}
	for _, v := range *resp.JSON200 {
		if v.Name != nil && *v.Name == name && v.Id != nil {
			return v.Id.String(), nil
		}
	}
	return "", nil
}

// ViewExists returns true if a view with the given id exists.
func ViewExists(id string, m map[string]string) (bool, error) {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return false, err
	}

	viewID, err := uuid.Parse(id)
	if err != nil {
		return false, err
	}

	resp, err := client.GetViewWithResponse(context.Background(), viewID)
	if err != nil {
		return false, err
	}
	return resp.StatusCode() != http.StatusNotFound, nil
}
