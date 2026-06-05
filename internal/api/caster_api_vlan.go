// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/cmu-sei/terraform-provider-crucible/internal/casterclient"
	"github.com/cmu-sei/terraform-provider-crucible/internal/structs"
)

// This file adapts the provider's VLAN operations onto the oapi-codegen-generated
// Caster API client (internal/casterclient). Exported function signatures are
// unchanged from the previous hand-written net/http implementation; only the
// bodies and model marshaling now go through the typed client. structs.Vlan
// remains the provider-facing model — conversion happens in fromVlan below.

// CreateVlan acquires a VLAN via the Caster API.
//
// param command: A struct containing info on acquiring a vlan
//
// param m: A map containing configuration info for the provider
//
// Returns the acquired vlan and error on failure or nil on success
func CreateVlan(command *structs.VlanCreateCommand, m map[string]string) (*structs.Vlan, error) {
	client, err := casterclient.NewAuthed(m)
	if err != nil {
		return nil, err
	}

	body := casterclient.AcquireVlanCommand{}
	if command.ProjectId != "" {
		id, err := uuid.Parse(command.ProjectId)
		if err != nil {
			return nil, err
		}
		body.ProjectId = &id
	}
	if command.PartitionId != "" {
		id, err := uuid.Parse(command.PartitionId)
		if err != nil {
			return nil, err
		}
		body.PartitionId = &id
	}
	if command.Tag != "" {
		tag := command.Tag
		body.Tag = &tag
	}
	if command.VlanId.Valid {
		v := command.VlanId.Int32
		body.VlanId = &v
	}

	resp, err := client.AcquireVlanWithResponse(context.Background(), body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("Caster API returned with status code %d when creating vlan", resp.StatusCode())
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("Caster API returned status 200 with an empty body when acquiring vlan")
	}
	return fromVlan(resp.JSON200), nil
}

// ReadVlan reads the fields of a vlan by id.
//
// Param id: the id of the vlan to read
//
// param m: A map containing configuration info for the provider
//
// Returns error on failure or the vlan on success
func ReadVlan(id string, m map[string]string) (*structs.Vlan, error) {
	client, err := casterclient.NewAuthed(m)
	if err != nil {
		return nil, err
	}

	vlanID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	resp, err := client.GetVlanWithResponse(context.Background(), vlanID)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("Caster API returned with status code %d when reading vlan", resp.StatusCode())
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("Caster API returned status 200 with an empty body when reading vlan %s", id)
	}
	return fromVlan(resp.JSON200), nil
}

// VlanExists reports whether a VLAN record still exists in Caster. A released
// VLAN persists in the pool (with InUse=false), but the record can disappear
// entirely (404) if its containing pool or partition is deleted out of band, so
// the resource Read uses this to remove the resource from state rather than
// erroring. Mirrors the other *Exists helpers.
func VlanExists(id string, m map[string]string) (bool, error) {
	client, err := casterclient.NewAuthed(m)
	if err != nil {
		return false, err
	}

	vlanID, err := uuid.Parse(id)
	if err != nil {
		return false, err
	}

	resp, err := client.GetVlanWithResponse(context.Background(), vlanID)
	if err != nil {
		return false, err
	}
	return resp.StatusCode() != http.StatusNotFound, nil
}

// DeleteVlan releases a vlan back into the pool.
//
// Param id: The id of the vlan to release
//
// param m: A map containing configuration info for the provider
//
// Returns error on failure or nil on success
func DeleteVlan(id string, m map[string]string) error {
	client, err := casterclient.NewAuthed(m)
	if err != nil {
		return err
	}

	vlanID, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	resp, err := client.ReleaseVlanWithResponse(context.Background(), vlanID)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("Caster API returned with status code %d when deleting vlan", resp.StatusCode())
	}
	return nil
}

// FindVlanByNumber returns the resource id of the IN-USE (acquired) vlan with the
// given number in the given partition, or an empty string if none is acquired. A
// vlan record persists in the pool after release, so matching only in-use vlans
// keeps this to "find a leaked/acquired vlan" — its sole purpose (test cleanup of
// a vlan whose computed resource id wasn't captured). An empty partitionID
// resolves to Caster's default partition (matching how the vlan resource acquires
// from the default partition when none is configured).
func FindVlanByNumber(number int, partitionID string, m map[string]string) (string, error) {
	client, err := casterclient.NewAuthed(m)
	if err != nil {
		return "", err
	}

	if partitionID == "" {
		partitionID, err = defaultPartitionID(client)
		if err != nil {
			return "", err
		}
		if partitionID == "" {
			return "", nil // no default partition to search
		}
	}

	pID, err := uuid.Parse(partitionID)
	if err != nil {
		return "", err
	}

	resp, err := client.GetVlansByPartitionWithResponse(context.Background(), pID)
	if err != nil {
		return "", err
	}
	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("Caster API returned with status code %d when listing vlans in partition", resp.StatusCode())
	}
	if resp.JSON200 == nil {
		return "", nil
	}
	for _, v := range *resp.JSON200 {
		inUse := v.InUse != nil && *v.InUse
		if v.VlanId != nil && int(*v.VlanId) == number && inUse && v.Id != nil {
			return v.Id.String(), nil
		}
	}
	return "", nil
}

// defaultPartitionID returns the id of Caster's default partition, or "" if
// there isn't one.
func defaultPartitionID(client *casterclient.ClientWithResponses) (string, error) {
	resp, err := client.GetPartitionsWithResponse(context.Background())
	if err != nil {
		return "", err
	}
	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("Caster API returned with status code %d when listing partitions", resp.StatusCode())
	}
	if resp.JSON200 == nil {
		return "", nil
	}
	for _, p := range *resp.JSON200 {
		if p.IsDefault != nil && *p.IsDefault && p.Id != nil {
			return p.Id.String(), nil
		}
	}
	return "", nil
}

// -------------------- structs.Vlan <-> generated model conversion --------------------

// fromVlan maps a generated Vlan onto the provider Vlan struct.
func fromVlan(v *casterclient.Vlan) *structs.Vlan {
	out := &structs.Vlan{
		InUse:    derefBool(v.InUse, false),
		Reserved: derefBool(v.Reserved, false),
		Tag:      derefStr(v.Tag),
	}
	if v.Id != nil {
		out.Id = v.Id.String()
	}
	if v.PoolId != nil {
		out.PoolId = v.PoolId.String()
	}
	if v.PartitionId != nil {
		out.PartitionId = v.PartitionId.String()
	}
	if v.VlanId != nil {
		out.VlanId = int(*v.VlanId)
	}
	return out
}
