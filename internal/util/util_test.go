// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

// Offline unit tests for the pure exported helpers in the util package. These need no
// live Crucible stack.
package util_test

import (
	"testing"

	"github.com/cmu-sei/terraform-provider-crucible/internal/util"
)

func TestToStringSlice(t *testing.T) {
	t.Run("converts interface slice", func(t *testing.T) {
		in := []interface{}{"x", "y", "z"}
		got := util.ToStringSlice(&in)
		want := []string{"x", "y", "z"}
		if len(*got) != len(want) {
			t.Fatalf("length = %d, want %d", len(*got), len(want))
		}
		for i := range want {
			if (*got)[i] != want[i] {
				t.Errorf("[%d] = %q, want %q", i, (*got)[i], want[i])
			}
		}
	})

	t.Run("empty slice yields empty result", func(t *testing.T) {
		in := []interface{}{}
		got := util.ToStringSlice(&in)
		if len(*got) != 0 {
			t.Errorf("length = %d, want 0", len(*got))
		}
	})
}

func TestPairInList(t *testing.T) {
	list := []interface{}{
		map[string]interface{}{"name": "foo", "id": "1"},
		map[string]interface{}{"name": "bar", "id": "2"},
	}
	if !util.PairInList(list, "name", "bar") {
		t.Error("expected to find name=bar")
	}
	if util.PairInList(list, "name", "baz") {
		t.Error("did not expect to find name=baz")
	}
	if util.PairInList(list, "missing", "x") {
		t.Error("did not expect to find a missing key")
	}
}

func TestGetApiUrl(t *testing.T) {
	tests := []struct {
		name    string
		m       map[string]string
		urlName string
		want    string
	}{
		{
			name:    "bare host gets /api/ appended",
			m:       map[string]string{"player_api_url": "http://localhost:4300"},
			urlName: "player_api_url",
			want:    "http://localhost:4300/api/",
		},
		{
			name:    "trailing slash is normalized",
			m:       map[string]string{"player_api_url": "http://localhost:4300/"},
			urlName: "player_api_url",
			want:    "http://localhost:4300/api/",
		},
		{
			name:    "existing /api suffix is normalized to /api/",
			m:       map[string]string{"player_api_url": "http://localhost:4300/api"},
			urlName: "player_api_url",
			want:    "http://localhost:4300/api/",
		},
		{
			name:    "missing key yields empty string",
			m:       map[string]string{},
			urlName: "player_api_url",
			want:    "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := util.GetApiUrl(tc.m, tc.urlName); got != tc.want {
				t.Errorf("GetApiUrl() = %q, want %q", got, tc.want)
			}
		})
	}
}
