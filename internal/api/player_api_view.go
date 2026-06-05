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
		return "", fmt.Errorf("Player API returned with status code %d when creating view", resp.StatusCode())
	}
	if resp.JSON201 == nil || resp.JSON201.Id == nil {
		return "", fmt.Errorf("Player API returned status 201 with no view id when creating view")
	}
	return resp.JSON201.Id.String(), nil
}

// ReadView reads a view's fields plus its applications and teams.
func ReadView(id string, m map[string]string) (*structs.ViewInfo, error) {
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
		return nil, fmt.Errorf("Player API returned with status code %d when reading view", resp.StatusCode())
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("Player API returned status 200 with an empty body when reading view %s", id)
	}

	view := &structs.ViewInfo{
		Name:        derefStr(resp.JSON200.Name),
		Description: derefStr(resp.JSON200.Description),
	}
	if resp.JSON200.Status != nil {
		view.Status = string(*resp.JSON200.Status)
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

// UpdateView updates a view's top-level fields.
func UpdateView(view *structs.ViewInfo, m map[string]string, id string) error {
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
		return fmt.Errorf("Player API returned with status code %d when updating view", resp.StatusCode())
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
		return fmt.Errorf("Player API returned with status code %d when deleting view", resp.StatusCode())
	}
	return nil
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
