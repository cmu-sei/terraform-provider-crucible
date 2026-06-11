// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package api

import (
	"testing"

	"github.com/cmu-sei/terraform-provider-crucible/internal/playerclient"
)

func TestMatchAppTemplateByName(t *testing.T) {
	tmpl := func(name string) playerclient.ApplicationTemplate {
		n := name
		return playerclient.ApplicationTemplate{Name: &n}
	}
	templates := []playerclient.ApplicationTemplate{
		tmpl("VMs"),
		tmpl("Maps"),
		tmpl("Console"),
		tmpl("Console"), // intentional duplicate
	}

	tests := []struct {
		name            string
		lookup          string
		caseInsensitive bool
		wantName        string
		wantErr         bool
	}{
		{name: "single exact match", lookup: "VMs", wantName: "VMs"},
		{name: "no match", lookup: "Nope", wantErr: true},
		{name: "duplicate names error", lookup: "Console", wantErr: true},
		{name: "case-sensitive miss", lookup: "vms", wantErr: true},
		{name: "case-insensitive hit", lookup: "vms", caseInsensitive: true, wantName: "VMs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := matchAppTemplateByName(templates, tt.lookup, tt.caseInsensitive)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (result %+v)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got == nil || got.Name == nil || *got.Name != tt.wantName {
				t.Fatalf("got %+v, want name %q", got, tt.wantName)
			}
		})
	}
}
