// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/cmu-sei/terraform-provider-crucible/internal/structs"
	"github.com/cmu-sei/terraform-provider-crucible/internal/vmclient"
)

// This file adapts the provider's view-network operations onto the
// oapi-codegen-generated Player VM API client (internal/vmclient) — the same
// client vm_api.go uses, since view networks are part of the VM API. Exported
// function signatures are unchanged; only the bodies and model marshaling go
// through the typed client. structs.ViewNetworkInfo remains the provider-facing
// model; conversion to/from the generated types happens in the helpers below.

// CreateViewNetwork creates a view network via the VM API.
func CreateViewNetwork(network *structs.ViewNetworkInfo, m map[string]string) (*structs.ViewNetworkInfo, error) {
	client, err := vmclient.NewAuthed(m)
	if err != nil {
		return nil, err
	}

	viewID, err := uuid.Parse(network.ViewID)
	if err != nil {
		return nil, err
	}

	teamIDs, err := toUUIDPtr(network.TeamIds)
	if err != nil {
		return nil, err
	}

	form := vmclient.CreateViewNetworkForm{
		Name:               network.Name,
		NetworkId:          network.NetworkId,
		ProviderInstanceId: network.ProviderInstanceId,
		ProviderType:       vmclient.VmType(network.ProviderType),
		TeamIds:            teamIDs,
	}

	resp, err := client.CreateViewNetworkWithResponse(context.Background(), viewID, form)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusCreated {
		return nil, fmt.Errorf("VM API returned status %d when creating view network for view %s", resp.StatusCode(), network.ViewID)
	}
	if resp.JSON201 == nil {
		return nil, fmt.Errorf("VM API returned status 201 with an empty body when creating view network for view %s", network.ViewID)
	}
	return fromViewNetwork(resp.JSON201), nil
}

// GetViewNetwork reads a single view network.
func GetViewNetwork(viewID, id string, m map[string]string) (*structs.ViewNetworkInfo, error) {
	client, err := vmclient.NewAuthed(m)
	if err != nil {
		return nil, err
	}

	vID, err := uuid.Parse(viewID)
	if err != nil {
		return nil, err
	}
	nID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	resp, err := client.GetViewNetworkWithResponse(context.Background(), vID, nID)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("VM API returned status %d when reading view network %s for view %s", resp.StatusCode(), id, viewID)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("VM API returned status 200 with an empty body when reading view network %s", id)
	}
	return fromViewNetwork(resp.JSON200), nil
}

// UpdateViewNetwork updates a view network.
func UpdateViewNetwork(network *structs.ViewNetworkInfo, m map[string]string) error {
	client, err := vmclient.NewAuthed(m)
	if err != nil {
		return err
	}

	viewID, err := uuid.Parse(network.ViewID)
	if err != nil {
		return err
	}
	nID, err := uuid.Parse(network.ID)
	if err != nil {
		return err
	}

	teamIDs, err := toUUIDPtr(network.TeamIds)
	if err != nil {
		return err
	}

	form := vmclient.UpdateViewNetworkForm{
		Name:               network.Name,
		NetworkId:          network.NetworkId,
		ProviderInstanceId: network.ProviderInstanceId,
		ProviderType:       vmclient.VmType(network.ProviderType),
		TeamIds:            teamIDs,
	}

	resp, err := client.UpdateViewNetworkWithResponse(context.Background(), viewID, nID, form)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("VM API returned status %d when updating view network %s for view %s", resp.StatusCode(), network.ID, network.ViewID)
	}
	return nil
}

// DeleteViewNetwork removes a view network.
func DeleteViewNetwork(viewID, id string, m map[string]string) error {
	client, err := vmclient.NewAuthed(m)
	if err != nil {
		return err
	}

	vID, err := uuid.Parse(viewID)
	if err != nil {
		return err
	}
	nID, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	resp, err := client.DeleteViewNetworkWithResponse(context.Background(), vID, nID)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusNoContent {
		return fmt.Errorf("VM API returned status %d when deleting view network %s for view %s", resp.StatusCode(), id, viewID)
	}
	return nil
}

// ViewNetworkExists returns true if a view network with the given ID exists.
func ViewNetworkExists(viewID, id string, m map[string]string) (bool, error) {
	client, err := vmclient.NewAuthed(m)
	if err != nil {
		return false, err
	}

	vID, err := uuid.Parse(viewID)
	if err != nil {
		return false, err
	}
	nID, err := uuid.Parse(id)
	if err != nil {
		return false, err
	}

	resp, err := client.GetViewNetworkWithResponse(context.Background(), vID, nID)
	if err != nil {
		return false, err
	}
	return resp.StatusCode() != http.StatusNotFound, nil
}

// -------------------- structs.ViewNetworkInfo <-> generated model conversion --------------------

// fromViewNetwork maps a generated ViewNetwork response onto the provider model.
func fromViewNetwork(n *vmclient.ViewNetwork) *structs.ViewNetworkInfo {
	info := &structs.ViewNetworkInfo{
		ProviderInstanceId: derefStr(n.ProviderInstanceId),
		NetworkId:          derefStr(n.NetworkId),
		Name:               derefStr(n.Name),
		TeamIds:            []string{},
	}
	if n.Id != nil {
		info.ID = n.Id.String()
	}
	if n.ViewId != nil {
		info.ViewID = n.ViewId.String()
	}
	if n.ProviderType != nil {
		info.ProviderType = string(*n.ProviderType)
	}
	if n.TeamIds != nil {
		for _, t := range *n.TeamIds {
			info.TeamIds = append(info.TeamIds, t.String())
		}
	}
	return info
}

// toUUIDPtr parses a slice of string ids into a pointer-to-slice of generated
// UUIDs, returning nil for an empty input so the field is omitted.
func toUUIDPtr(ids []string) (*[]uuid.UUID, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	out, err := toUUIDs(ids)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
