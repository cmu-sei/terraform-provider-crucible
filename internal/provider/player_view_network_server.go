// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider

import (
	"context"
	"fmt"

	"github.com/cmu-sei/terraform-provider-crucible/internal/api"
	"github.com/cmu-sei/terraform-provider-crucible/internal/structs"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &viewNetworkResource{}
	_ resource.ResourceWithConfigure   = &viewNetworkResource{}
	_ resource.ResourceWithImportState = &viewNetworkResource{}
)

// viewNetworkResource is the resource implementation for
// crucible_player_view_network.
type viewNetworkResource struct {
	cfg map[string]string
}

type viewNetworkModel struct {
	ID                 types.String `tfsdk:"id"`
	ViewID             types.String `tfsdk:"view_id"`
	ProviderType       types.String `tfsdk:"provider_type"`
	ProviderInstanceID types.String `tfsdk:"provider_instance_id"`
	NetworkID          types.String `tfsdk:"network_id"`
	Name               types.String `tfsdk:"name"`
	TeamIDs            types.List   `tfsdk:"team_ids"`
}

// NewViewNetworkResource is a helper to instantiate the resource.
func NewViewNetworkResource() resource.Resource {
	return &viewNetworkResource{}
}

func (r *viewNetworkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_player_view_network"
}

func (r *viewNetworkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	cfg, ok := req.ProviderData.(map[string]string)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Configuration Type",
			fmt.Sprintf("Expected map[string]string, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.cfg = cfg
}

func (r *viewNetworkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"view_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"provider_type": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf("Unknown", "Vsphere", "Proxmox", "Azure"),
				},
			},
			"provider_instance_id": schema.StringAttribute{
				Required: true,
			},
			"network_id": schema.StringAttribute{
				Required: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"team_ids": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *viewNetworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan viewNetworkModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamIDs, diags := toStringSlice(ctx, plan.TeamIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	network := &structs.ViewNetworkInfo{
		ViewID:             plan.ViewID.ValueString(),
		ProviderType:       plan.ProviderType.ValueString(),
		ProviderInstanceId: plan.ProviderInstanceID.ValueString(),
		NetworkId:          plan.NetworkID.ValueString(),
		Name:               plan.Name.ValueString(),
		TeamIds:            teamIDs,
	}

	result, err := api.CreateViewNetwork(network, r.cfg)
	if err != nil {
		resp.Diagnostics.AddError("Error creating view network", err.Error())
		return
	}

	plan.ID = types.StringValue(result.ID)

	resp.Diagnostics.Append(r.read(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *viewNetworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state viewNetworkModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, err := api.ViewNetworkExists(state.ViewID.ValueString(), state.ID.ValueString(), r.cfg)
	if err != nil {
		resp.Diagnostics.AddError("Error checking view network existence", err.Error())
		return
	}
	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(r.read(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *viewNetworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan viewNetworkModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamIDs, diags := toStringSlice(ctx, plan.TeamIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	network := &structs.ViewNetworkInfo{
		ID:                 plan.ID.ValueString(),
		ViewID:             plan.ViewID.ValueString(),
		ProviderType:       plan.ProviderType.ValueString(),
		ProviderInstanceId: plan.ProviderInstanceID.ValueString(),
		NetworkId:          plan.NetworkID.ValueString(),
		Name:               plan.Name.ValueString(),
		TeamIds:            teamIDs,
	}

	if err := api.UpdateViewNetwork(network, r.cfg); err != nil {
		resp.Diagnostics.AddError("Error updating view network", err.Error())
		return
	}

	resp.Diagnostics.Append(r.read(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *viewNetworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state viewNetworkModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	viewID := state.ViewID.ValueString()
	id := state.ID.ValueString()
	exists, err := api.ViewNetworkExists(viewID, id, r.cfg)
	if err != nil {
		resp.Diagnostics.AddError("Error checking view network existence", err.Error())
		return
	}
	if !exists {
		return
	}

	if err := api.DeleteViewNetwork(viewID, id, r.cfg); err != nil {
		resp.Diagnostics.AddError("Error deleting view network", err.Error())
	}
}

func (r *viewNetworkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// read refreshes the model from the API.
func (r *viewNetworkResource) read(ctx context.Context, m *viewNetworkModel) diag.Diagnostics {
	var diags diag.Diagnostics

	network, err := api.GetViewNetwork(m.ViewID.ValueString(), m.ID.ValueString(), r.cfg)
	if err != nil {
		diags.AddError("Error reading view network", err.Error())
		return diags
	}

	m.ViewID = types.StringValue(network.ViewID)
	m.ProviderType = types.StringValue(network.ProviderType)
	m.ProviderInstanceID = types.StringValue(network.ProviderInstanceId)
	m.NetworkID = types.StringValue(network.NetworkId)
	m.Name = types.StringValue(network.Name)

	// team_ids: the API returns ids in arbitrary order, so preserve the order
	// already in the model (config order on create, state order on refresh) to
	// avoid "inconsistent result after apply"; see orderTeamIDs.
	priorTeamIDs, d := toStringSlice(ctx, m.TeamIDs)
	diags.Append(d...)
	teamIDs, d := types.ListValueFrom(ctx, types.StringType, orderTeamIDs(priorTeamIDs, network.TeamIds))
	diags.Append(d...)
	m.TeamIDs = teamIDs

	return diags
}
