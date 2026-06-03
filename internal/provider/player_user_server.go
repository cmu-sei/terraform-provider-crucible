// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider

import (
	"context"
	"fmt"

	"github.com/cmu-sei/terraform-provider-crucible/internal/api"
	"github.com/cmu-sei/terraform-provider-crucible/internal/structs"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &userResource{}
	_ resource.ResourceWithConfigure   = &userResource{}
	_ resource.ResourceWithImportState = &userResource{}
)

// userResource is the resource implementation for crucible_player_user.
type userResource struct {
	cfg map[string]string
}

// userModel maps the crucible_player_user schema to a Go type.
type userModel struct {
	ID     types.String `tfsdk:"id"`
	UserID types.String `tfsdk:"user_id"`
	Name   types.String `tfsdk:"name"`
	Role   types.String `tfsdk:"role"`
}

// NewUserResource is a helper to instantiate the resource.
func NewUserResource() resource.Resource {
	return &userResource{}
}

func (r *userResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_player_user"
}

func (r *userResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *userResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"role": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
		},
	}
}

func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user := structs.PlayerUser{
		ID:   plan.UserID.ValueString(),
		Name: plan.Name.ValueString(),
		Role: plan.Role.ValueString(),
	}

	if err := api.CreateUser(user, r.cfg); err != nil {
		resp.Diagnostics.AddError("Error creating user", err.Error())
		return
	}

	plan.ID = types.StringValue(user.ID)

	// Read back to populate computed values (e.g. role name).
	resp.Diagnostics.Append(r.read(&plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, err := api.UserExists(state.ID.ValueString(), r.cfg)
	if err != nil {
		resp.Diagnostics.AddError("Error checking user existence", err.Error())
		return
	}
	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(r.read(&state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan userModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user := structs.PlayerUser{
		ID:   plan.UserID.ValueString(),
		Name: plan.Name.ValueString(),
		Role: plan.Role.ValueString(),
	}

	if err := api.UpdateUser(user, r.cfg); err != nil {
		resp.Diagnostics.AddError("Error updating user", err.Error())
		return
	}

	plan.ID = types.StringValue(user.ID)

	resp.Diagnostics.Append(r.read(&plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	exists, err := api.UserExists(id, r.cfg)
	if err != nil {
		resp.Diagnostics.AddError("Error checking user existence", err.Error())
		return
	}
	if !exists {
		return
	}

	if err := api.DeleteUser(id, r.cfg); err != nil {
		resp.Diagnostics.AddError("Error deleting user", err.Error())
	}
}

func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), req.ID)...)
}

// read refreshes the model from the API, resolving the role ID back to its
// display name as the SDKv1 implementation did.
func (r *userResource) read(m *userModel) diag.Diagnostics {
	var diags diag.Diagnostics

	user, err := api.ReadUser(m.ID.ValueString(), r.cfg)
	if err != nil {
		diags.AddError("Error reading user", err.Error())
		return diags
	}

	m.UserID = types.StringValue(user.ID)
	m.Name = types.StringValue(user.Name)

	// Resolve the role ID returned by the API back to its display name.
	if roleID, ok := user.Role.(string); ok && roleID != "" {
		role, err := api.GetRoleByID(roleID, r.cfg)
		if err != nil {
			diags.AddError("Error reading user role", err.Error())
			return diags
		}
		m.Role = types.StringValue(role)
	} else {
		m.Role = types.StringNull()
	}

	return diags
}
