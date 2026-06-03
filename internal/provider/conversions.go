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

// computedStringWithUseState builds a computed, optional string attribute that
// preserves its prior-state value when the configuration omits it. Used for
// API-assigned identifiers within nested blocks.
func computedStringWithUseState() schema.StringAttribute {
	return schema.StringAttribute{
		Optional: true,
		Computed: true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
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
