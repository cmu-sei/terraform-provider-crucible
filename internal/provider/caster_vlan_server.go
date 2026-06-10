// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cmu-sei/terraform-provider-crucible/internal/api"
	"github.com/cmu-sei/terraform-provider-crucible/internal/structs"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &vlanResource{}
	_ resource.ResourceWithConfigure   = &vlanResource{}
	_ resource.ResourceWithImportState = &vlanResource{}
)

// vlanResource is the resource implementation for crucible_vlan.
type vlanResource struct {
	cfg map[string]string
}

type vlanModel struct {
	ID          types.String `tfsdk:"id"`
	PartitionID types.String `tfsdk:"partition_id"`
	PoolID      types.String `tfsdk:"pool_id"`
	ProjectID   types.String `tfsdk:"project_id"`
	Tag         types.String `tfsdk:"tag"`
	VlanID      types.Int64  `tfsdk:"vlan_id"`
}

// NewVlanResource is a helper to instantiate the resource.
func NewVlanResource() resource.Resource {
	return &vlanResource{}
}

func (r *vlanResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vlan"
}

func (r *vlanResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *vlanResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"partition_id": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("project_id")),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"pool_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"tag": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vlan_id": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *vlanResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan vlanModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cmd := &structs.VlanCreateCommand{
		ProjectId:   plan.ProjectID.ValueString(),
		PartitionId: plan.PartitionID.ValueString(),
		Tag:         plan.Tag.ValueString(),
	}

	// Only request a specific VLAN ID when one was configured.
	if !plan.VlanID.IsNull() && !plan.VlanID.IsUnknown() {
		cmd.VlanId = sql.NullInt32{Int32: int32(plan.VlanID.ValueInt64()), Valid: true}
	}

	vlan, err := api.CreateVlan(cmd, r.cfg)
	if err != nil {
		resp.Diagnostics.AddError("Error creating vlan", err.Error())
		return
	}

	plan.ID = types.StringValue(vlan.Id)
	applyVlan(&plan, vlan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *vlanResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state vlanModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The VLAN record can be removed entirely (404) if its pool/partition is
	// deleted out of band; treat that as gone rather than erroring.
	exists, err := api.VlanExists(state.ID.ValueString(), r.cfg)
	if err != nil {
		resp.Diagnostics.AddError("Error checking vlan existence", err.Error())
		return
	}
	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	vlan, err := api.ReadVlan(state.ID.ValueString(), r.cfg)
	if err != nil {
		resp.Diagnostics.AddError("Error reading vlan", err.Error())
		return
	}

	// A VLAN that is no longer in use has been released externally.
	if !vlan.InUse {
		resp.State.RemoveResource(ctx)
		return
	}

	applyVlan(&state, vlan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is required by the resource.Resource interface. Every configurable
// attribute on this resource forces replacement, so Update is never expected to
// run; it simply persists the planned values.
func (r *vlanResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan vlanModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *vlanResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state vlanModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := api.DeleteVlan(state.ID.ValueString(), r.cfg); err != nil {
		resp.Diagnostics.AddError("Error deleting vlan", err.Error())
	}
}

func (r *vlanResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// applyVlan copies the API representation of a VLAN into the model.
func applyVlan(m *vlanModel, vlan *structs.Vlan) {
	m.VlanID = types.Int64Value(int64(vlan.VlanId))
	m.PoolID = types.StringValue(vlan.PoolId)
	m.PartitionID = types.StringValue(vlan.PartitionId)
	m.Tag = types.StringValue(vlan.Tag)
}
