// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
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
	_ datasource.DataSource              = &applicationTemplateDataSource{}
	_ datasource.DataSourceWithConfigure = &applicationTemplateDataSource{}
)

// applicationTemplateDataSource is the data source implementation for
// crucible_player_application_template. It looks up an application template by
// name and exposes its id (plus the rest of the template fields), letting
// configs reference templates by name instead of hardcoded UUIDs.
type applicationTemplateDataSource struct {
	cfg map[string]string
}

type applicationTemplateDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	URL              types.String `tfsdk:"url"`
	Icon             types.String `tfsdk:"icon"`
	Embeddable       types.Bool   `tfsdk:"embeddable"`
	LoadInBackground types.Bool   `tfsdk:"load_in_background"`
	CaseInsensitive  types.Bool   `tfsdk:"case_insensitive"`
}

// NewApplicationTemplateDataSource is a helper to instantiate the data source.
func NewApplicationTemplateDataSource() datasource.DataSource {
	return &applicationTemplateDataSource{}
}

func (d *applicationTemplateDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_player_application_template"
}

func (d *applicationTemplateDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	d.cfg = cfg
}

func (d *applicationTemplateDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			// case_insensitive defaults to false: matching is exact by default
			// so existing lookups are not silently broadened. Set it to true to
			// match the name ignoring case.
			"case_insensitive": schema.BoolAttribute{
				Optional: true,
			},
			"url": schema.StringAttribute{
				Computed: true,
			},
			"icon": schema.StringAttribute{
				Computed: true,
			},
			"embeddable": schema.BoolAttribute{
				Computed: true,
			},
			"load_in_background": schema.BoolAttribute{
				Computed: true,
			},
		},
	}
}

func (d *applicationTemplateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config applicationTemplateDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	template, err := api.AppTemplateFindByName(config.Name.ValueString(), config.CaseInsensitive.ValueBool(), d.cfg)
	if err != nil {
		resp.Diagnostics.AddError("Error looking up application template", err.Error())
		return
	}

	if template.Id == nil {
		resp.Diagnostics.AddError("Error looking up application template", "player API returned a template with no id")
		return
	}
	config.ID = types.StringValue(template.Id.String())
	config.Name = types.StringValue(strDeref(template.Name))
	config.URL = types.StringValue(strDeref(template.Url))
	config.Icon = types.StringValue(strDeref(template.Icon))
	config.Embeddable = types.BoolValue(boolDeref(template.Embeddable))
	config.LoadInBackground = types.BoolValue(boolDeref(template.LoadInBackground))

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

// strDeref returns the pointed-to string, or "" if the pointer is nil.
func strDeref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// boolDeref returns the pointed-to bool, or false if the pointer is nil.
func boolDeref(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
