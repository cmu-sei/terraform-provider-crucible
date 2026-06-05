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

// This file adapts the provider's application and application-instance operations
// onto the generated Player API client (internal/playerclient). Signatures
// unchanged. readApps is shared with player_api_view.go; getTeamAppInstances is
// used by player_api_team.go.

// CreateApps creates an application for each struct passed, under the given view.
func CreateApps(apps *[]*structs.AppInfo, m map[string]string, viewID string) error {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return err
	}

	vID, err := uuid.Parse(viewID)
	if err != nil {
		return err
	}

	for i, app := range *apps {
		body := playerclient.CreateApplicationCommand{
			ViewId:           vID,
			Name:             ifaceStrPtr(app.Name),
			Url:              ifaceStrPtr(app.URL),
			Icon:             ifaceStrPtr(app.Icon),
			Embeddable:       ifaceBoolPtr(app.Embeddable),
			LoadInBackground: ifaceBoolPtr(app.LoadInBackground),
		}
		if tid := ifaceStrPtr(app.AppTemplateID); tid != nil {
			id, err := uuid.Parse(*tid)
			if err != nil {
				return err
			}
			body.ApplicationTemplateId = &id
		}

		resp, err := client.CreateApplicationWithResponse(context.Background(), vID, body)
		if err != nil {
			return err
		}
		if resp.StatusCode() != http.StatusCreated {
			return fmt.Errorf("player API returned with status code %d when creating app. %d apps created before error", resp.StatusCode(), i)
		}
		if resp.JSON201 == nil || resp.JSON201.Id == nil {
			return fmt.Errorf("player API returned status 201 with no id when creating app")
		}
		app.ID = resp.JSON201.Id.String()
	}
	return nil
}

// UpdateApps updates the specified applications.
func UpdateApps(apps *[]*structs.AppInfo, m map[string]string) error {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return err
	}

	for i, app := range *apps {
		appID, err := uuid.Parse(app.ID)
		if err != nil {
			return err
		}
		viewID, err := uuid.Parse(app.ViewID)
		if err != nil {
			return err
		}

		body := playerclient.EditApplicationCommand{
			ViewId:           viewID,
			Name:             ifaceStrPtr(app.Name),
			Url:              ifaceStrPtr(app.URL),
			Icon:             ifaceStrPtr(app.Icon),
			Embeddable:       ifaceBoolPtr(app.Embeddable),
			LoadInBackground: ifaceBoolPtr(app.LoadInBackground),
		}
		if tid := ifaceStrPtr(app.AppTemplateID); tid != nil {
			id, err := uuid.Parse(*tid)
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
			return fmt.Errorf("player API returned with status code %d when updating app. %d apps updated before error", resp.StatusCode(), i)
		}
	}
	return nil
}

// DeleteApps deletes the specified applications.
func DeleteApps(ids *[]string, m map[string]string) error {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return err
	}

	for i, id := range *ids {
		appID, err := uuid.Parse(id)
		if err != nil {
			return err
		}
		resp, err := client.DeleteApplicationWithResponse(context.Background(), appID)
		if err != nil {
			return err
		}
		if resp.StatusCode() != http.StatusNoContent {
			return fmt.Errorf("player API returned with status code %d when deleting app. %d apps deleted before error", resp.StatusCode(), i)
		}
	}
	return nil
}

// UpdateAppInstance updates an application instance.
func UpdateAppInstance(inst structs.AppInstance, teamID string, m map[string]string) error {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return err
	}

	instID, err := uuid.Parse(inst.ID)
	if err != nil {
		return err
	}
	tID, err := uuid.Parse(teamID)
	if err != nil {
		return err
	}
	appID, err := uuid.Parse(inst.Parent)
	if err != nil {
		return err
	}
	order := float32(inst.DisplayOrder)

	resp, err := client.UpdateApplicationInstanceWithResponse(context.Background(), instID,
		playerclient.EditApplicationInstanceCommand{
			ApplicationId: appID,
			TeamId:        tID,
			DisplayOrder:  &order,
		})
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("player API returned with status code %d when updating app instance", resp.StatusCode())
	}
	return nil
}

// AddApplication adds an application to a team and returns the app-instance id.
func AddApplication(appID, teamID string, displayOrder float64, m map[string]string) (string, error) {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return "", err
	}

	tID, err := uuid.Parse(teamID)
	if err != nil {
		return "", err
	}
	aID, err := uuid.Parse(appID)
	if err != nil {
		return "", err
	}
	order := float32(displayOrder)

	resp, err := client.CreateApplicationInstanceWithResponse(context.Background(), tID,
		playerclient.CreateApplicationInstanceCommand{
			ApplicationId: aID,
			TeamId:        tID,
			DisplayOrder:  &order,
		})
	if err != nil {
		return "", err
	}
	if resp.StatusCode() != http.StatusCreated {
		return "", fmt.Errorf("Player API returned with status %d when adding application to team", resp.StatusCode())
	}
	if resp.JSON201 == nil || resp.JSON201.Id == nil {
		return "", fmt.Errorf("Player API returned status 201 with no id when adding application to team")
	}
	return resp.JSON201.Id.String(), nil
}

// DeleteAppInstances deletes the specified application instances.
func DeleteAppInstances(toDelete *[]string, m map[string]string) error {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return err
	}

	for i, id := range *toDelete {
		instID, err := uuid.Parse(id)
		if err != nil {
			return err
		}
		resp, err := client.DeleteApplicationInstanceWithResponse(context.Background(), instID)
		if err != nil {
			return err
		}
		if resp.StatusCode() != http.StatusNoContent {
			return fmt.Errorf("player API returned with status code %d when deleting app instance. %d instances deleted before error", resp.StatusCode(), i)
		}
	}
	return nil
}

// -------------------- Helper functions --------------------

// readApps reads the applications for a given view.
func readApps(id string, m map[string]string) (*[]structs.AppInfo, error) {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return nil, err
	}

	viewID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	resp, err := client.GetViewApplicationsWithResponse(context.Background(), viewID)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("Player API returned with status %d when retreiving application info", resp.StatusCode())
	}

	apps := new([]structs.AppInfo)
	if resp.JSON200 == nil {
		return apps, nil
	}
	for _, a := range *resp.JSON200 {
		info := structs.AppInfo{
			Name:             strOrNil(a.Name),
			URL:              strOrNil(a.Url),
			Icon:             strOrNil(a.Icon),
			Embeddable:       boolOrNil(a.Embeddable),
			LoadInBackground: boolOrNil(a.LoadInBackground),
			ViewID:           id,
		}
		if a.Id != nil {
			info.ID = a.Id.String()
		}
		if a.ApplicationTemplateId != nil {
			info.AppTemplateID = a.ApplicationTemplateId.String()
		}
		*apps = append(*apps, info)
	}
	return apps, nil
}

// getTeamAppInstances returns all application instances for a team.
func getTeamAppInstances(teamID string, m map[string]string) (*[]structs.AppInstance, error) {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return nil, err
	}

	tID, err := uuid.Parse(teamID)
	if err != nil {
		return nil, err
	}

	resp, err := client.GetTeamApplicationInstancesWithResponse(context.Background(), tID)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("Player API returned with status %d when retreiving application info", resp.StatusCode())
	}

	instances := new([]structs.AppInstance)
	if resp.JSON200 == nil {
		return instances, nil
	}
	for _, inst := range *resp.JSON200 {
		ai := structs.AppInstance{
			Name:         derefStr(inst.Name),
			DisplayOrder: float64(derefFloat32(inst.DisplayOrder)),
		}
		if inst.Id != nil {
			ai.ID = inst.Id.String()
		}
		if inst.ApplicationId != nil {
			ai.Parent = inst.ApplicationId.String()
		}
		*instances = append(*instances, ai)
	}
	return instances, nil
}
