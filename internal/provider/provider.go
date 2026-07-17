// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider

import (
	"context"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure crucibleProvider satisfies the provider.Provider interface.
var _ provider.Provider = &crucibleProvider{}

// crucibleProvider is the provider implementation.
type crucibleProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" during acceptance testing.
	version string
}

// crucibleProviderModel maps provider schema data to a Go type.
type crucibleProviderModel struct {
	Username     types.String `tfsdk:"username"`
	Password     types.String `tfsdk:"password"`
	AuthURL      types.String `tfsdk:"auth_url"`
	TokenURL     types.String `tfsdk:"token_url"`
	VMAPIURL     types.String `tfsdk:"vm_api_url"`
	PlayerAPIURL types.String `tfsdk:"player_api_url"`
	CasterAPIURL types.String `tfsdk:"caster_api_url"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	ClientScopes types.List   `tfsdk:"client_scopes"`
}

// New returns a function that creates a new instance of the provider with the
// given version.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &crucibleProvider{
			version: version,
		}
	}
}

// Metadata returns the provider type name.
func (p *crucibleProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "crucible"
	resp.Version = p.version
}

// Schema defines the provider-level configuration schema.
func (p *crucibleProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			// All attributes are Optional because each falls back to a
			// SEI_CRUCIBLE_* environment variable in Configure, preserving the
			// DefaultFunc behavior of the previous SDKv1 implementation.
			"username": schema.StringAttribute{
				Optional: true,
			},
			"password": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
			},
			"auth_url": schema.StringAttribute{
				Optional: true,
			},
			"token_url": schema.StringAttribute{
				Optional: true,
			},
			"vm_api_url": schema.StringAttribute{
				Optional: true,
			},
			"player_api_url": schema.StringAttribute{
				Optional: true,
			},
			"caster_api_url": schema.StringAttribute{
				Optional: true,
			},
			"client_id": schema.StringAttribute{
				Optional: true,
			},
			"client_secret": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
			},
			"client_scopes": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
		},
	}
}

// Configure reads provider configuration, applies SEI_CRUCIBLE_* environment
// variable fallbacks, and builds the map[string]string consumed by the api
// layer. The map is supplied to all resources via resp.ResourceData.
func (p *crucibleProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg crucibleProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve each value: use the configured value when present and non-empty,
	// otherwise fall back to the corresponding environment variable.
	username := resolve(cfg.Username, "SEI_CRUCIBLE_USERNAME")
	password := resolve(cfg.Password, "SEI_CRUCIBLE_PASSWORD")
	authURL := resolve(cfg.AuthURL, "SEI_CRUCIBLE_AUTH_URL")
	tokenURL := resolve(cfg.TokenURL, "SEI_CRUCIBLE_TOK_URL")
	vmAPIURL := resolve(cfg.VMAPIURL, "SEI_CRUCIBLE_VM_API_URL")
	playerAPIURL := resolve(cfg.PlayerAPIURL, "SEI_CRUCIBLE_PLAYER_API_URL")
	casterAPIURL := resolve(cfg.CasterAPIURL, "SEI_CRUCIBLE_CASTER_API_URL")
	clientID := resolve(cfg.ClientID, "SEI_CRUCIBLE_CLIENT_ID")
	clientSecret := resolve(cfg.ClientSecret, "SEI_CRUCIBLE_CLIENT_SECRET")

	// client_scopes is a list; fall back to the comma/space delimited env var.
	var scopes string
	if !cfg.ClientScopes.IsNull() && !cfg.ClientScopes.IsUnknown() {
		var scopesList []string
		resp.Diagnostics.Append(cfg.ClientScopes.ElementsAs(ctx, &scopesList, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		scopes = strings.Join(scopesList, ",")
	} else {
		scopes = os.Getenv("SEI_CRUCIBLE_CLIENT_SCOPES")
	}

	m := map[string]string{
		"username":         username,
		"password":         password,
		"auth_url":         authURL,
		"player_token_url": tokenURL,
		"vm_api_url":       vmAPIURL,
		"player_api_url":   playerAPIURL,
		"caster_api_url":   casterAPIURL,
		"client_id":        clientID,
		"client_secret":    clientSecret,
		"client_scopes":    scopes,
	}

	// Make the configuration map available to resources and data sources.
	resp.ResourceData = m
	resp.DataSourceData = m
}

// resolve returns the configured string when set and non-empty, otherwise the
// value of the named environment variable.
func resolve(val types.String, envVar string) string {
	if !val.IsNull() && !val.IsUnknown() && val.ValueString() != "" {
		return val.ValueString()
	}
	return os.Getenv(envVar)
}

// Resources returns the resource types implemented by the provider.
func (p *crucibleProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewUserResource,
		NewVlanResource,
		NewViewNetworkResource,
		NewApplicationTemplateResource,
		NewVirtualMachineResource,
		NewViewResource,
		NewPlayerApplicationResource,
		NewPlayerTeamResource,
		NewPlayerViewDefaultTeamResource,
		NewPlayerTeamUserResource,
		NewPlayerApplicationInstanceResource,
	}
}

// DataSources returns the data source types implemented by the provider.
func (p *crucibleProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewApplicationTemplateDataSource,
		NewPlayerRoleDataSource,
		NewPlayerTeamRoleDataSource,
	}
}
