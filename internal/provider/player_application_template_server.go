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

var (
	_ resource.Resource                = &applicationTemplateResource{}
	_ resource.ResourceWithConfigure   = &applicationTemplateResource{}
	_ resource.ResourceWithImportState = &applicationTemplateResource{}
)

// applicationTemplateResource is the resource implementation for
// crucible_player_application_template.
type applicationTemplateResource struct {
	cfg map[string]string
}

type applicationTemplateModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	URL              types.String `tfsdk:"url"`
	Icon             types.String `tfsdk:"icon"`
	Embeddable       types.Bool   `tfsdk:"embeddable"`
	LoadInBackground types.Bool   `tfsdk:"load_in_background"`
}

// NewApplicationTemplateResource is a helper to instantiate the resource.
func NewApplicationTemplateResource() resource.Resource {
	return &applicationTemplateResource{}
}

func (r *applicationTemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_player_application_template"
}

func (r *applicationTemplateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *applicationTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			// These attributes are Optional+Computed: the API echoes them back
			// (defaulting empty/false), so the provider must be allowed to set
			// them even when the configuration omits them.
			"url": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"icon": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"embeddable": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"load_in_background": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
		},
	}
}

func (r *applicationTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan applicationTemplateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	template := &structs.AppTemplate{
		Name:             plan.Name.ValueString(),
		URL:              plan.URL.ValueString(),
		Icon:             plan.Icon.ValueString(),
		Embeddable:       plan.Embeddable.ValueBool(),
		LoadInBackground: plan.LoadInBackground.ValueBool(),
	}

	id, err := api.CreateAppTemplate(template, r.cfg)
	if err != nil {
		resp.Diagnostics.AddError("Error creating application template", err.Error())
		return
	}

	plan.ID = types.StringValue(id)

	resp.Diagnostics.Append(r.read(&plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *applicationTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state applicationTemplateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, err := api.AppTemplateExists(state.ID.ValueString(), r.cfg)
	if err != nil {
		resp.Diagnostics.AddError("Error checking application template existence", err.Error())
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

func (r *applicationTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan applicationTemplateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	template := &structs.AppTemplate{
		Name:             plan.Name.ValueString(),
		URL:              plan.URL.ValueString(),
		Icon:             plan.Icon.ValueString(),
		Embeddable:       plan.Embeddable.ValueBool(),
		LoadInBackground: plan.LoadInBackground.ValueBool(),
	}

	if err := api.AppTemplateUpdate(plan.ID.ValueString(), template, r.cfg); err != nil {
		resp.Diagnostics.AddError("Error updating application template", err.Error())
		return
	}

	resp.Diagnostics.Append(r.read(&plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *applicationTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state applicationTemplateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	exists, err := api.AppTemplateExists(id, r.cfg)
	if err != nil {
		resp.Diagnostics.AddError("Error checking application template existence", err.Error())
		return
	}
	if !exists {
		return
	}

	if err := api.DeleteAppTemplate(id, r.cfg); err != nil {
		resp.Diagnostics.AddError("Error deleting application template", err.Error())
	}
}

func (r *applicationTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// read refreshes the model from the API.
func (r *applicationTemplateResource) read(m *applicationTemplateModel) diag.Diagnostics {
	var diags diag.Diagnostics

	template, err := api.AppTemplateRead(m.ID.ValueString(), r.cfg)
	if err != nil {
		diags.AddError("Error reading application template", err.Error())
		return diags
	}

	m.Name = types.StringValue(template.Name)
	m.URL = types.StringValue(template.URL)
	m.Icon = types.StringValue(template.Icon)
	m.Embeddable = types.BoolValue(template.Embeddable)
	m.LoadInBackground = types.BoolValue(template.LoadInBackground)

	return diags
}
