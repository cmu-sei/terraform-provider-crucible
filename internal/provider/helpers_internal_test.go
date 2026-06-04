// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

// This file is declared `package provider` (an internal test package) rather than
// `package provider_test` so it can exercise the package's unexported, pure helpers
// (ordering/rank helpers and the unknownIfNull plan modifier). These tests are fully
// offline: they build framework values in memory and need no live Crucible stack or
// TF_ACC.
package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// --- test fixtures / builders for framework values ---

// userObj builds a user object ({user_id, role}) for use in a user list.
func userObj(t *testing.T, userID, role string) types.Object {
	t.Helper()
	o, d := types.ObjectValue(viewUserAttrTypes, map[string]attr.Value{
		"user_id": types.StringValue(userID),
		"role":    types.StringValue(role),
	})
	if d.HasError() {
		t.Fatalf("building user object: %v", d)
	}
	return o
}

// userList builds a list of user objects.
func userList(t *testing.T, ids ...string) types.List {
	t.Helper()
	vals := make([]attr.Value, 0, len(ids))
	for _, id := range ids {
		vals = append(vals, userObj(t, id, ""))
	}
	l, d := types.ListValue(types.ObjectType{AttrTypes: viewUserAttrTypes}, vals)
	if d.HasError() {
		t.Fatalf("building user list: %v", d)
	}
	return l
}

// instObj builds an app_instance object ({name, display_order, id}).
func instObj(t *testing.T, name string) types.Object {
	t.Helper()
	o, d := types.ObjectValue(viewAppInstanceAttrTypes, map[string]attr.Value{
		"name":          types.StringValue(name),
		"display_order": types.Float64Value(0),
		"id":            types.StringValue(""),
	})
	if d.HasError() {
		t.Fatalf("building instance object: %v", d)
	}
	return o
}

// instList builds a list of app_instance objects.
func instList(t *testing.T, names ...string) types.List {
	t.Helper()
	vals := make([]attr.Value, 0, len(names))
	for _, n := range names {
		vals = append(vals, instObj(t, n))
	}
	l, d := types.ListValue(types.ObjectType{AttrTypes: viewAppInstanceAttrTypes}, vals)
	if d.HasError() {
		t.Fatalf("building instance list: %v", d)
	}
	return l
}

// stringList builds a list of plain strings.
func stringList(t *testing.T, ss ...string) types.List {
	t.Helper()
	vals := make([]attr.Value, 0, len(ss))
	for _, s := range ss {
		vals = append(vals, types.StringValue(s))
	}
	l, d := types.ListValue(types.StringType, vals)
	if d.HasError() {
		t.Fatalf("building string list: %v", d)
	}
	return l
}

// namedBlockList builds a list of objects that have a "name" attribute, mirroring the
// shape blockNameRank consumes. Uses the user attr types but only the relevant key.
func namedBlockList(t *testing.T, attrTypes map[string]attr.Type, names ...string) types.List {
	t.Helper()
	vals := make([]attr.Value, 0, len(names))
	for _, n := range names {
		m := map[string]attr.Value{}
		for k, at := range attrTypes {
			switch {
			case k == "name":
				m[k] = types.StringValue(n)
			case at == types.StringType:
				m[k] = types.StringValue("")
			default:
				m[k] = newNullValue(at)
			}
		}
		o, d := types.ObjectValue(attrTypes, m)
		if d.HasError() {
			t.Fatalf("building named object: %v", d)
		}
		vals = append(vals, o)
	}
	l, d := types.ListValue(types.ObjectType{AttrTypes: attrTypes}, vals)
	if d.HasError() {
		t.Fatalf("building named block list: %v", d)
	}
	return l
}

// newNullValue returns a typed null for non-string attribute types used in fixtures.
func newNullValue(at attr.Type) attr.Value {
	switch t := at.(type) {
	case types.ListType:
		return types.ListNull(t.ElemType)
	case basetypes.Float64Type:
		return types.Float64Null()
	default:
		return types.StringNull()
	}
}

// --- unknownIfNull plan modifier ---

func TestUnknownIfNull_PlanModifyString(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		plan        types.String
		wantUnknown bool
		wantValue   string
	}{
		{name: "null becomes unknown", plan: types.StringNull(), wantUnknown: true},
		{name: "known is unchanged", plan: types.StringValue("View Member"), wantUnknown: false, wantValue: "View Member"},
		{name: "empty string is left as known", plan: types.StringValue(""), wantUnknown: false, wantValue: ""},
		{name: "unknown stays unknown", plan: types.StringUnknown(), wantUnknown: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := planmodifier.StringRequest{PlanValue: tc.plan}
			resp := &planmodifier.StringResponse{PlanValue: tc.plan}

			unknownIfNull{}.PlanModifyString(ctx, req, resp)

			if got := resp.PlanValue.IsUnknown(); got != tc.wantUnknown {
				t.Fatalf("IsUnknown() = %v, want %v (resp=%v)", got, tc.wantUnknown, resp.PlanValue)
			}
			if !tc.wantUnknown {
				if resp.PlanValue.ValueString() != tc.wantValue {
					t.Fatalf("ValueString() = %q, want %q", resp.PlanValue.ValueString(), tc.wantValue)
				}
			}
		})
	}
}

func TestUnknownIfNull_Descriptions(t *testing.T) {
	ctx := context.Background()
	m := unknownIfNull{}
	want := "Marks the value unknown when it would otherwise plan as null."
	if got := m.Description(ctx); got != want {
		t.Errorf("Description() = %q, want %q", got, want)
	}
	if got := m.MarkdownDescription(ctx); got != want {
		t.Errorf("MarkdownDescription() = %q, want %q", got, want)
	}
}

// --- nameLess ---

func TestNameLess(t *testing.T) {
	rank := map[string]int{"foo": 0, "bar": 1}

	tests := []struct {
		name string
		a, b string
		want bool
	}{
		{"both ranked, a first", "foo", "bar", true},
		{"both ranked, b first", "bar", "foo", false},
		{"only a ranked sorts first", "foo", "zzz", true},
		{"only b ranked sorts first", "zzz", "foo", false},
		{"neither ranked falls back alphabetical (a<b)", "alpha", "beta", true},
		{"neither ranked falls back alphabetical (a>b)", "beta", "alpha", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := nameLess(tc.a, tc.b, rank); got != tc.want {
				t.Errorf("nameLess(%q,%q) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

// --- blockNameRank ---

func TestBlockNameRank(t *testing.T) {
	ctx := context.Background()

	t.Run("maps names to indices", func(t *testing.T) {
		list := namedBlockList(t, viewTeamAttrTypes, "foo", "bar", "baz")
		got := blockNameRank(ctx, list)
		want := map[string]int{"foo": 0, "bar": 1, "baz": 2}
		assertIntMap(t, got, want)
	})

	t.Run("duplicate names: first occurrence wins", func(t *testing.T) {
		list := namedBlockList(t, viewTeamAttrTypes, "foo", "foo")
		got := blockNameRank(ctx, list)
		assertIntMap(t, got, map[string]int{"foo": 0})
	})

	t.Run("null list yields empty map", func(t *testing.T) {
		got := blockNameRank(ctx, types.ListNull(types.ObjectType{AttrTypes: viewTeamAttrTypes}))
		assertIntMap(t, got, map[string]int{})
	})

	t.Run("unknown list yields empty map", func(t *testing.T) {
		got := blockNameRank(ctx, types.ListUnknown(types.ObjectType{AttrTypes: viewTeamAttrTypes}))
		assertIntMap(t, got, map[string]int{})
	})
}

// --- childRankByAttr ---

func TestChildRankByAttr(t *testing.T) {
	t.Run("ranks users by user_id", func(t *testing.T) {
		got := childRankByAttr(userList(t, "u1", "u2", "u3"), "user_id")
		assertIntMap(t, got, map[string]int{"u1": 0, "u2": 1, "u3": 2})
	})

	t.Run("ranks instances by name", func(t *testing.T) {
		got := childRankByAttr(instList(t, "a", "b"), "name")
		assertIntMap(t, got, map[string]int{"a": 0, "b": 1})
	})

	t.Run("duplicate attr value: first wins", func(t *testing.T) {
		got := childRankByAttr(userList(t, "dup", "dup"), "user_id")
		assertIntMap(t, got, map[string]int{"dup": 0})
	})

	t.Run("null list yields empty map", func(t *testing.T) {
		got := childRankByAttr(types.ListNull(types.ObjectType{AttrTypes: viewUserAttrTypes}), "user_id")
		assertIntMap(t, got, map[string]int{})
	})
}

// --- stringListRank ---

func TestStringListRank(t *testing.T) {
	t.Run("ranks strings by position", func(t *testing.T) {
		got := stringListRank(stringList(t, "p1", "p2", "p3"))
		assertIntMap(t, got, map[string]int{"p1": 0, "p2": 1, "p3": 2})
	})

	t.Run("duplicate string: first wins", func(t *testing.T) {
		got := stringListRank(stringList(t, "x", "x"))
		assertIntMap(t, got, map[string]int{"x": 0})
	})

	t.Run("null list yields empty map", func(t *testing.T) {
		got := stringListRank(types.ListNull(types.StringType))
		assertIntMap(t, got, map[string]int{})
	})
}

// --- teamChildRanks ---

func TestTeamChildRanks(t *testing.T) {
	ctx := context.Background()

	// Build a single team "t1" with users, app instances, and permissions.
	teamObj, d := types.ObjectValue(viewTeamAttrTypes, map[string]attr.Value{
		"team_id":      types.StringValue(""),
		"name":         types.StringValue("t1"),
		"role":         types.StringValue(""),
		"permissions":  stringList(t, "perm-a", "perm-b"),
		"app_instance": instList(t, "inst-a"),
		"user":         userList(t, "user-a", "user-b"),
	})
	if d.HasError() {
		t.Fatalf("building team object: %v", d)
	}
	teams, d := types.ListValue(types.ObjectType{AttrTypes: viewTeamAttrTypes}, []attr.Value{teamObj})
	if d.HasError() {
		t.Fatalf("building team list: %v", d)
	}

	users, insts, perms := teamChildRanks(ctx, teams)
	assertIntMap(t, users["t1"], map[string]int{"user-a": 0, "user-b": 1})
	assertIntMap(t, insts["t1"], map[string]int{"inst-a": 0})
	assertIntMap(t, perms["t1"], map[string]int{"perm-a": 0, "perm-b": 1})

	t.Run("null list yields empty maps", func(t *testing.T) {
		u, i, p := teamChildRanks(ctx, types.ListNull(types.ObjectType{AttrTypes: viewTeamAttrTypes}))
		if len(u) != 0 || len(i) != 0 || len(p) != 0 {
			t.Fatalf("expected empty maps, got users=%v insts=%v perms=%v", u, i, p)
		}
	})
}

// --- ifaceStr ---

func TestIfaceStr(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		want string
	}{
		{"nil -> empty", nil, ""},
		{"string passthrough", "hello", "hello"},
		{"empty string", "", ""},
		{"bool via Sprintf", true, "true"},
		{"int via Sprintf", 42, "42"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ifaceStr(tc.in); got != tc.want {
				t.Errorf("ifaceStr(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// --- shared assert helper ---

func assertIntMap(t *testing.T, got, want map[string]int) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("map length = %d, want %d (got=%v want=%v)", len(got), len(want), got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("map[%q] = %d, want %d (full got=%v)", k, got[k], v, got)
		}
	}
}
