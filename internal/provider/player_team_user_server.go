// Copyright 2026 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider

import (
	"context"

	"github.com/cmu-sei/terraform-provider-crucible/internal/api"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &playerTeamUserResource{}
	_ resource.ResourceWithConfigure   = &playerTeamUserResource{}
	_ resource.ResourceWithImportState = &playerTeamUserResource{}
)

type playerTeamUserResource struct{ playerConfiguredResource }

type playerTeamUserModel struct {
	ID     types.String `tfsdk:"id"`
	TeamID types.String `tfsdk:"team_id"`
	UserID types.String `tfsdk:"user_id"`
	Role   types.String `tfsdk:"role"`
}

func NewPlayerTeamUserResource() resource.Resource { return &playerTeamUserResource{} }

func (r *playerTeamUserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_player_team_user"
}

func (r *playerTeamUserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages one Player user-to-team membership.",
		Attributes: map[string]schema.Attribute{
			"id":      computedIDAttribute(),
			"team_id": requiredUUIDAttribute("UUID of the membership's team."),
			"user_id": requiredUUIDAttribute("UUID of the membership's user."),
			"role": schema.StringAttribute{
				Optional:    true,
				Description: "Optional membership role override. Omit to inherit the team role.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

func membershipFromModel(model *playerTeamUserModel) *api.PlayerTeamUser {
	return &api.PlayerTeamUser{ID: model.ID.ValueString(), TeamID: model.TeamID.ValueString(), UserID: model.UserID.ValueString(), Role: optionalString(model.Role)}
}

func (r *playerTeamUserResource) read(model *playerTeamUserModel) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	value, exists, err := api.ReadPlayerTeamUser(model.ID.ValueString(), r.cfg)
	if err != nil {
		diags.AddError("Error reading Player team membership", err.Error())
		return false, diags
	}
	if !exists {
		return false, diags
	}
	model.ID = types.StringValue(value.ID)
	model.TeamID = types.StringValue(value.TeamID)
	model.UserID = types.StringValue(value.UserID)
	model.Role = stringValue(value.Role)
	return true, diags
}

func (r *playerTeamUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan playerTeamUserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	value := membershipFromModel(&plan)
	if err := api.CreatePlayerTeamUser(value, r.cfg); err != nil {
		resp.Diagnostics.AddError("Error creating Player team membership", err.Error())
		return
	}
	plan.ID = types.StringValue(value.ID)
	_, diags := r.read(&plan)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *playerTeamUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state playerTeamUserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	exists, diags := r.read(&state)
	resp.Diagnostics.Append(diags...)
	if !exists && !resp.Diagnostics.HasError() {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *playerTeamUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan playerTeamUserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := api.UpdatePlayerTeamUser(membershipFromModel(&plan), r.cfg); err != nil {
		resp.Diagnostics.AddError("Error updating Player team membership", err.Error())
		return
	}
	_, diags := r.read(&plan)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *playerTeamUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state playerTeamUserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := api.DeletePlayerTeamUser(state.TeamID.ValueString(), state.UserID.ValueString(), r.cfg); err != nil {
		resp.Diagnostics.AddError("Error deleting Player team membership", err.Error())
	}
}

func (r *playerTeamUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
