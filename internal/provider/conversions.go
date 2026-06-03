// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// httpURLRegex validates that a string is an http:// or https:// URL,
// replacing the SDKv1 validation.IsURLWithHTTPorHTTPS helper.
var httpURLRegex = regexp.MustCompile(`^https?://`)

// boolStaticTrue returns a bool default of true, equivalent to the SDKv1
// Default: true on a TypeBool attribute.
func boolStaticTrue() defaults.Bool {
	return booldefault.StaticBool(true)
}

// computedStringWithUseState builds a Computed-only string attribute for an
// API-assigned identifier within a nested block (app_id, v_id, team_id,
// app_instance id). It carries the prior-state value forward when present, and
// otherwise plans as unknown.
//
// The unknownIfNull modifier is required in addition to UseStateForUnknown:
// when a new element is added to a ListNestedBlock on update, there is no prior
// state to carry, and the framework would otherwise plan the computed id as
// null. The API then assigns a real id, producing a "provider produced
// inconsistent result" error. Forcing unknown lets the API fill it.
func computedStringWithUseState() schema.StringAttribute {
	return schema.StringAttribute{
		Computed: true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
			unknownIfNull{},
		},
	}
}

// unknownIfNull is a plan modifier that marks a computed attribute as unknown
// when its planned value would otherwise be null. Used for API-assigned ids in
// nested blocks where a newly added element has no prior state to carry.
type unknownIfNull struct{}

func (unknownIfNull) Description(_ context.Context) string {
	return "Marks the value unknown when it would otherwise plan as null."
}

func (m unknownIfNull) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (unknownIfNull) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if resp.PlanValue.IsNull() {
		resp.PlanValue = types.StringUnknown()
	}
}

// toStringSlice converts a framework types.List of strings into a []string.
// Null and unknown lists yield an empty (non-nil) slice so the value is safe to
// hand to the api layer.
func toStringSlice(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := []string{}
	if list.IsNull() || list.IsUnknown() {
		return out, diags
	}
	diags = list.ElementsAs(ctx, &out, false)
	return out, diags
}
