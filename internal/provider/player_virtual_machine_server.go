// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/cmu-sei/terraform-provider-crucible/internal/api"
	"github.com/cmu-sei/terraform-provider-crucible/internal/structs"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &virtualMachineResource{}
	_ resource.ResourceWithConfigure   = &virtualMachineResource{}
	_ resource.ResourceWithImportState = &virtualMachineResource{}
)

// virtualMachineResource is the resource implementation for
// crucible_player_virtual_machine.
type virtualMachineResource struct {
	cfg map[string]string
}

type vmModel struct {
	ID         types.String `tfsdk:"id"`
	VMID       types.String `tfsdk:"vm_id"`
	URL        types.String `tfsdk:"url"`
	DefaultURL types.Bool   `tfsdk:"default_url"`
	Name       types.String `tfsdk:"name"`
	TeamIDs    types.List   `tfsdk:"team_ids"`
	UserID     types.String `tfsdk:"user_id"`
	Embeddable types.Bool   `tfsdk:"embeddable"`
	Connection types.List   `tfsdk:"console_connection_info"`
	Proxmox    types.List   `tfsdk:"proxmox_vm_info"`
}

// Object attribute types for the nested blocks.
var consoleConnectionAttrTypes = map[string]attr.Type{
	"hostname": types.StringType,
	"port":     types.StringType,
	"protocol": types.StringType,
	"username": types.StringType,
	"password": types.StringType,
}

var proxmoxAttrTypes = map[string]attr.Type{
	"id":   types.StringType,
	"node": types.StringType,
	"type": types.StringType,
}

// NewVirtualMachineResource is a helper to instantiate the resource.
func NewVirtualMachineResource() resource.Resource {
	return &virtualMachineResource{}
}

func (r *virtualMachineResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_player_virtual_machine"
}

func (r *virtualMachineResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *virtualMachineResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vm_id": schema.StringAttribute{
				// Optional+Computed: when omitted, a UUID is generated and kept
				// in state (the old DiffSuppress suppressed empty new values).
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"url": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(httpURLRegex, "must be a valid URL beginning with http:// or https://"),
				},
				// When default_url is true the API computes the URL, so an empty
				// configured value should not produce a diff.
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"default_url": schema.BoolAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"team_ids": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},
			"user_id": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"embeddable": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
		},
		Blocks: map[string]schema.Block{
			"console_connection_info": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"hostname": schema.StringAttribute{Optional: true},
						"port":     schema.StringAttribute{Optional: true},
						"protocol": schema.StringAttribute{Optional: true},
						"username": schema.StringAttribute{Optional: true},
						"password": schema.StringAttribute{Optional: true},
					},
				},
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
			"proxmox_vm_info": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Optional: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
								proxmoxIDPlanModifier{},
							},
						},
						"node": schema.StringAttribute{Optional: true},
						"type": schema.StringAttribute{
							Optional: true,
							Computed: true,
							Default:  stringdefault.StaticString("QEMU"),
							Validators: []validator.String{
								stringvalidator.OneOf("QEMU", "LXC"),
							},
						},
					},
				},
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
		},
	}
}

func (r *virtualMachineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan vmModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamIDs, diags := toStringSlice(ctx, plan.TeamIDs)
	resp.Diagnostics.Append(diags...)

	connection, diags := expandConnection(ctx, plan.Connection)
	resp.Diagnostics.Append(diags...)

	proxmox, diags := expandProxmox(ctx, plan.Proxmox)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate a VM ID when one was not supplied.
	vmID := plan.VMID.ValueString()
	if vmID == "" {
		vmID = uuid.NewString()
	}

	reqBody := &structs.VMInfo{
		ID:         vmID,
		URL:        plan.URL.ValueString(),
		Name:       plan.Name.ValueString(),
		TeamIDs:    teamIDs,
		UserID:     userIDValue(plan.UserID),
		Embeddable: plan.Embeddable.ValueBool(),
		Connection: connection,
		Proxmox:    proxmox,
	}

	if err := api.CreateVM(reqBody, r.cfg); err != nil {
		resp.Diagnostics.AddError("Error creating virtual machine", err.Error())
		return
	}

	plan.ID = types.StringValue(vmID)

	resp.Diagnostics.Append(r.read(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *virtualMachineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state vmModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, err := api.VMExists(state.ID.ValueString(), r.cfg)
	if err != nil {
		resp.Diagnostics.AddError("Error checking virtual machine existence", err.Error())
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

func (r *virtualMachineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state vmModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()

	// Reconcile team membership when team_ids changed.
	if !plan.TeamIDs.Equal(state.TeamIDs) {
		oldTeams, diags := toStringSlice(ctx, state.TeamIDs)
		resp.Diagnostics.Append(diags...)
		newTeams, diags := toStringSlice(ctx, plan.TeamIDs)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		toRemove := stringsDifference(oldTeams, newTeams)
		toAdd := stringsDifference(newTeams, oldTeams)

		if err := api.AddVMToTeams(&toAdd, id, r.cfg); err != nil {
			resp.Diagnostics.AddError("Error adding virtual machine to teams", err.Error())
			return
		}
		if err := api.RemoveVMFromTeams(&toRemove, id, r.cfg); err != nil {
			resp.Diagnostics.AddError("Error removing virtual machine from teams", err.Error())
			return
		}
	}

	connection, diags := expandConnection(ctx, plan.Connection)
	resp.Diagnostics.Append(diags...)

	proxmox, diags := expandProxmox(ctx, plan.Proxmox)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If default_url is set, the URL is computed by the API, so send empty.
	url := plan.URL.ValueString()
	if state.DefaultURL.ValueBool() {
		url = ""
	}

	// The ID and TeamIDs parameters are ignored by the update API.
	reqBody := &structs.VMInfo{
		ID:         "",
		URL:        url,
		Name:       plan.Name.ValueString(),
		TeamIDs:    []string{""},
		UserID:     userIDValue(plan.UserID),
		Embeddable: plan.Embeddable.ValueBool(),
		Connection: connection,
		Proxmox:    proxmox,
	}

	if err := api.UpdateVM(reqBody, id, r.cfg); err != nil {
		resp.Diagnostics.AddError("Error updating virtual machine", err.Error())
		return
	}

	resp.Diagnostics.Append(r.read(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *virtualMachineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state vmModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	exists, err := api.VMExists(id, r.cfg)
	if err != nil {
		resp.Diagnostics.AddError("Error checking virtual machine existence", err.Error())
		return
	}
	if !exists {
		return
	}

	if err := api.DeleteVM(id, r.cfg); err != nil {
		resp.Diagnostics.AddError("Error deleting virtual machine", err.Error())
	}
}

func (r *virtualMachineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("vm_id"), req.ID)...)
}

// read refreshes the model from the API.
func (r *virtualMachineResource) read(ctx context.Context, m *vmModel) diag.Diagnostics {
	var diags diag.Diagnostics

	info, err := api.GetVMInfo(m.ID.ValueString(), r.cfg)
	if err != nil {
		diags.AddError("Error reading virtual machine", err.Error())
		return diags
	}

	// Sort team IDs for consistent state.
	sort.Strings(info.TeamIDs)

	m.ID = types.StringValue(info.ID)
	m.VMID = types.StringValue(info.ID)
	m.URL = types.StringValue(info.URL)
	m.DefaultURL = types.BoolValue(info.DefaultURL)
	m.Name = types.StringValue(info.Name)
	m.Embeddable = types.BoolValue(info.Embeddable)

	teamIDs, d := types.ListValueFrom(ctx, types.StringType, info.TeamIDs)
	diags.Append(d...)
	m.TeamIDs = teamIDs

	if uid, ok := info.UserID.(string); ok && uid != "" {
		m.UserID = types.StringValue(uid)
	} else {
		m.UserID = types.StringNull()
	}

	connection, d := flattenConnection(ctx, info.Connection)
	diags.Append(d...)
	m.Connection = connection

	proxmox, d := flattenProxmox(ctx, info.Proxmox)
	diags.Append(d...)
	m.Proxmox = proxmox

	return diags
}

// expandConnection converts the console_connection_info block list into a
// ConsoleConnection struct, or nil when the block is absent.
func expandConnection(ctx context.Context, list types.List) (*structs.ConsoleConnection, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() || len(list.Elements()) == 0 {
		return nil, diags
	}

	var elems []struct {
		Hostname types.String `tfsdk:"hostname"`
		Port     types.String `tfsdk:"port"`
		Protocol types.String `tfsdk:"protocol"`
		Username types.String `tfsdk:"username"`
		Password types.String `tfsdk:"password"`
	}
	diags = list.ElementsAs(ctx, &elems, false)
	if diags.HasError() || len(elems) == 0 {
		return nil, diags
	}

	e := elems[0]
	return &structs.ConsoleConnection{
		Hostname: e.Hostname.ValueString(),
		Port:     e.Port.ValueString(),
		Protocol: e.Protocol.ValueString(),
		Username: e.Username.ValueString(),
		Password: e.Password.ValueString(),
	}, diags
}

// flattenConnection converts a ConsoleConnection struct into a block list.
func flattenConnection(ctx context.Context, conn *structs.ConsoleConnection) (types.List, diag.Diagnostics) {
	objType := types.ObjectType{AttrTypes: consoleConnectionAttrTypes}
	if conn == nil {
		return types.ListNull(objType), nil
	}

	obj, diags := types.ObjectValue(consoleConnectionAttrTypes, map[string]attr.Value{
		"hostname": types.StringValue(conn.Hostname),
		"port":     types.StringValue(conn.Port),
		"protocol": types.StringValue(conn.Protocol),
		"username": types.StringValue(conn.Username),
		"password": types.StringValue(conn.Password),
	})
	if diags.HasError() {
		return types.ListNull(objType), diags
	}

	list, d := types.ListValue(objType, []attr.Value{obj})
	diags.Append(d...)
	return list, diags
}

// expandProxmox converts the proxmox_vm_info block list into a ProxmoxInfo
// struct, reusing the existing map-based parser, or nil when absent.
func expandProxmox(ctx context.Context, list types.List) (*structs.ProxmoxInfo, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() || len(list.Elements()) == 0 {
		return nil, diags
	}

	var elems []struct {
		ID   types.String `tfsdk:"id"`
		Node types.String `tfsdk:"node"`
		Type types.String `tfsdk:"type"`
	}
	diags = list.ElementsAs(ctx, &elems, false)
	if diags.HasError() || len(elems) == 0 {
		return nil, diags
	}

	e := elems[0]
	return structs.ProxmoxInfoFromMap(map[string]interface{}{
		"id":   e.ID.ValueString(),
		"node": e.Node.ValueString(),
		"type": e.Type.ValueString(),
	}), diags
}

// flattenProxmox converts a ProxmoxInfo struct into a block list.
func flattenProxmox(ctx context.Context, proxmox *structs.ProxmoxInfo) (types.List, diag.Diagnostics) {
	objType := types.ObjectType{AttrTypes: proxmoxAttrTypes}
	if proxmox == nil {
		return types.ListNull(objType), nil
	}

	asMap := proxmox.ToMap()
	obj, diags := types.ObjectValue(proxmoxAttrTypes, map[string]attr.Value{
		"id":   types.StringValue(asMap["id"].(string)),
		"node": types.StringValue(asMap["node"].(string)),
		"type": types.StringValue(asMap["type"].(string)),
	})
	if diags.HasError() {
		return types.ListNull(objType), diags
	}

	list, d := types.ListValue(objType, []attr.Value{obj})
	diags.Append(d...)
	return list, diags
}

// userIDValue mirrors the SDKv1 behavior of sending a nil user_id when empty.
func userIDValue(v types.String) interface{} {
	if v.IsNull() || v.IsUnknown() || v.ValueString() == "" {
		return nil
	}
	return v.ValueString()
}

// stringsDifference returns elements present in a but not in b.
func stringsDifference(a, b []string) []string {
	set := make(map[string]struct{}, len(b))
	for _, s := range b {
		set[s] = struct{}{}
	}
	out := []string{}
	for _, s := range a {
		if _, ok := set[s]; !ok {
			out = append(out, s)
		}
	}
	return out
}

// proxmoxIDPlanModifier replicates the SDKv1 DiffSuppressFunc on the proxmox
// id: the API stores a bare integer id, but the proxmox provider supplies it as
// "{node}/{type}/{id}". When the configured value is the composite form whose
// trailing token matches the stored id, no change is planned.
type proxmoxIDPlanModifier struct{}

func (m proxmoxIDPlanModifier) Description(_ context.Context) string {
	return "Suppresses diffs when the configured composite proxmox id matches the stored id."
}

func (m proxmoxIDPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m proxmoxIDPlanModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}
	old := req.StateValue.ValueString()
	newVal := req.PlanValue.ValueString()
	tokens := strings.Split(newVal, "/")
	if len(tokens) == 3 && tokens[2] == old {
		resp.PlanValue = req.StateValue
	}
}
