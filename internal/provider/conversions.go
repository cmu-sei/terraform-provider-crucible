// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider

import (
	"context"
	"regexp"
	"sort"

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

// unknownSetIfNull is the collection equivalent of unknownIfNull. A newly
// added ListNestedBlock has no prior state, so an omitted Optional+Computed set
// otherwise plans as null even though the API read returns an empty set.
type unknownSetIfNull struct{}

func (unknownSetIfNull) Description(_ context.Context) string {
	return "Marks the set unknown when it would otherwise plan as null."
}

func (m unknownSetIfNull) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (unknownSetIfNull) PlanModifySet(ctx context.Context, _ planmodifier.SetRequest, resp *planmodifier.SetResponse) {
	if resp.PlanValue.IsNull() {
		resp.PlanValue = types.SetUnknown(resp.PlanValue.ElementType(ctx))
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

// toStringSet converts a framework set of strings into a stable []string.
func toStringSet(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := []string{}
	if set.IsNull() || set.IsUnknown() {
		return out, diags
	}
	diags = set.ElementsAs(ctx, &out, false)
	sort.Strings(out)
	return out, diags
}

// orderTeamIDs returns a resource's team ids in a stable order for state: ids
// that were already present in prior (config/state) order come first in that
// order, followed by any ids the API reports that weren't there before, sorted
// for determinism. The Crucible APIs return team ids in an arbitrary order;
// keeping the configured order avoids "inconsistent result after apply" on a
// Required/ordered team_ids list, while still surfacing out-of-band team
// membership changes on refresh. Shared by the VM and view-network resources.
func orderTeamIDs(prior, fromAPI []string) []string {
	apiSet := make(map[string]struct{}, len(fromAPI))
	for _, id := range fromAPI {
		apiSet[id] = struct{}{}
	}

	out := make([]string, 0, len(fromAPI))
	seen := make(map[string]struct{}, len(fromAPI))
	for _, id := range prior {
		if _, ok := apiSet[id]; ok {
			out = append(out, id)
			seen[id] = struct{}{}
		}
	}

	extra := make([]string, 0)
	for _, id := range fromAPI {
		if _, ok := seen[id]; !ok {
			extra = append(extra, id)
		}
	}
	sort.Strings(extra)
	return append(out, extra...)
}
