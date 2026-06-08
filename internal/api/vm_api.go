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

// This file adapts the provider's VM operations onto the oapi-codegen–generated
// Player VM API client (internal/vmclient). The exported function signatures are
// unchanged from the previous hand-written net/http implementation so the VM
// resource (player_virtual_machine_server.go) is unaffected; only the bodies and
// the model marshaling now go through the typed client. structs.VMInfo remains
// the provider-facing model — conversion to/from the generated types happens in
// the helpers at the bottom of this file.

// -------------------- API Wrappers --------------------

// CreateVM creates a new VM via the VM API.
//
// param requestBody: The struct representing the VM to be created
//
// param m: map containing provider config info.
func CreateVM(requestBody *structs.VMInfo, m map[string]string) error {
	client, err := vmclient.NewAuthed(m)
	if err != nil {
		return err
	}

	form, err := toCreateForm(requestBody)
	if err != nil {
		return err
	}

	resp, err := client.CreateVmWithResponse(context.Background(), form)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusCreated {
		return fmt.Errorf("request returned with status code %d", resp.StatusCode())
	}
	return nil
}

// GetVMInfo retrieves a VM's info by id.
//
// param id: the id of the VM to look up
//
// Returns a struct containing the VM's info, and a possible error.
func GetVMInfo(id string, m map[string]string) (*structs.VMInfo, error) {
	client, err := vmclient.NewAuthed(m)
	if err != nil {
		return nil, err
	}

	vmID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	resp, err := client.GetVmWithResponse(context.Background(), vmID)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("request returned with status code %d", resp.StatusCode())
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("VM API returned status 200 with an empty body")
	}
	return fromVM(resp.JSON200), nil
}

// UpdateVM updates a VM via the VM API.
//
// requestBody: struct representing the data to update the VM with
//
// id: the ID of the VM to be updated
//
// Returns some error on failure and nil on success.
func UpdateVM(requestBody *structs.VMInfo, id string, m map[string]string) error {
	client, err := vmclient.NewAuthed(m)
	if err != nil {
		return err
	}

	vmID, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	resp, err := client.UpdateVmWithResponse(context.Background(), vmID, toUpdateForm(requestBody))
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("request returned with status code %d", resp.StatusCode())
	}
	return nil
}

// DeleteVM deletes a given VM.
//
// id: the id of the VM to delete
//
// returns error on failure or nil on success.
func DeleteVM(id string, m map[string]string) error {
	client, err := vmclient.NewAuthed(m)
	if err != nil {
		return err
	}

	vmID, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	resp, err := client.DeleteVmWithResponse(context.Background(), vmID)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusNoContent {
		return fmt.Errorf("request returned with status code %d", resp.StatusCode())
	}
	return nil
}

// VMExists returns true if a VM with the given id exists.
func VMExists(id string, m map[string]string) (bool, error) {
	client, err := vmclient.NewAuthed(m)
	if err != nil {
		return false, err
	}

	vmID, err := uuid.Parse(id)
	if err != nil {
		return false, err
	}

	resp, err := client.GetVmWithResponse(context.Background(), vmID)
	if err != nil {
		return false, err
	}
	return resp.StatusCode() != http.StatusNotFound, nil
}

// RemoveVMFromTeams removes the specified VM from the specified teams.
//
// param teams: The IDs of the teams to remove the VM from
//
// param vm: The ID of the VM
//
// param m: A map containing config info for the provider
//
// Returns nil on success or some error on failure.
func RemoveVMFromTeams(teams *[]string, vm string, m map[string]string) error {
	client, err := vmclient.NewAuthed(m)
	if err != nil {
		return err
	}

	vmID, err := uuid.Parse(vm)
	if err != nil {
		return err
	}

	for _, team := range *teams {
		teamID, err := uuid.Parse(team)
		if err != nil {
			return err
		}

		resp, err := client.RemoveVmFromTeamWithResponse(context.Background(), teamID, vmID)
		if err != nil {
			return err
		}
		if resp.StatusCode() != http.StatusNoContent {
			return fmt.Errorf("api returned with status code %d when removing VM %s from team %s", resp.StatusCode(), vm, team)
		}
	}

	return nil
}

// AddVMToTeams adds the specified VM to the specified teams.
//
// param teams: The IDs of the teams to add this VM to
//
// param vm: The ID of the VM
//
// param m: A map containing config info for the provider
//
// Returns nil on success or some error on failure.
func AddVMToTeams(teams *[]string, vm string, m map[string]string) error {
	client, err := vmclient.NewAuthed(m)
	if err != nil {
		return err
	}

	vmID, err := uuid.Parse(vm)
	if err != nil {
		return err
	}

	for _, team := range *teams {
		teamID, err := uuid.Parse(team)
		if err != nil {
			return err
		}

		resp, err := client.AddVmToTeamWithResponse(context.Background(), teamID, vmID)
		if err != nil {
			return err
		}
		if resp.StatusCode() != http.StatusOK {
			return fmt.Errorf("api returned with status code %d when adding VM %s to team %s", resp.StatusCode(), vm, team)
		}
	}

	return nil
}

// -------------------- structs.VMInfo <-> generated model conversion --------------------

// toCreateForm maps a provider VMInfo onto the generated VmCreateForm. Url and
// Embeddable are always sent (matching the previous client, which marshaled the
// whole struct) so an empty Url lets the API compute the default. UserId is sent
// only when set, preserving the nil-when-empty behavior.
func toCreateForm(vm *structs.VMInfo) (vmclient.VmCreateForm, error) {
	teamIDs, err := toUUIDs(vm.TeamIDs)
	if err != nil {
		return vmclient.VmCreateForm{}, err
	}

	url := vm.URL
	embeddable := vm.Embeddable
	form := vmclient.VmCreateForm{
		Name:                  vm.Name,
		TeamIds:               teamIDs,
		Url:                   &url,
		Embeddable:            &embeddable,
		ConsoleConnectionInfo: toConnInfo(vm.Connection),
		ProxmoxVmInfo:         toProxmoxInfo(vm.Proxmox),
	}

	if vm.ID != "" {
		id, err := uuid.Parse(vm.ID)
		if err != nil {
			return vmclient.VmCreateForm{}, err
		}
		form.Id = &id
	}

	if uid := userIDString(vm.UserID); uid != "" {
		u, err := uuid.Parse(uid)
		if err != nil {
			return vmclient.VmCreateForm{}, err
		}
		form.UserId = &u
	}

	return form, nil
}

// toUpdateForm maps a provider VMInfo onto the generated VmUpdateForm. The update
// API ignores id/team membership (handled separately via Add/RemoveVMToTeams), so
// those fields are absent from the form.
func toUpdateForm(vm *structs.VMInfo) vmclient.VmUpdateForm {
	url := vm.URL
	embeddable := vm.Embeddable
	form := vmclient.VmUpdateForm{
		Name:                  vm.Name,
		Url:                   &url,
		Embeddable:            &embeddable,
		ConsoleConnectionInfo: toConnInfo(vm.Connection),
		ProxmoxVmInfo:         toProxmoxInfo(vm.Proxmox),
	}

	if uid := userIDString(vm.UserID); uid != "" {
		if u, err := uuid.Parse(uid); err == nil {
			form.UserId = &u
		}
	}

	return form
}

// fromVM maps a generated Vm response onto the provider VMInfo. The defaultUrl
// (false) and embeddable (true) fallbacks preserve the previous client's
// behavior for older API versions that omit those fields.
func fromVM(vm *vmclient.Vm) *structs.VMInfo {
	info := &structs.VMInfo{
		URL:        derefStr(vm.Url),
		Name:       derefStr(vm.Name),
		DefaultURL: derefBool(vm.DefaultUrl, false),
		Embeddable: derefBool(vm.Embeddable, true),
		Connection: fromConnInfo(vm.ConsoleConnectionInfo),
		Proxmox:    fromProxmoxInfo(vm.ProxmoxVmInfo),
	}

	if vm.Id != nil {
		info.ID = vm.Id.String()
	}
	if vm.TeamIds != nil {
		for _, t := range *vm.TeamIds {
			info.TeamIDs = append(info.TeamIDs, t.String())
		}
	}
	// UserID stays an interface{}: a string when set, nil otherwise (the resource
	// type-asserts it to string).
	if vm.UserId != nil {
		info.UserID = vm.UserId.String()
	}

	return info
}

func toConnInfo(c *structs.ConsoleConnection) *vmclient.ConsoleConnectionInfo {
	if c == nil {
		return nil
	}
	return &vmclient.ConsoleConnectionInfo{
		Hostname: &c.Hostname,
		Port:     &c.Port,
		Protocol: &c.Protocol,
		Username: &c.Username,
		Password: &c.Password,
	}
}

func fromConnInfo(c *vmclient.ConsoleConnectionInfo) *structs.ConsoleConnection {
	if c == nil {
		return nil
	}
	return &structs.ConsoleConnection{
		Hostname: derefStr(c.Hostname),
		Port:     derefStr(c.Port),
		Protocol: derefStr(c.Protocol),
		Username: derefStr(c.Username),
		Password: derefStr(c.Password),
	}
}

func toProxmoxInfo(p *structs.ProxmoxInfo) *vmclient.ProxmoxVmInfo {
	if p == nil {
		return nil
	}
	id := int32(p.Id)
	node := p.Node
	t := vmclient.ProxmoxVmType(p.Type)
	return &vmclient.ProxmoxVmInfo{
		Id:   &id,
		Node: &node,
		Type: &t,
	}
}

func fromProxmoxInfo(p *vmclient.ProxmoxVmInfo) *structs.ProxmoxInfo {
	if p == nil {
		return nil
	}
	info := &structs.ProxmoxInfo{
		Node: derefStr(p.Node),
	}
	if p.Id != nil {
		info.Id = int(*p.Id)
	}
	if p.Type != nil {
		info.Type = string(*p.Type)
	}
	return info
}

// userIDString extracts a string user id from the VMInfo.UserID interface, which
// is either a string or nil.
func userIDString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
