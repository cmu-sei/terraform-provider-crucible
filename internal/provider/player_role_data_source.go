// Copyright 2026 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider

import (
	"context"
	"fmt"

	"github.com/cmu-sei/terraform-provider-crucible/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &playerRoleDataSource{}
	_ datasource.DataSourceWithConfigure = &playerRoleDataSource{}
)

type playerRoleDataSource struct {
	cfg      map[string]string
	teamRole bool
}

type playerRoleDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	CaseInsensitive types.Bool   `tfsdk:"case_insensitive"`
	AllPermissions  types.Bool   `tfsdk:"all_permissions"`
	Immutable       types.Bool   `tfsdk:"immutable"`
	Permissions     types.Set    `tfsdk:"permissions"`
}

func NewPlayerRoleDataSource() datasource.DataSource {
	return &playerRoleDataSource{}
}

func NewPlayerTeamRoleDataSource() datasource.DataSource {
	return &playerRoleDataSource{teamRole: true}
}

func (d *playerRoleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	suffix := "_player_role"
	if d.teamRole {
		suffix = "_player_team_role"
	}
	resp.TypeName = req.ProviderTypeName + suffix
}

func (d *playerRoleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	cfg, ok := req.ProviderData.(map[string]string)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Provider Configuration Type", fmt.Sprintf("Expected map[string]string, got: %T.", req.ProviderData))
		return
	}
	d.cfg = cfg
}

func (d *playerRoleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"case_insensitive": schema.BoolAttribute{
				Optional: true,
			},
			"all_permissions": schema.BoolAttribute{
				Computed: true,
			},
			"immutable": schema.BoolAttribute{
				Computed: true,
			},
			"permissions": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *playerRoleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config playerRoleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var (
		role *api.PlayerRole
		err  error
	)
	if d.teamRole {
		role, err = api.FindPlayerTeamRoleByName(config.Name.ValueString(), config.CaseInsensitive.ValueBool(), d.cfg)
	} else {
		role, err = api.FindPlayerRoleByName(config.Name.ValueString(), config.CaseInsensitive.ValueBool(), d.cfg)
	}
	if err != nil {
		resp.Diagnostics.AddError("Error looking up Player role", err.Error())
		return
	}

	config.ID = types.StringValue(role.ID)
	config.Name = types.StringValue(role.Name)
	config.AllPermissions = types.BoolValue(role.AllPermissions)
	config.Immutable = types.BoolValue(role.Immutable)
	config.Permissions = stringsSet(role.PermissionIDs)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
