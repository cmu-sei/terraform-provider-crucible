// Copyright 2026 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/cmu-sei/terraform-provider-crucible/internal/playerclient"
)

type PlayerRole struct {
	ID             string
	Name           string
	AllPermissions bool
	Immutable      bool
	PermissionIDs  []string
}

func FindPlayerRoleByName(name string, caseInsensitive bool, m map[string]string) (*PlayerRole, error) {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return nil, err
	}
	resp, err := client.GetRolesWithResponse(context.Background())
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("player API returned status %d when listing roles", resp.StatusCode())
	}

	var matches []playerclient.Role
	if resp.JSON200 != nil {
		for _, role := range *resp.JSON200 {
			if role.Name != nil && namesMatch(*role.Name, name, caseInsensitive) {
				matches = append(matches, role)
			}
		}
	}
	if len(matches) != 1 {
		return nil, fmt.Errorf("role name %q resolved to %d roles; expected exactly one", name, len(matches))
	}
	role := matches[0]
	if role.Id == nil {
		return nil, fmt.Errorf("player API returned role %q without an id", name)
	}
	out := &PlayerRole{
		ID:             role.Id.String(),
		Name:           derefStr(role.Name),
		AllPermissions: derefBool(role.AllPermissions, false),
		Immutable:      derefBool(role.Immutable, false),
	}
	if role.Permissions != nil {
		for _, permission := range *role.Permissions {
			if permission.Id != nil {
				out.PermissionIDs = append(out.PermissionIDs, permission.Id.String())
			}
		}
	}
	return out, nil
}

func FindPlayerTeamRoleByName(name string, caseInsensitive bool, m map[string]string) (*PlayerRole, error) {
	client, err := playerclient.NewAuthed(m)
	if err != nil {
		return nil, err
	}
	resp, err := client.GetTeamRolesWithResponse(context.Background())
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("player API returned status %d when listing team roles", resp.StatusCode())
	}

	var matches []playerclient.TeamRole
	if resp.JSON200 != nil {
		for _, role := range *resp.JSON200 {
			if role.Name != nil && namesMatch(*role.Name, name, caseInsensitive) {
				matches = append(matches, role)
			}
		}
	}
	if len(matches) != 1 {
		return nil, fmt.Errorf("team role name %q resolved to %d team roles; expected exactly one", name, len(matches))
	}
	role := matches[0]
	if role.Id == nil {
		return nil, fmt.Errorf("player API returned team role %q without an id", name)
	}
	out := &PlayerRole{
		ID:             role.Id.String(),
		Name:           derefStr(role.Name),
		AllPermissions: derefBool(role.AllPermissions, false),
		Immutable:      derefBool(role.Immutable, false),
	}
	if role.Permissions != nil {
		for _, permission := range *role.Permissions {
			if permission.Id != nil {
				out.PermissionIDs = append(out.PermissionIDs, permission.Id.String())
			}
		}
	}
	return out, nil
}

func namesMatch(candidate, requested string, caseInsensitive bool) bool {
	if caseInsensitive {
		return strings.EqualFold(candidate, requested)
	}
	return candidate == requested
}
