// Copyright 2026 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider

import (
	"context"

	"github.com/cmu-sei/terraform-provider-crucible/internal/api"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &playerTeamResource{}
	_ resource.ResourceWithConfigure   = &playerTeamResource{}
	_ resource.ResourceWithImportState = &playerTeamResource{}
)

type playerTeamResource struct{ playerConfiguredResource }

type playerTeamModel struct {
	ID            types.String `tfsdk:"id"`
	ViewID        types.String `tfsdk:"view_id"`
	Name          types.String `tfsdk:"name"`
	Role          types.String `tfsdk:"role"`
	Permissions   types.Set    `tfsdk:"permissions"`
	ScopedTeamIDs types.Set    `tfsdk:"scoped_team_ids"`
}

func NewPlayerTeamResource() resource.Resource { return &playerTeamResource{} }

func (r *playerTeamResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_player_team"
}

func (r *playerTeamResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	uuidSet := func(description string) schema.SetAttribute {
		return schema.SetAttribute{
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			Default:     setdefault.StaticValue(emptyStringSet()),
			Description: description,
			Validators: []validator.Set{
				setvalidator.ValueStringsAre(
					stringvalidator.RegexMatches(uuidRegex, "must be a valid UUID"),
				),
				setvalidator.NoNullValues(),
			},
		}
	}
	resp.Schema = schema.Schema{
		Description: "Manages one team in a Player view configured for standalone child management.",
		Attributes: map[string]schema.Attribute{
			"id":      computedIDAttribute(),
			"view_id": requiredUUIDAttribute("UUID of the owning Player view."),
			"name":    requiredNameAttribute("Team name."),
			"role": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("View Member"),
				Description: "Team role name.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"permissions":     uuidSet("Authoritative unordered set of team-permission UUIDs."),
			"scoped_team_ids": uuidSet("Authoritative unordered set of target team UUIDs for permission scopes."),
		},
	}
}

func teamFromModel(ctx context.Context, model *playerTeamModel) (*api.PlayerTeam, diag.Diagnostics) {
	permissions, diags := setStrings(ctx, model.Permissions)
	scopes, scopeDiags := setStrings(ctx, model.ScopedTeamIDs)
	diags.Append(scopeDiags...)
	return &api.PlayerTeam{ID: model.ID.ValueString(), ViewID: model.ViewID.ValueString(), Name: model.Name.ValueString(), Role: model.Role.ValueString(), Permissions: permissions, ScopedTeamIDs: scopes}, diags
}

func (r *playerTeamResource) read(model *playerTeamModel) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	value, exists, err := api.ReadPlayerTeam(model.ID.ValueString(), r.cfg)
	if err != nil {
		diags.AddError("Error reading Player team", err.Error())
		return false, diags
	}
	if !exists {
		return false, diags
	}
	model.ID = types.StringValue(value.ID)
	model.ViewID = types.StringValue(value.ViewID)
	model.Name = types.StringValue(value.Name)
	model.Role = types.StringValue(value.Role)
	model.Permissions = stringsSet(value.Permissions)
	model.ScopedTeamIDs = stringsSet(value.ScopedTeamIDs)
	return true, diags
}

func (r *playerTeamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan playerTeamModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	value, diags := teamFromModel(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := api.CreatePlayerTeam(value, r.cfg); err != nil {
		resp.Diagnostics.AddError("Error creating Player team", err.Error())
		return
	}
	plan.ID = types.StringValue(value.ID)
	_, readDiags := r.read(&plan)
	resp.Diagnostics.Append(readDiags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *playerTeamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state playerTeamModel
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

func (r *playerTeamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan playerTeamModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	value, diags := teamFromModel(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := api.UpdatePlayerTeam(value, r.cfg); err != nil {
		resp.Diagnostics.AddError("Error updating Player team", err.Error())
		return
	}
	_, readDiags := r.read(&plan)
	resp.Diagnostics.Append(readDiags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *playerTeamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state playerTeamModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := api.DeletePlayerTeam(state.ID.ValueString(), r.cfg); err != nil {
		resp.Diagnostics.AddError("Error deleting Player team", err.Error())
	}
}

func (r *playerTeamResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
