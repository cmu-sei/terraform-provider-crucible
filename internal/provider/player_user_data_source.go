// Copyright 2026 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider

import (
	"context"
	"fmt"

	"github.com/cmu-sei/terraform-provider-crucible/internal/api"
	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource                     = &playerUserDataSource{}
	_ datasource.DataSourceWithConfigure        = &playerUserDataSource{}
	_ datasource.DataSourceWithConfigValidators = &playerUserDataSource{}
)

type playerUserDataSource struct {
	cfg map[string]string
}

type playerUserDataSourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	IdentityAttributes types.Map    `tfsdk:"identity_attributes"`
	CaseInsensitive    types.Bool   `tfsdk:"case_insensitive"`
	RoleID             types.String `tfsdk:"role_id"`
	RoleName           types.String `tfsdk:"role_name"`
	Attributes         types.Map    `tfsdk:"attributes"`
}

func NewPlayerUserDataSource() datasource.DataSource {
	return &playerUserDataSource{}
}

func (d *playerUserDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_player_user"
}

func (d *playerUserDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *playerUserDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up one Player user by name, by identity attributes, or both.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "User UUID.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "User name to match. Set by the lookup when omitted.",
				Optional:    true,
				Computed:    true,
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"identity_attributes": schema.MapAttribute{
				Description: "Identity attribute values to match, keyed by attribute key. Every entry must match.",
				Optional:    true,
				ElementType: types.StringType,
				Validators:  []validator.Map{mapvalidator.SizeAtLeast(1)},
			},
			"case_insensitive": schema.BoolAttribute{
				Description: "Match the name and identity attribute values without regard to case. Defaults to false.",
				Optional:    true,
			},
			"role_id": schema.StringAttribute{
				Description: "UUID of the user's system role, or empty when the user has none.",
				Computed:    true,
			},
			"role_name": schema.StringAttribute{
				Description: "Name of the user's system role, or empty when the user has none.",
				Computed:    true,
			},
			"attributes": schema.MapAttribute{
				Description: "Every identity attribute Player holds a value for on the user, keyed by attribute key.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *playerUserDataSource) ConfigValidators(context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.AtLeastOneOf(
			path.MatchRoot("name"),
			path.MatchRoot("identity_attributes"),
		),
	}
}

func (d *playerUserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config playerUserDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	filter := api.PlayerUserFilter{
		Name:            config.Name.ValueString(),
		CaseInsensitive: config.CaseInsensitive.ValueBool(),
	}
	if !config.IdentityAttributes.IsNull() {
		resp.Diagnostics.Append(config.IdentityAttributes.ElementsAs(ctx, &filter.IdentityAttributes, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	user, err := api.FindPlayerUser(filter, d.cfg)
	if err != nil {
		resp.Diagnostics.AddError("Error looking up Player user", err.Error())
		return
	}

	attributes, diags := types.MapValueFrom(ctx, types.StringType, user.IdentityAttributes)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config.ID = types.StringValue(user.ID)
	config.Name = types.StringValue(user.Name)
	config.RoleID = types.StringValue(user.RoleID)
	config.RoleName = types.StringValue(user.RoleName)
	config.Attributes = attributes
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
