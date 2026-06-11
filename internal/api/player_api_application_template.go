// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/cmu-sei/terraform-provider-crucible/internal/playerclient"
	"github.com/cmu-sei/terraform-provider-crucible/internal/structs"
)

// This file adapts the provider's application-template operations onto the
// generated Player API client (internal/playerclient). Signatures unchanged;
// structs.AppTemplate stays the provider-facing model.

// CreateAppTemplate creates an application template and returns its id.
func CreateAppTemplate(template *structs.AppTemplate, m map[string]string) (string, error) {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return "", err
	}

	body := playerclient.CreateApplicationTemplateCommand{
		Name:             strPtr(template.Name),
		Embeddable:       boolPtr(template.Embeddable),
		LoadInBackground: boolPtr(template.LoadInBackground),
	}
	if template.URL != "" {
		body.Url = strPtr(template.URL)
	}
	if template.Icon != "" {
		body.Icon = strPtr(template.Icon)
	}

	resp, err := client.CreateApplicationTemplateWithResponse(context.Background(), body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode() != http.StatusCreated {
		return "", fmt.Errorf("player API returned with status code %d when creating template", resp.StatusCode())
	}
	if resp.JSON201 == nil || resp.JSON201.Id == nil {
		return "", fmt.Errorf("player API returned status 201 with no id when creating template")
	}
	return resp.JSON201.Id.String(), nil
}

// AppTemplateRead returns the remote state of an application template. Note: the
// Player API returns 200 with an empty body for a missing template (not 404), so
// a deleted template surfaces here as an AppTemplate with an empty Name.
func AppTemplateRead(id string, m map[string]string) (*structs.AppTemplate, error) {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return nil, err
	}

	tmplID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	resp, err := client.GetApplicationTemplateWithResponse(context.Background(), tmplID)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("player API returned with status code %d when reading template", resp.StatusCode())
	}
	if resp.JSON200 == nil {
		// Empty body for a missing template — represent as a zero-valued struct.
		return &structs.AppTemplate{}, nil
	}
	return fromAppTemplate(resp.JSON200), nil
}

// AppTemplateUpdate updates an application template.
func AppTemplateUpdate(id string, template *structs.AppTemplate, m map[string]string) error {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return err
	}

	tmplID, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	body := playerclient.EditApplicationTemplateCommand{
		Name:             strPtr(template.Name),
		Url:              strPtr(template.URL),
		Icon:             strPtr(template.Icon),
		Embeddable:       boolPtr(template.Embeddable),
		LoadInBackground: boolPtr(template.LoadInBackground),
	}

	resp, err := client.UpdateApplicationTemplateWithResponse(context.Background(), tmplID, body)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("player API returned with status code %d when updating template", resp.StatusCode())
	}
	return nil
}

// DeleteAppTemplate deletes an application template.
func DeleteAppTemplate(id string, m map[string]string) error {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return err
	}

	tmplID, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	resp, err := client.DeleteApplicationTemplateWithResponse(context.Background(), tmplID)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusNoContent {
		return fmt.Errorf("player API returned with status code %d when deleting template", resp.StatusCode())
	}
	return nil
}

// AppTemplateExists returns whether a template exists.
//
// The current Player API signals a missing template as 200 with an empty body
// (not 404), so checking the status code alone would always report "exists".
// Treat BOTH a 404 and a 200-with-empty-body as gone, so this keeps working if
// the API later switches to returning 404.
func AppTemplateExists(id string, m map[string]string) (bool, error) {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return false, err
	}

	tmplID, err := uuid.Parse(id)
	if err != nil {
		return false, err
	}

	resp, err := client.GetApplicationTemplateWithResponse(context.Background(), tmplID)
	if err != nil {
		return false, err
	}
	if resp.StatusCode() == http.StatusNotFound {
		return false, nil // future-proof: API may switch to 404
	}
	if resp.StatusCode() != http.StatusOK {
		return false, fmt.Errorf("player API returned status code %d when checking template existence", resp.StatusCode())
	}
	// The current API returns 200 with an empty body for a missing template.
	return resp.JSON200 != nil, nil
}

// AppTemplateFindByName returns the id and provider-facing model of the
// application template whose name matches the given name. The Player API has no
// get-by-name endpoint, so this lists all templates and filters client-side.
// Player does not enforce unique template names, so this errors if more than one
// matches; it also errors if none match. When caseInsensitive is true, matching
// uses strings.EqualFold.
//
// The id is returned separately because structs.AppTemplate does not carry it.
func AppTemplateFindByName(name string, caseInsensitive bool, m map[string]string) (string, *structs.AppTemplate, error) {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return "", nil, err
	}

	resp, err := client.GetApplicationTemplatesWithResponse(context.Background())
	if err != nil {
		return "", nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return "", nil, fmt.Errorf("player API returned with status code %d when listing templates", resp.StatusCode())
	}
	if resp.JSON200 == nil {
		return "", nil, fmt.Errorf("player API returned status 200 with no body when listing templates")
	}

	match, err := matchAppTemplateByName(*resp.JSON200, name, caseInsensitive)
	if err != nil {
		return "", nil, err
	}
	if match.Id == nil {
		return "", nil, fmt.Errorf("player API returned a template with no id")
	}
	return match.Id.String(), fromAppTemplate(match), nil
}

// matchAppTemplateByName returns the single template whose name matches; it
// errors when zero or more than one match. Kept separate from the HTTP call so
// the 0/1/many and case-insensitive logic can be unit tested without a server.
func matchAppTemplateByName(templates []playerclient.ApplicationTemplate, name string, caseInsensitive bool) (*playerclient.ApplicationTemplate, error) {
	var matches []playerclient.ApplicationTemplate
	for _, t := range templates {
		tName := derefStr(t.Name)
		if (caseInsensitive && strings.EqualFold(tName, name)) || tName == name {
			matches = append(matches, t)
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no application template found with name %q", name)
	case 1:
		return &matches[0], nil
	default:
		return nil, fmt.Errorf("found %d application templates with name %q; names are not unique, look up by id instead", len(matches), name)
	}
}

// fromAppTemplate maps a generated ApplicationTemplate onto the provider struct.
func fromAppTemplate(t *playerclient.ApplicationTemplate) *structs.AppTemplate {
	return &structs.AppTemplate{
		Name:             derefStr(t.Name),
		URL:              derefStr(t.Url),
		Icon:             derefStr(t.Icon),
		Embeddable:       derefBool(t.Embeddable, false),
		LoadInBackground: derefBool(t.LoadInBackground, false),
	}
}
