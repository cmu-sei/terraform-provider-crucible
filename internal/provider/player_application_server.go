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
	_ resource.Resource                = &playerApplicationResource{}
	_ resource.ResourceWithConfigure   = &playerApplicationResource{}
	_ resource.ResourceWithImportState = &playerApplicationResource{}
)

type playerApplicationResource struct{ playerConfiguredResource }

type playerApplicationModel struct {
	ID                    types.String `tfsdk:"id"`
	ViewID                types.String `tfsdk:"view_id"`
	Name                  types.String `tfsdk:"name"`
	URL                   types.String `tfsdk:"url"`
	Icon                  types.String `tfsdk:"icon"`
	Embeddable            types.Bool   `tfsdk:"embeddable"`
	LoadInBackground      types.Bool   `tfsdk:"load_in_background"`
	ApplicationTemplateID types.String `tfsdk:"application_template_id"`
}

func NewPlayerApplicationResource() resource.Resource { return &playerApplicationResource{} }

func (r *playerApplicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_player_application"
}

func (r *playerApplicationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages one application in a Player view configured for standalone child management.",
		Attributes: map[string]schema.Attribute{
			"id":      computedIDAttribute(),
			"view_id": requiredUUIDAttribute("UUID of the owning Player view."),
			"name":    requiredNameAttribute("Application name."),
			"url": schema.StringAttribute{
				Optional:    true,
				Description: "Application URL.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(httpURLRegex, "must be a valid URL beginning with http:// or https://"),
				},
			},
			"icon":               schema.StringAttribute{Optional: true, Description: "Application icon value."},
			"embeddable":         schema.BoolAttribute{Optional: true, Description: "Whether the application can be embedded."},
			"load_in_background": schema.BoolAttribute{Optional: true, Description: "Whether the application loads in the background."},
			"application_template_id": schema.StringAttribute{
				Optional:    true,
				Description: "Optional application-template UUID.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex, "must be a valid UUID"),
				},
			},
		},
	}
}

func applicationFromModel(m *playerApplicationModel) *api.PlayerApplication {
	return &api.PlayerApplication{
		ID:                    m.ID.ValueString(),
		ViewID:                m.ViewID.ValueString(),
		Name:                  m.Name.ValueString(),
		URL:                   optionalString(m.URL),
		Icon:                  optionalString(m.Icon),
		Embeddable:            optionalBool(m.Embeddable),
		LoadInBackground:      optionalBool(m.LoadInBackground),
		ApplicationTemplateID: optionalString(m.ApplicationTemplateID),
	}
}

func (r *playerApplicationResource) read(model *playerApplicationModel) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	value, exists, err := api.ReadPlayerApplication(model.ID.ValueString(), r.cfg)
	if err != nil {
		diags.AddError("Error reading Player application", err.Error())
		return false, diags
	}
	if !exists {
		return false, diags
	}
	model.ID = types.StringValue(value.ID)
	model.ViewID = types.StringValue(value.ViewID)
	model.Name = types.StringValue(value.Name)
	model.URL = stringValue(value.URL)
	model.Icon = stringValue(value.Icon)
	model.Embeddable = boolValue(value.Embeddable)
	model.LoadInBackground = boolValue(value.LoadInBackground)
	model.ApplicationTemplateID = stringValue(value.ApplicationTemplateID)
	return true, diags
}

func (r *playerApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan playerApplicationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	value := applicationFromModel(&plan)
	if err := api.CreatePlayerApplication(value, r.cfg); err != nil {
		resp.Diagnostics.AddError("Error creating Player application", err.Error())
		return
	}
	plan.ID = types.StringValue(value.ID)
	_, diags := r.read(&plan)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *playerApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state playerApplicationModel
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

func (r *playerApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan playerApplicationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := api.UpdatePlayerApplication(applicationFromModel(&plan), r.cfg); err != nil {
		resp.Diagnostics.AddError("Error updating Player application", err.Error())
		return
	}
	_, diags := r.read(&plan)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *playerApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state playerApplicationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := api.DeletePlayerApplication(state.ID.ValueString(), r.cfg); err != nil {
		resp.Diagnostics.AddError("Error deleting Player application", err.Error())
	}
}

func (r *playerApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
