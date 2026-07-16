// Copyright 2026 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider

import (
	"context"
	"strings"

	"github.com/cmu-sei/terraform-provider-crucible/internal/api"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64default"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &playerApplicationInstanceResource{}
	_ resource.ResourceWithConfigure   = &playerApplicationInstanceResource{}
	_ resource.ResourceWithImportState = &playerApplicationInstanceResource{}
)

type playerApplicationInstanceResource struct{ playerConfiguredResource }

type playerApplicationInstanceModel struct {
	ID            types.String       `tfsdk:"id"`
	TeamID        types.String       `tfsdk:"team_id"`
	ApplicationID types.String       `tfsdk:"application_id"`
	DisplayOrder  playerFloat32Value `tfsdk:"display_order"`
}

func NewPlayerApplicationInstanceResource() resource.Resource {
	return &playerApplicationInstanceResource{}
}

func (r *playerApplicationInstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_player_application_instance"
}

func (r *playerApplicationInstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages one application instance assigned to a Player team.",
		Attributes: map[string]schema.Attribute{
			"id":             computedIDAttribute(),
			"team_id":        requiredUUIDAttribute("UUID of the owning Player team."),
			"application_id": requiredUUIDAttribute("UUID of the Player application."),
			"display_order": schema.Float64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     float64default.StaticFloat64(0),
				CustomType:  playerFloat32Type{},
				Description: "Display order within the team.",
			},
		},
	}
}

func instanceFromModel(model *playerApplicationInstanceModel) *api.PlayerApplicationInstance {
	return &api.PlayerApplicationInstance{ID: model.ID.ValueString(), TeamID: model.TeamID.ValueString(), ApplicationID: model.ApplicationID.ValueString(), DisplayOrder: model.DisplayOrder.ValueFloat64()}
}

func (r *playerApplicationInstanceResource) read(model *playerApplicationInstanceModel) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	value, exists, err := api.ReadPlayerApplicationInstance(model.ID.ValueString(), model.TeamID.ValueString(), r.cfg)
	if err != nil {
		diags.AddError("Error reading Player application instance", err.Error())
		return false, diags
	}
	if !exists {
		return false, diags
	}
	model.ID = types.StringValue(value.ID)
	model.TeamID = types.StringValue(value.TeamID)
	model.ApplicationID = types.StringValue(value.ApplicationID)
	model.DisplayOrder = newPlayerFloat32Value(value.DisplayOrder)
	return true, diags
}

func (r *playerApplicationInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan playerApplicationInstanceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	value := instanceFromModel(&plan)
	if err := api.CreatePlayerApplicationInstance(value, r.cfg); err != nil {
		resp.Diagnostics.AddError("Error creating Player application instance", err.Error())
		return
	}
	plan.ID = types.StringValue(value.ID)
	_, diags := r.read(&plan)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *playerApplicationInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state playerApplicationInstanceModel
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

func (r *playerApplicationInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan playerApplicationInstanceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := api.UpdatePlayerApplicationInstance(instanceFromModel(&plan), r.cfg); err != nil {
		resp.Diagnostics.AddError("Error updating Player application instance", err.Error())
		return
	}
	_, diags := r.read(&plan)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *playerApplicationInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state playerApplicationInstanceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := api.DeletePlayerApplicationInstance(state.ID.ValueString(), r.cfg); err != nil {
		resp.Diagnostics.AddError("Error deleting Player application instance", err.Error())
	}
}

func (r *playerApplicationInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || !uuidRegex.MatchString(parts[0]) || !uuidRegex.MatchString(parts[1]) {
		resp.Diagnostics.AddError("Invalid application-instance import identifier", "Expected <team_uuid>/<instance_uuid>.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("team_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
