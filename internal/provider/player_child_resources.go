// Copyright 2026 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/cmu-sei/terraform-provider-crucible/internal/api"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var uuidRegex = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

type playerConfiguredResource struct {
	cfg map[string]string
}

func (r *playerConfiguredResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	cfg, ok := req.ProviderData.(map[string]string)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Provider Configuration Type", fmt.Sprintf("Expected map[string]string, got %T.", req.ProviderData))
		return
	}
	r.cfg = cfg
}

func computedIDAttribute() schema.StringAttribute {
	return schema.StringAttribute{
		Computed:    true,
		Description: "The remote UUID.",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
}

func requiredUUIDAttribute(description string) schema.StringAttribute {
	return schema.StringAttribute{
		Required:    true,
		Description: description,
		Validators: []validator.String{
			stringvalidator.RegexMatches(uuidRegex, "must be a valid UUID"),
		},
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
	}
}

func requiredNameAttribute(description string) schema.StringAttribute {
	return schema.StringAttribute{
		Required:    true,
		Description: description,
		Validators: []validator.String{
			stringvalidator.LengthAtLeast(1),
		},
	}
}

func emptyStringSet() types.Set {
	return types.SetValueMust(types.StringType, []attr.Value{})
}

func optionalString(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	value := v.ValueString()
	return &value
}

func optionalBool(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	value := v.ValueBool()
	return &value
}

func stringValue(v *string) types.String {
	if v == nil {
		return types.StringNull()
	}
	return types.StringValue(*v)
}

func boolValue(v *bool) types.Bool {
	if v == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*v)
}

func setStrings(ctx context.Context, value types.Set) ([]string, diag.Diagnostics) {
	var values []string
	diags := value.ElementsAs(ctx, &values, false)
	return values, diags
}

func stringsSet(values []string) types.Set {
	elements := make([]attr.Value, 0, len(values))
	for _, value := range values {
		elements = append(elements, types.StringValue(value))
	}
	return types.SetValueMust(types.StringType, elements)
}

// Application resource.

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
		Description: "Manages one application in a separately managed Player view.",
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

// Team resource.

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
		Description: "Manages one team in a separately managed Player view.",
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

// Team-user membership resource.

var (
	_ resource.Resource                = &playerTeamUserResource{}
	_ resource.ResourceWithConfigure   = &playerTeamUserResource{}
	_ resource.ResourceWithImportState = &playerTeamUserResource{}
)

type playerTeamUserResource struct{ playerConfiguredResource }

type playerTeamUserModel struct {
	ID     types.String `tfsdk:"id"`
	ViewID types.String `tfsdk:"view_id"`
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
			"view_id": requiredUUIDAttribute("UUID of the membership's Player view."),
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
	return &api.PlayerTeamUser{ID: model.ID.ValueString(), ViewID: model.ViewID.ValueString(), TeamID: model.TeamID.ValueString(), UserID: model.UserID.ValueString(), Role: optionalString(model.Role)}
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
	model.ViewID = types.StringValue(value.ViewID)
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

// Application-instance resource.

var (
	_ resource.Resource                = &playerApplicationInstanceResource{}
	_ resource.ResourceWithConfigure   = &playerApplicationInstanceResource{}
	_ resource.ResourceWithImportState = &playerApplicationInstanceResource{}
)

type playerApplicationInstanceResource struct{ playerConfiguredResource }

type playerApplicationInstanceModel struct {
	ID            types.String  `tfsdk:"id"`
	TeamID        types.String  `tfsdk:"team_id"`
	ApplicationID types.String  `tfsdk:"application_id"`
	DisplayOrder  types.Float64 `tfsdk:"display_order"`
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
	model.DisplayOrder = types.Float64Value(value.DisplayOrder)
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
