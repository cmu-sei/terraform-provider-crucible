// Copyright 2026 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
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
