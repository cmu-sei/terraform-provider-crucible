// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"sort"

	"github.com/cmu-sei/terraform-provider-crucible/internal/api"
	"github.com/cmu-sei/terraform-provider-crucible/internal/structs"
	"github.com/cmu-sei/terraform-provider-crucible/internal/util"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                     = &viewResource{}
	_ resource.ResourceWithConfigure        = &viewResource{}
	_ resource.ResourceWithConfigValidators = &viewResource{}
	_ resource.ResourceWithImportState      = &viewResource{}
)

// viewResource is the resource implementation for crucible_player_view.
type viewResource struct {
	cfg map[string]string
}

type viewModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	Status          types.String `tfsdk:"status"`
	CreateAdminTeam types.Bool   `tfsdk:"create_admin_team"`
	ChildManagement types.String `tfsdk:"child_management"`
	Application     types.List   `tfsdk:"application"`
	Team            types.List   `tfsdk:"team"`
}

type viewChildManagementValidator struct{}

func (viewChildManagementValidator) Description(context.Context) string {
	return "separate child management cannot be combined with inline children or automatic Admin-team creation"
}

func (v viewChildManagementValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (viewChildManagementValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config viewModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || config.ChildManagement.IsNull() || config.ChildManagement.IsUnknown() || config.ChildManagement.ValueString() != "separate" {
		return
	}
	if !config.Application.IsNull() && !config.Application.IsUnknown() && len(config.Application.Elements()) > 0 {
		resp.Diagnostics.AddAttributeError(path.Root("application"), "Invalid child ownership configuration", "application blocks cannot be configured when child_management is \"separate\".")
	}
	if !config.Team.IsNull() && !config.Team.IsUnknown() && len(config.Team.Elements()) > 0 {
		resp.Diagnostics.AddAttributeError(path.Root("team"), "Invalid child ownership configuration", "team blocks cannot be configured when child_management is \"separate\".")
	}
	if config.CreateAdminTeam.IsNull() || (!config.CreateAdminTeam.IsUnknown() && config.CreateAdminTeam.ValueBool()) {
		resp.Diagnostics.AddAttributeError(path.Root("create_admin_team"), "Invalid child ownership configuration", "create_admin_team must be explicitly set to false when child_management is \"separate\".")
	}
}

// Object attribute types for the nested blocks. These mirror the SDKv1 schema
// exactly (note that the application block's boolean-like fields are strings).
var viewUserAttrTypes = map[string]attr.Type{
	"user_id": types.StringType,
	"role":    types.StringType,
}

var viewAppInstanceAttrTypes = map[string]attr.Type{
	"name":          types.StringType,
	"display_order": types.Float64Type,
	"id":            types.StringType,
}

var viewAppAttrTypes = map[string]attr.Type{
	"app_id":             types.StringType,
	"name":               types.StringType,
	"url":                types.StringType,
	"icon":               types.StringType,
	"embeddable":         types.StringType,
	"load_in_background": types.StringType,
	"app_template_id":    types.StringType,
	"v_id":               types.StringType,
}

var viewTeamAttrTypes = map[string]attr.Type{
	"team_id":      types.StringType,
	"name":         types.StringType,
	"role":         types.StringType,
	"permissions":  types.ListType{ElemType: types.StringType},
	"scoped_teams": types.SetType{ElemType: types.StringType},
	"app_instance": types.ListType{ElemType: types.ObjectType{AttrTypes: viewAppInstanceAttrTypes}},
	"user":         types.ListType{ElemType: types.ObjectType{AttrTypes: viewUserAttrTypes}},
}

// NewViewResource is a helper to instantiate the resource.
func NewViewResource() resource.Resource {
	return &viewResource{}
}

func (r *viewResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_player_view"
}

func (r *viewResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *viewResource) ConfigValidators(context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{viewChildManagementValidator{}}
}

func (r *viewResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	// computedString builds a computed, optional string attribute whose value is
	// carried forward from prior state when the configuration omits it. This is
	// the framework equivalent of SDKv1's positional computed-value carry on
	// nested TypeList resources.
	computedString := func() schema.StringAttribute {
		return schema.StringAttribute{
			Optional: true,
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		}
	}

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
			"description": schema.StringAttribute{
				Optional: true,
			},
			"status": schema.StringAttribute{
				// Optional+Computed with a Default: the default supplies the value
				// when config omits it, so no UseStateForUnknown is needed (the
				// plan is never unknown). Verified by mutation test.
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("Active"),
			},
			"create_admin_team": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  boolStaticTrue(),
			},
			"child_management": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("inline"),
				Description: "Selects whether child applications and teams are managed by nested blocks (inline) or standalone resources (separate).",
				Validators: []validator.String{
					stringvalidator.OneOf("inline", "separate"),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"application": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"app_id": computedStringWithUseState(),
						"name": schema.StringAttribute{
							Required: true,
						},
						"url":                computedString(),
						"icon":               computedString(),
						"embeddable":         computedString(),
						"load_in_background": computedString(),
						"app_template_id":    computedString(),
						"v_id":               computedStringWithUseState(),
					},
				},
			},
			"team": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"team_id": computedStringWithUseState(),
						"name": schema.StringAttribute{
							Required: true,
						},
						"role": schema.StringAttribute{
							// Has a Default, which covers the omitted case, so no
							// UseStateForUnknown is needed. Verified by mutation test.
							Optional: true,
							Computed: true,
							Default:  stringdefault.StaticString("View Member"),
						},
						"permissions": schema.ListAttribute{
							Optional:    true,
							ElementType: types.StringType,
						},
						"scoped_teams": schema.SetAttribute{
							Optional:    true,
							Computed:    true,
							ElementType: types.StringType,
							Default: setdefault.StaticValue(
								types.SetValueMust(types.StringType, []attr.Value{}),
							),
						},
					},
					Blocks: map[string]schema.Block{
						"app_instance": schema.ListNestedBlock{
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										Required: true,
									},
									"display_order": schema.Float64Attribute{
										// Optional+Computed with NO default, so when
										// config omits it the prior value must be
										// carried forward or it re-plans as "known
										// after apply" on a sibling edit. Verified by
										// mutation test.
										Optional: true,
										Computed: true,
										PlanModifiers: []planmodifier.Float64{
											float64planmodifier.UseStateForUnknown(),
										},
									},
									"id": computedStringWithUseState(),
								},
							},
						},
						"user": schema.ListNestedBlock{
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"user_id": schema.StringAttribute{
										Required: true,
									},
									"role": schema.StringAttribute{
										// Optional+Computed: the API echoes a role
										// ("" when none is configured), so carry the
										// prior-state value forward on update instead
										// of re-planning it as "known after apply".
										// unknownIfNull covers a newly-added user that
										// has no prior state to carry.
										Optional: true,
										Computed: true,
										PlanModifiers: []planmodifier.String{
											stringplanmodifier.UseStateForUnknown(),
											unknownIfNull{},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *viewResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan viewModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scopes, diags := desiredTeamScopes(ctx, plan.Team)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	view := &structs.ViewInfo{
		Name:            plan.Name.ValueString(),
		Description:     plan.Description.ValueString(),
		Status:          plan.Status.ValueString(),
		CreateAdminTeam: plan.CreateAdminTeam.ValueBool(),
	}

	id, err := api.CreateView(view, r.cfg)
	if err != nil {
		resp.Diagnostics.AddError("Error creating view", err.Error())
		return
	}
	plan.ID = types.StringValue(id)
	if plan.ChildManagement.ValueString() == "separate" {
		resp.Diagnostics.Append(r.readTopLevel(&plan)...)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	apps, diags := expandApps(ctx, plan.Application)
	resp.Diagnostics.Append(diags...)
	teams, diags := expandTeams(ctx, plan.Team)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(apps) > 0 {
		if err := createApps(id, r.cfg, apps); err != nil {
			resp.Diagnostics.AddError("Error creating view applications", err.Error())
			return
		}
	}
	if len(teams) > 0 {
		if err := createTeams(id, r.cfg, teams, apps); err != nil {
			resp.Diagnostics.AddError("Error creating view teams", err.Error())
			return
		}
	}
	if err := api.UpdateTeamScopes(id, scopes, r.cfg); err != nil {
		resp.Diagnostics.AddError("Error creating scoped team relationships", err.Error())
		return
	}

	resp.Diagnostics.Append(r.read(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *viewResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state viewModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if state.ChildManagement.IsNull() || state.ChildManagement.IsUnknown() {
		state.ChildManagement = types.StringValue("inline")
	}

	exists, err := api.ViewExists(state.ID.ValueString(), r.cfg)
	if err != nil {
		resp.Diagnostics.AddError("Error checking view existence", err.Error())
		return
	}
	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	if state.ChildManagement.ValueString() == "separate" {
		resp.Diagnostics.Append(r.readTopLevel(&state)...)
	} else {
		resp.Diagnostics.Append(r.read(ctx, &state)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *viewResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state viewModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scopes, diags := desiredTeamScopes(ctx, plan.Team)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()

	view := &structs.ViewInfo{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Status:      plan.Status.ValueString(),
	}
	if err := api.UpdateView(view, r.cfg, id); err != nil {
		resp.Diagnostics.AddError("Error updating view", err.Error())
		return
	}
	if plan.ChildManagement.ValueString() == "separate" {
		resp.Diagnostics.Append(r.readTopLevel(&plan)...)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	oldApps, diags := expandApps(ctx, state.Application)
	resp.Diagnostics.Append(diags...)
	newApps, diags := expandApps(ctx, plan.Application)
	resp.Diagnostics.Append(diags...)
	oldTeams, diags := expandTeams(ctx, state.Team)
	resp.Diagnostics.Append(diags...)
	newTeams, diags := expandTeams(ctx, plan.Team)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.Application.Equal(state.Application) {
		if err := updateApps(id, r.cfg, oldApps, newApps); err != nil {
			resp.Diagnostics.AddError("Error updating view applications", err.Error())
			return
		}
	}
	if !plan.Team.Equal(state.Team) {
		if err := updateTeams(r.cfg, id, oldTeams, newTeams, newApps); err != nil {
			resp.Diagnostics.AddError("Error updating view teams", err.Error())
			return
		}
	}
	if err := api.UpdateTeamScopes(id, scopes, r.cfg); err != nil {
		resp.Diagnostics.AddError("Error updating scoped team relationships", err.Error())
		return
	}

	resp.Diagnostics.Append(r.read(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *viewResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state viewModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	exists, err := api.ViewExists(id, r.cfg)
	if err != nil {
		resp.Diagnostics.AddError("Error checking view existence", err.Error())
		return
	}
	if !exists {
		return
	}

	if err := api.DeleteView(id, r.cfg); err != nil {
		resp.Diagnostics.AddError("Error deleting view", err.Error())
	}
}

func (r *viewResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// read refreshes the model from the API, mirroring the sorting and Admin-team
// filtering of the SDKv1 implementation.
func (r *viewResource) read(ctx context.Context, m *viewModel) diag.Diagnostics {
	var diags diag.Diagnostics

	view, err := api.ReadView(m.ID.ValueString(), r.cfg)
	if err != nil {
		diags.AddError("Error reading view", err.Error())
		return diags
	}

	m.Name = types.StringValue(view.Name)
	if view.Description != "" || !m.Description.IsNull() {
		m.Description = types.StringValue(view.Description)
	}
	m.Status = types.StringValue(view.Status)

	// Order applications and teams to match the order already present in the
	// model (the plan during create/update, or prior state during read). The
	// framework requires the post-apply order of a ListNestedBlock to match the
	// configured order, so we must NOT impose an independent (e.g. alphabetical)
	// sort here. Names not present in the model sort to the end, alphabetically,
	// for a stable result.
	appRank := blockNameRank(ctx, m.Application)
	teamRank := blockNameRank(ctx, m.Team)
	sort.SliceStable(view.Applications, func(i, j int) bool {
		return nameLess(ifaceStr(view.Applications[i].Name), ifaceStr(view.Applications[j].Name), appRank)
	})
	sort.SliceStable(view.Teams, func(i, j int) bool {
		return nameLess(ifaceStr(view.Teams[i].Name), ifaceStr(view.Teams[j].Name), teamRank)
	})

	// Users and app instances are themselves ListNestedBlocks, so their order
	// must also match the configured order. Reorder each team's users (by
	// user_id) and app instances (by name) to the order found in the model team
	// of the same name; anything not in the model sorts to the end.
	userRanks, instRanks, permRanks := teamChildRanks(ctx, m.Team)
	for _, team := range view.Teams {
		tn := ifaceStr(team.Name)
		uRank := userRanks[tn]
		iRank := instRanks[tn]
		pRank := permRanks[tn]
		sort.SliceStable(team.Users, func(i, j int) bool {
			return nameLess(team.Users[i].ID, team.Users[j].ID, uRank)
		})
		sort.SliceStable(team.AppInstances, func(i, j int) bool {
			return nameLess(team.AppInstances[i].Name, team.AppInstances[j].Name, iRank)
		})
		sort.SliceStable(team.Permissions, func(i, j int) bool {
			return nameLess(team.Permissions[i], team.Permissions[j], pRank)
		})
	}

	// Applications.
	appObjType := types.ObjectType{AttrTypes: viewAppAttrTypes}
	appVals := []attr.Value{}
	for _, app := range view.Applications {
		obj, d := flattenAppMap(app.ToMap())
		diags.Append(d...)
		appVals = append(appVals, obj)
	}
	appList, d := types.ListValue(appObjType, appVals)
	diags.Append(d...)
	m.Application = appList

	// Teams (skip the auto-created Admin team when create_admin_team is set).
	createAdmin := m.CreateAdminTeam.ValueBool()
	teamObjType := types.ObjectType{AttrTypes: viewTeamAttrTypes}
	teamVals := []attr.Value{}
	for _, team := range view.Teams {
		if ifaceStr(team.Name) == "Admin" && createAdmin {
			continue
		}
		obj, d := flattenTeamMap(ctx, team.ToMap())
		diags.Append(d...)
		teamVals = append(teamVals, obj)
	}
	teamList, d := types.ListValue(teamObjType, teamVals)
	diags.Append(d...)
	m.Team = teamList

	return diags
}

func (r *viewResource) readTopLevel(m *viewModel) diag.Diagnostics {
	var diags diag.Diagnostics
	view, err := api.ReadViewTopLevel(m.ID.ValueString(), r.cfg)
	if err != nil {
		diags.AddError("Error reading view", err.Error())
		return diags
	}
	m.Name = types.StringValue(view.Name)
	if view.Description != "" || !m.Description.IsNull() {
		m.Description = types.StringValue(view.Description)
	}
	m.Status = types.StringValue(view.Status)
	return diags
}

// blockNameRank maps each nested block's "name" attribute to its index within
// the given list, so API results can be reordered to match the configured
// (model) block order. A null/unknown list yields an empty map.
func blockNameRank(_ context.Context, list types.List) map[string]int {
	rank := map[string]int{}
	if list.IsNull() || list.IsUnknown() {
		return rank
	}
	for i, elem := range list.Elements() {
		obj, ok := elem.(types.Object)
		if !ok {
			continue
		}
		nameAttr, ok := obj.Attributes()["name"]
		if !ok {
			continue
		}
		nameStr, ok := nameAttr.(types.String)
		if !ok || nameStr.IsNull() || nameStr.IsUnknown() {
			continue
		}
		if _, seen := rank[nameStr.ValueString()]; !seen {
			rank[nameStr.ValueString()] = i
		}
	}
	return rank
}

// teamChildRanks extracts, for each team (keyed by team name) in the model, the
// configured order of its users (keyed by user_id) and app instances (keyed by
// name). Used to reorder API results to match the configured block order.
func teamChildRanks(_ context.Context, teams types.List) (map[string]map[string]int, map[string]map[string]int, map[string]map[string]int) {
	users := map[string]map[string]int{}
	insts := map[string]map[string]int{}
	perms := map[string]map[string]int{}
	if teams.IsNull() || teams.IsUnknown() {
		return users, insts, perms
	}
	for _, elem := range teams.Elements() {
		team, ok := elem.(types.Object)
		if !ok {
			continue
		}
		attrs := team.Attributes()
		nameAttr, _ := attrs["name"].(types.String)
		if nameAttr.IsNull() || nameAttr.IsUnknown() {
			continue
		}
		teamName := nameAttr.ValueString()

		if userList, ok := attrs["user"].(types.List); ok {
			users[teamName] = childRankByAttr(userList, "user_id")
		}
		if instList, ok := attrs["app_instance"].(types.List); ok {
			insts[teamName] = childRankByAttr(instList, "name")
		}
		if permList, ok := attrs["permissions"].(types.List); ok {
			perms[teamName] = stringListRank(permList)
		}
	}
	return users, insts, perms
}

// stringListRank maps each string element of a list to its index.
func stringListRank(list types.List) map[string]int {
	rank := map[string]int{}
	if list.IsNull() || list.IsUnknown() {
		return rank
	}
	for i, elem := range list.Elements() {
		s, ok := elem.(types.String)
		if !ok || s.IsNull() || s.IsUnknown() {
			continue
		}
		if _, seen := rank[s.ValueString()]; !seen {
			rank[s.ValueString()] = i
		}
	}
	return rank
}

// childRankByAttr maps the given string attribute of each object in list to its
// index within the list.
func childRankByAttr(list types.List, attrName string) map[string]int {
	rank := map[string]int{}
	if list.IsNull() || list.IsUnknown() {
		return rank
	}
	for i, elem := range list.Elements() {
		obj, ok := elem.(types.Object)
		if !ok {
			continue
		}
		v, ok := obj.Attributes()[attrName].(types.String)
		if !ok || v.IsNull() || v.IsUnknown() {
			continue
		}
		if _, seen := rank[v.ValueString()]; !seen {
			rank[v.ValueString()] = i
		}
	}
	return rank
}

// desiredTeamScopes returns the complete outgoing scope set for every planned
// team. The schema defaults omitted scoped_teams attributes to an empty set.
func desiredTeamScopes(ctx context.Context, teams types.List) (map[string][]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	scopes := make(map[string][]string)
	if teams.IsNull() || teams.IsUnknown() {
		return scopes, diags
	}

	nameCounts := make(map[string]int)
	type configuredTeam struct {
		name    string
		targets types.Set
	}
	configured := make([]configuredTeam, 0, len(teams.Elements()))

	for _, elem := range teams.Elements() {
		team, ok := elem.(types.Object)
		if !ok {
			continue
		}
		attrs := team.Attributes()
		name, ok := attrs["name"].(types.String)
		if !ok || name.IsNull() || name.IsUnknown() {
			continue
		}
		teamName := name.ValueString()
		nameCounts[teamName]++

		targets, ok := attrs["scoped_teams"].(types.Set)
		if !ok || targets.IsNull() {
			targets = types.SetValueMust(types.StringType, []attr.Value{})
		} else if targets.IsUnknown() {
			diags.AddError(
				"Unknown scoped teams",
				fmt.Sprintf("Team %q has an unknown scoped_teams value during apply.", teamName),
			)
			continue
		}
		configured = append(configured, configuredTeam{name: teamName, targets: targets})
	}

	for _, team := range configured {
		if nameCounts[team.name] != 1 {
			diags.AddError(
				"Ambiguous scoped team source",
				fmt.Sprintf("Team %q cannot manage scoped_teams because its name is not unique within the view.", team.name),
			)
			continue
		}

		targets, d := toStringSet(ctx, team.targets)
		diags.Append(d...)
		for _, target := range targets {
			switch {
			case target == team.name:
				diags.AddError(
					"Invalid scoped team target",
					fmt.Sprintf("Team %q cannot scope its permissions onto itself.", team.name),
				)
			case nameCounts[target] == 0:
				diags.AddError(
					"Unknown scoped team target",
					fmt.Sprintf("Team %q references scoped team %q, but no configured sibling team has that name.", team.name, target),
				)
			case nameCounts[target] > 1:
				diags.AddError(
					"Ambiguous scoped team target",
					fmt.Sprintf("Team %q references scoped team %q, but multiple configured sibling teams have that name.", team.name, target),
				)
			}
		}
		scopes[team.name] = targets
	}

	return scopes, diags
}

// nameLess orders two block names by their position in rank (configured order
// first); names absent from rank sort after ranked names, alphabetically.
func nameLess(a, b string, rank map[string]int) bool {
	ra, aok := rank[a]
	rb, bok := rank[b]
	switch {
	case aok && bok:
		return ra < rb
	case aok:
		return true
	case bok:
		return false
	default:
		return a < b
	}
}

// ------------ Conversion helpers (framework <-> native maps) ------------

// expandApps converts the application block list into the []interface{} of
// map[string]interface{} shape consumed by structs.AppInfoFromMap.
func expandApps(ctx context.Context, list types.List) ([]interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := []interface{}{}
	if list.IsNull() || list.IsUnknown() {
		return out, diags
	}

	var apps []struct {
		AppID            types.String `tfsdk:"app_id"`
		Name             types.String `tfsdk:"name"`
		URL              types.String `tfsdk:"url"`
		Icon             types.String `tfsdk:"icon"`
		Embeddable       types.String `tfsdk:"embeddable"`
		LoadInBackground types.String `tfsdk:"load_in_background"`
		AppTemplateID    types.String `tfsdk:"app_template_id"`
		VID              types.String `tfsdk:"v_id"`
	}
	diags = list.ElementsAs(ctx, &apps, false)
	if diags.HasError() {
		return out, diags
	}

	for _, a := range apps {
		out = append(out, map[string]interface{}{
			"app_id":             a.AppID.ValueString(),
			"name":               a.Name.ValueString(),
			"url":                a.URL.ValueString(),
			"icon":               a.Icon.ValueString(),
			"embeddable":         a.Embeddable.ValueString(),
			"load_in_background": a.LoadInBackground.ValueString(),
			"app_template_id":    a.AppTemplateID.ValueString(),
			"v_id":               a.VID.ValueString(),
		})
	}
	return out, diags
}

// expandTeams converts the team block list into the []interface{} of
// map[string]interface{} shape consumed by structs.TeamInfoFromMap.
func expandTeams(ctx context.Context, list types.List) ([]interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := []interface{}{}
	if list.IsNull() || list.IsUnknown() {
		return out, diags
	}

	var teams []struct {
		TeamID      types.String `tfsdk:"team_id"`
		Name        types.String `tfsdk:"name"`
		Role        types.String `tfsdk:"role"`
		Permissions types.List   `tfsdk:"permissions"`
		ScopedTeams types.Set    `tfsdk:"scoped_teams"`
		AppInstance types.List   `tfsdk:"app_instance"`
		User        types.List   `tfsdk:"user"`
	}
	diags = list.ElementsAs(ctx, &teams, false)
	if diags.HasError() {
		return out, diags
	}

	for _, t := range teams {
		perms, d := toStringSlice(ctx, t.Permissions)
		diags.Append(d...)
		permIface := make([]interface{}, len(perms))
		for i, p := range perms {
			permIface[i] = p
		}
		scopedTeams, d := toStringSet(ctx, t.ScopedTeams)
		diags.Append(d...)
		scopedIface := make([]interface{}, len(scopedTeams))
		for i, scopedTeam := range scopedTeams {
			scopedIface[i] = scopedTeam
		}

		users := []interface{}{}
		if !t.User.IsNull() && !t.User.IsUnknown() {
			var us []struct {
				UserID types.String `tfsdk:"user_id"`
				Role   types.String `tfsdk:"role"`
			}
			d := t.User.ElementsAs(ctx, &us, false)
			diags.Append(d...)
			for _, u := range us {
				users = append(users, map[string]interface{}{
					"user_id": u.UserID.ValueString(),
					"role":    u.Role.ValueString(),
				})
			}
		}

		instances := []interface{}{}
		if !t.AppInstance.IsNull() && !t.AppInstance.IsUnknown() {
			var is []struct {
				Name         types.String  `tfsdk:"name"`
				DisplayOrder types.Float64 `tfsdk:"display_order"`
				ID           types.String  `tfsdk:"id"`
			}
			d := t.AppInstance.ElementsAs(ctx, &is, false)
			diags.Append(d...)
			for _, inst := range is {
				instances = append(instances, map[string]interface{}{
					"name":          inst.Name.ValueString(),
					"display_order": inst.DisplayOrder.ValueFloat64(),
					"id":            inst.ID.ValueString(),
				})
			}
		}

		out = append(out, map[string]interface{}{
			"team_id":      t.TeamID.ValueString(),
			"name":         t.Name.ValueString(),
			"role":         t.Role.ValueString(),
			"permissions":  permIface,
			"scoped_teams": scopedIface,
			"user":         users,
			"app_instance": instances,
		})
	}
	return out, diags
}

// flattenAppMap converts an AppInfo.ToMap() result into a framework object.
func flattenAppMap(m map[string]interface{}) (attr.Value, diag.Diagnostics) {
	return types.ObjectValue(viewAppAttrTypes, map[string]attr.Value{
		"app_id":             types.StringValue(ifaceStr(m["app_id"])),
		"name":               types.StringValue(ifaceStr(m["name"])),
		"url":                types.StringValue(ifaceStr(m["url"])),
		"icon":               types.StringValue(ifaceStr(m["icon"])),
		"embeddable":         types.StringValue(ifaceStr(m["embeddable"])),
		"load_in_background": types.StringValue(ifaceStr(m["load_in_background"])),
		"app_template_id":    types.StringValue(ifaceStr(m["app_template_id"])),
		"v_id":               types.StringValue(ifaceStr(m["v_id"])),
	})
}

// flattenTeamMap converts a TeamInfo.ToMap() result into a framework object.
func flattenTeamMap(ctx context.Context, m map[string]interface{}) (attr.Value, diag.Diagnostics) {
	var diags diag.Diagnostics

	// permissions
	perms := []string{}
	if p, ok := m["permissions"].([]string); ok {
		perms = p
	}
	permList, d := types.ListValueFrom(ctx, types.StringType, perms)
	diags.Append(d...)

	scopedTeams := []string{}
	if s, ok := m["scoped_teams"].([]string); ok {
		scopedTeams = append(scopedTeams, s...)
	}
	scopedTeamSet, d := types.SetValueFrom(ctx, types.StringType, scopedTeams)
	diags.Append(d...)

	// users
	userObjType := types.ObjectType{AttrTypes: viewUserAttrTypes}
	userVals := []attr.Value{}
	if users, ok := m["user"].([]map[string]interface{}); ok {
		for _, u := range users {
			obj, d := types.ObjectValue(viewUserAttrTypes, map[string]attr.Value{
				"user_id": types.StringValue(ifaceStr(u["user_id"])),
				"role":    types.StringValue(ifaceStr(u["role"])),
			})
			diags.Append(d...)
			userVals = append(userVals, obj)
		}
	}
	userList, d := types.ListValue(userObjType, userVals)
	diags.Append(d...)

	// app instances
	instObjType := types.ObjectType{AttrTypes: viewAppInstanceAttrTypes}
	instVals := []attr.Value{}
	if insts, ok := m["app_instance"].([]map[string]interface{}); ok {
		for _, inst := range insts {
			var order float64
			if o, ok := inst["display_order"].(float64); ok {
				order = o
			}
			obj, d := types.ObjectValue(viewAppInstanceAttrTypes, map[string]attr.Value{
				"name":          types.StringValue(ifaceStr(inst["name"])),
				"display_order": types.Float64Value(order),
				"id":            types.StringValue(ifaceStr(inst["id"])),
			})
			diags.Append(d...)
			instVals = append(instVals, obj)
		}
	}
	instList, d := types.ListValue(instObjType, instVals)
	diags.Append(d...)

	obj, d := types.ObjectValue(viewTeamAttrTypes, map[string]attr.Value{
		"team_id":      types.StringValue(ifaceStr(m["team_id"])),
		"name":         types.StringValue(ifaceStr(m["name"])),
		"role":         types.StringValue(ifaceStr(m["role"])),
		"permissions":  permList,
		"scoped_teams": scopedTeamSet,
		"app_instance": instList,
		"user":         userList,
	})
	diags.Append(d...)
	return obj, diags
}

// ifaceStr coerces an interface{} that may be nil or a string into a string.
func ifaceStr(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// ------------ Create/update functions for nested resources ------------
// These retain the SDKv1 diff algorithms verbatim, refactored to operate on
// native []interface{} slices (rather than *schema.ResourceData) and to leave
// state reconciliation to the resource's read() method.

// createApps creates the apps specified in the configuration. The server-
// assigned application IDs are written back into the app_id field of each entry
// in apps so that createTeams can resolve app_instance parents to real IDs.
func createApps(viewID string, m map[string]string, apps []interface{}) error {
	appStructs := new([]*structs.AppInfo)
	for _, app := range apps {
		asMap := util.As[map[string]interface{}](app)

		// Generate a uuid if one is not provided.
		if asMap["app_id"] == "" {
			asMap["app_id"] = uuid.New().String()
			log.Printf("! app_id = %v", asMap["app_id"])
		}
		curr := structs.AppInfoFromMap(asMap)
		curr.ViewID = viewID
		*appStructs = append(*appStructs, curr)
	}

	if err := api.CreateApps(appStructs, m, viewID); err != nil {
		return err
	}

	// api.CreateApps overwrites each struct's ID with the server-assigned GUID.
	// Propagate those back into the shared apps slice (built 1:1 and in order)
	// so the app_instance parent lookup in createTeams uses real IDs.
	for i, app := range apps {
		util.As[map[string]interface{}](app)["app_id"] = (*appStructs)[i].ID
	}

	return nil
}

// createTeams creates the teams specified in the configuration.
func createTeams(viewID string, m map[string]string, teams []interface{}, applications []interface{}) error {
	teamStructs := new([]*structs.TeamInfo)
	for _, team := range teams {
		asMap := util.As[map[string]interface{}](team)
		curr := structs.TeamInfoFromMap(asMap)
		*teamStructs = append(*teamStructs, curr)
	}

	appStructs := new([]*structs.AppInfo)
	for _, app := range applications {
		asMap := util.As[map[string]interface{}](app)
		*appStructs = append(*appStructs, structs.AppInfoFromMap(asMap))
	}

	// Find the GUID mapping to the app name provided in each app instance.
	for i, team := range *teamStructs {
		for j, app := range team.AppInstances {
			for _, parent := range *appStructs {
				if parent.Name == app.Name {
					app.Parent = parent.ID
					(*teamStructs)[i].AppInstances[j] = app
				}
			}
		}
	}

	if err := api.CreateTeams(teamStructs, viewID, m); err != nil {
		return err
	}

	return api.AddPermissionsToTeam(teamStructs, m)
}

// updateApps reconciles the applications within a view.
func updateApps(viewID string, m map[string]string, old, current []interface{}) error {
	toDelete := new([]string)
	toUpdate := new([]*structs.AppInfo)
	toCreate := new([]*structs.AppInfo)

	for _, app := range old {
		oldMap := util.As[map[string]interface{}](app)
		value := util.As[string](oldMap["app_id"])
		if !util.PairInList(current, "app_id", value) {
			*toDelete = append(*toDelete, value)
		} else {
			for _, curr := range current {
				currMap := util.As[map[string]interface{}](curr)
				if util.As[string](currMap["app_id"]) == value && !reflect.DeepEqual(oldMap, currMap) {
					info := structs.AppInfoFromMap(currMap)
					*toUpdate = append(*toUpdate, info)
				}
			}
		}
	}
	for _, app := range current {
		currMap := util.As[map[string]interface{}](app)
		value := util.As[string](currMap["app_id"])
		if !util.PairInList(old, "app_id", value) {
			info := structs.AppInfoFromMap(currMap)
			info.ID = uuid.New().String()
			info.ViewID = viewID
			*toCreate = append(*toCreate, info)
		}
	}

	if err := api.DeleteApps(toDelete, m); err != nil {
		return err
	}
	if err := api.UpdateApps(toUpdate, m); err != nil {
		return err
	}
	return api.CreateApps(toCreate, m, viewID)
}

// updateTeams reconciles the teams within a view.
func updateTeams(m map[string]string, viewID string, old, current, applications []interface{}) error {
	toDelete := new([]string)
	toUpdate := new([]*structs.TeamInfo)
	toCreate := new([]*structs.TeamInfo)

	// The old versions of the teams we updated, for efficient user/instance diffs.
	oldUpdated := new([]*structs.TeamInfo)

	for _, team := range old {
		oldMap := util.As[map[string]interface{}](team)
		value := util.As[string](oldMap["team_id"])
		if !util.PairInList(current, "team_id", value) {
			*toDelete = append(*toDelete, value)
		} else {
			for _, curr := range current {
				currMap := util.As[map[string]interface{}](curr)
				if util.As[string](currMap["team_id"]) == value && !reflect.DeepEqual(oldMap, currMap) {
					info := structs.TeamInfoFromMap(currMap)
					*toUpdate = append(*toUpdate, info)
					*oldUpdated = append(*oldUpdated, structs.TeamInfoFromMap(oldMap))
				}
			}
		}
	}

	for _, team := range current {
		currMap := util.As[map[string]interface{}](team)
		value := util.As[string](currMap["team_id"])
		if !util.PairInList(old, "team_id", value) {
			info := structs.TeamInfoFromMap(currMap)
			*toCreate = append(*toCreate, info)
		}
	}

	// Map team IDs to the permissions to add to/remove from those teams.
	permsToRemove := make(map[string][]string)
	permsToAdd := make(map[string][]string)

	toRemoveCurr := new([]string)
	toAddCurr := new([]string)
	for i, oldTeam := range *oldUpdated {
		*toRemoveCurr = (*toRemoveCurr)[:0]
		*toAddCurr = (*toAddCurr)[:0]

		oldPerms := oldTeam.Permissions
		currPerms := (*toUpdate)[i].Permissions

		for _, oldPerm := range oldPerms {
			if !util.StrSliceContains(&currPerms, oldPerm) {
				*toRemoveCurr = append(*toRemoveCurr, oldPerm)
			}
		}
		permsToRemove[util.As[string](oldTeam.ID)] = *toRemoveCurr

		for _, currPerm := range currPerms {
			if !util.StrSliceContains(&oldPerms, currPerm) {
				*toAddCurr = append(*toAddCurr, currPerm)
			}
		}
		permsToAdd[util.As[string](oldTeam.ID)] = *toAddCurr
	}

	if err := api.UpdateTeamPermissions(permsToAdd, permsToRemove, m); err != nil {
		return err
	}

	appStructs := new([]*structs.AppInfo)
	for _, app := range applications {
		asMap := util.As[map[string]interface{}](app)
		*appStructs = append(*appStructs, structs.AppInfoFromMap(asMap))
	}

	for i, team := range *toCreate {
		for j, app := range team.AppInstances {
			for _, parent := range *appStructs {
				if parent.Name == app.Name {
					app.Parent = parent.ID
					(*toCreate)[i].AppInstances[j] = app
				}
			}
		}
	}

	if err := api.DeleteTeams(toDelete, m); err != nil {
		return err
	}
	if err := api.UpdateTeams(toUpdate, m); err != nil {
		return err
	}
	if err := api.CreateTeams(toCreate, viewID, m); err != nil {
		return err
	}
	if err := api.AddPermissionsToTeam(toCreate, m); err != nil {
		return err
	}

	if err := updateUsers(oldUpdated, toUpdate, m, viewID); err != nil {
		return err
	}

	return updateInstances(oldUpdated, toUpdate, applications, m)
}

// updateUsers reconciles the users within the updated teams.
func updateUsers(oldUpdated, toUpdate *[]*structs.TeamInfo, m map[string]string, viewID string) error {
	removedUsers := make(map[string][]string)

	oldUsers := new([]structs.UserInfo)
	currUsers := new([]structs.UserInfo)
	toAdd := new([]string)

	toChangeRole := make(map[string][]structs.UserInfo)
	usersWithChangedRole := new([]structs.UserInfo)

	for i, oldTeam := range *oldUpdated {
		*oldUsers = (*oldUsers)[:0]
		*oldUsers = oldTeam.Users

		currTeam := (*toUpdate)[i]
		*currUsers = (*currUsers)[:0]
		*currUsers = currTeam.Users

		*usersWithChangedRole = (*usersWithChangedRole)[:0]
		if len(*currUsers) == 0 {
			for _, user := range *oldUsers {
				removedUsers[util.As[string](currTeam.ID)] = append(removedUsers[util.As[string](currTeam.ID)], user.ID)
			}
		} else {
			for _, oldUser := range *oldUsers {
				if !structs.UserHasID(*currUsers, oldUser.ID) {
					removedUsers[util.As[string](currTeam.ID)] = append(removedUsers[util.As[string](currTeam.ID)], oldUser.ID)
				}
			}
			toChangeRole[util.As[string](oldTeam.ID)] = *usersWithChangedRole
		}

		*toAdd = (*toAdd)[:0]
		for _, currUser := range *currUsers {
			if !structs.UserHasID(*oldUsers, currUser.ID) {
				*toAdd = append(*toAdd, currUser.ID)
			} else {
				var old structs.UserInfo
				found := false
				for _, oldUser := range *oldUsers {
					if oldUser.ID == currUser.ID {
						old = oldUser
						found = true
					}
				}
				if found && old.Role != currUser.Role {
					if err := api.SetUserRole(util.As[string](oldTeam.ID), viewID, currUser, m); err != nil {
						return err
					}
				}
			}
		}

		if err := api.AddUsersToTeam(toAdd, util.As[string](currTeam.ID), m); err != nil {
			return err
		}
	}

	return api.RemoveUsers(removedUsers, m)
}

// updateInstances reconciles the application instances within the updated teams.
func updateInstances(old, current *[]*structs.TeamInfo, apps []interface{}, m map[string]string) error {
	deleted := new([]string)

	oldInstances := new([]structs.AppInstance)
	currInstances := new([]structs.AppInstance)

	for i, oldTeam := range *old {
		*oldInstances = (*oldInstances)[:0]
		*oldInstances = oldTeam.AppInstances

		currTeam := (*current)[i]
		*currInstances = (*currInstances)[:0]
		*currInstances = currTeam.AppInstances

		if len(*currInstances) == 0 {
			for _, inst := range *oldInstances {
				*deleted = append(*deleted, inst.ID)
			}
		} else {
			for _, oldInst := range *oldInstances {
				if !structs.InstanceHasID(currInstances, oldInst.ID) {
					*deleted = append(*deleted, oldInst.ID)
				}
			}
		}

		for _, currInst := range *currInstances {
			if !structs.InstanceHasID(oldInstances, currInst.ID) {
				for _, app := range apps {
					asMap := util.As[map[string]interface{}](app)
					if asMap["name"] == currInst.Name {
						if _, err := api.AddApplication(util.As[string](asMap["app_id"]), util.As[string](oldTeam.ID), currInst.DisplayOrder, m); err != nil {
							return err
						}
					}
				}
			} else {
				var old structs.AppInstance
				found := false
				for _, oldInst := range *oldInstances {
					if oldInst.ID == currInst.ID {
						old = oldInst
						found = true
					}
				}
				if found && (old.Name != currInst.Name || old.DisplayOrder != currInst.DisplayOrder) {
					// Resolve the parent application GUID by name; the plan-derived
					// instance does not carry Parent, and UpdateAppInstance sends it
					// as applicationId (an empty value yields a 500).
					for _, app := range apps {
						asMap := util.As[map[string]interface{}](app)
						if asMap["name"] == currInst.Name {
							currInst.Parent = util.As[string](asMap["app_id"])
						}
					}
					if err := api.UpdateAppInstance(currInst, util.As[string](oldTeam.ID), m); err != nil {
						return err
					}
				}
			}
		}
	}

	return api.DeleteAppInstances(deleted, m)
}
