// Copyright 2026 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const readTestVMID = "6a7ec409-d275-4b31-94d3-a51cb61d2519"

// fakeVMAPI serves a token endpoint and answers GET /api/vms/{id} with status.
func fakeVMAPI(t *testing.T, status int) map[string]string {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"test","token_type":"Bearer","expires_in":3600}`))
	})
	mux.HandleFunc("GET /api/vms/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return map[string]string{
		"vm_api_url":       srv.URL,
		"player_token_url": srv.URL + "/token",
		"client_id":        "test",
		"username":         "test",
		"password":         "test",
	}
}

// vmReadState builds a stored state holding one VM.
func vmReadState(t *testing.T, ctx context.Context, r *virtualMachineResource) tfsdk.State {
	t.Helper()
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	state := tfsdk.State{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
	}
	model := vmModel{
		ID:         types.StringValue(readTestVMID),
		VMID:       types.StringValue(readTestVMID),
		URL:        types.StringValue("https://example.com/console"),
		DefaultURL: types.BoolValue(false),
		Name:       types.StringValue("vm"),
		TeamIDs:    types.ListNull(types.StringType),
		UserID:     types.StringNull(),
		Embeddable: types.BoolValue(true),
		Connection: types.ListNull(types.ObjectType{AttrTypes: consoleConnectionAttrTypes}),
		Proxmox:    types.ListNull(types.ObjectType{AttrTypes: proxmoxAttrTypes}),
	}
	if diags := state.Set(ctx, &model); diags.HasError() {
		t.Fatalf("building state: %v", diags)
	}
	return state
}

func runVMRead(t *testing.T, status int) (tfsdk.State, resource.ReadResponse) {
	t.Helper()
	ctx := context.Background()
	r := &virtualMachineResource{cfg: fakeVMAPI(t, status)}
	state := vmReadState(t, ctx, r)

	resp := resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, &resp)
	return state, resp
}

// A 403 keeps the stored state with a warning, so a destroy still reaches
// Delete, which needs only a manage permission.
func TestVMReadForbiddenKeepsState(t *testing.T) {
	stored, resp := runVMRead(t, http.StatusForbidden)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read returned errors on a 403: %v", resp.Diagnostics)
	}
	if got := resp.Diagnostics.WarningsCount(); got != 1 {
		t.Errorf("Read returned %d warnings on a 403, want 1", got)
	}
	if !resp.State.Raw.Equal(stored.Raw) {
		t.Errorf("Read changed the stored state on a 403")
	}
}

func TestVMReadNotFoundRemovesState(t *testing.T) {
	_, resp := runVMRead(t, http.StatusNotFound)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read returned errors on a 404: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Errorf("Read kept the VM in state on a 404")
	}
}

func TestVMReadOtherStatusFails(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusInternalServerError} {
		_, resp := runVMRead(t, status)

		if !resp.Diagnostics.HasError() {
			t.Errorf("Read returned no error on a %d", status)
		}
	}
}
