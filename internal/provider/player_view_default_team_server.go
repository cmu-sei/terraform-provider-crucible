// Copyright 2026 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider

import (
	"context"
	"fmt"
	"sync"

	"github.com/cmu-sei/terraform-provider-crucible/internal/api"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &playerViewDefaultTeamResource{}
	_ resource.ResourceWithConfigure   = &playerViewDefaultTeamResource{}
	_ resource.ResourceWithImportState = &playerViewDefaultTeamResource{}

	defaultTeamCreateMu sync.Mutex
)

type playerViewDefaultTeamResource struct{ playerConfiguredResource }

type playerViewDefaultTeamModel struct {
	ID     types.String `tfsdk:"id"`
	ViewID types.String `tfsdk:"view_id"`
	TeamID types.String `tfsdk:"team_id"`
}

func NewPlayerViewDefaultTeamResource() resource.Resource {
	return &playerViewDefaultTeamResource{}
}

func (r *playerViewDefaultTeamResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_player_view_default_team"
}

func (r *playerViewDefaultTeamResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the default team for one Player view.",
		Attributes: map[string]schema.Attribute{
			"id":      computedIDAttribute(),
			"view_id": requiredUUIDAttribute("UUID of the Player view."),
			"team_id": schema.StringAttribute{
				Required:    true,
				Description: "UUID of the default team.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex, "must be a valid UUID"),
				},
			},
		},
	}
}

func (r *playerViewDefaultTeamResource) validateTeam(viewID, teamID string) error {
	team, exists, err := api.ReadPlayerTeam(teamID, r.cfg)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("Player team %s does not exist", teamID)
	}
	if team.ViewID != viewID {
		return fmt.Errorf("Player team %s belongs to view %s, not view %s", teamID, team.ViewID, viewID)
	}
	return nil
}

func (r *playerViewDefaultTeamResource) read(model *playerViewDefaultTeamModel) (bool, error) {
	viewID := model.ViewID.ValueString()
	if viewID == "" {
		viewID = model.ID.ValueString()
	}
	exists, err := api.ViewExists(viewID, r.cfg)
	if err != nil || !exists {
		return false, err
	}
	view, err := api.ReadViewTopLevel(viewID, r.cfg)
	if err != nil {
		return false, err
	}
	if view.DefaultTeamID == "" {
		return false, nil
	}
	model.ID = types.StringValue(viewID)
	model.ViewID = types.StringValue(viewID)
	model.TeamID = types.StringValue(view.DefaultTeamID)
	return true, nil
}

func (r *playerViewDefaultTeamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan playerViewDefaultTeamModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	viewID := plan.ViewID.ValueString()
	teamID := plan.TeamID.ValueString()
	if err := r.validateTeam(viewID, teamID); err != nil {
		resp.Diagnostics.AddError("Invalid Player default team", err.Error())
		return
	}

	defaultTeamCreateMu.Lock()
	defer defaultTeamCreateMu.Unlock()

	view, err := api.ReadViewTopLevel(viewID, r.cfg)
	if err != nil {
		resp.Diagnostics.AddError("Error reading Player view", err.Error())
		return
	}
	if view.DefaultTeamID != "" {
		resp.Diagnostics.AddError(
			"Player view already has a default team",
			fmt.Sprintf("View %s already uses team %s as its default. Import the existing association or manage only one crucible_player_view_default_team resource for the view.", viewID, view.DefaultTeamID),
		)
		return
	}
	if err := api.ReconcileDefaultTeam(viewID, teamID, true, r.cfg); err != nil {
		resp.Diagnostics.AddError("Error setting Player default team", err.Error())
		return
	}

	plan.ID = types.StringValue(viewID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *playerViewDefaultTeamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state playerViewDefaultTeamModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	exists, err := r.read(&state)
	if err != nil {
		resp.Diagnostics.AddError("Error reading Player default team", err.Error())
		return
	}
	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *playerViewDefaultTeamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan playerViewDefaultTeamModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	viewID := plan.ViewID.ValueString()
	teamID := plan.TeamID.ValueString()
	if err := r.validateTeam(viewID, teamID); err != nil {
		resp.Diagnostics.AddError("Invalid Player default team", err.Error())
		return
	}
	if err := api.ReconcileDefaultTeam(viewID, teamID, true, r.cfg); err != nil {
		resp.Diagnostics.AddError("Error updating Player default team", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *playerViewDefaultTeamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state playerViewDefaultTeamModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	viewID := state.ViewID.ValueString()
	exists, err := api.ViewExists(viewID, r.cfg)
	if err != nil {
		resp.Diagnostics.AddError("Error checking Player view", err.Error())
		return
	}
	if !exists {
		return
	}
	if err := api.ReconcileDefaultTeam(viewID, state.TeamID.ValueString(), false, r.cfg); err != nil {
		resp.Diagnostics.AddError("Error clearing Player default team", err.Error())
	}
}

func (r *playerViewDefaultTeamResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("view_id"), req.ID)...)
}
