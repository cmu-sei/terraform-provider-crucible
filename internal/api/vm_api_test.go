// Copyright 2026 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package api

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestVMExistsFromStatus(t *testing.T) {
	vmID := uuid.New()

	tests := []struct {
		name       string
		status     int
		wantExists bool
		wantErr    bool
		wantInErr  string
	}{
		{name: "ok", status: http.StatusOK, wantExists: true},
		{name: "not found", status: http.StatusNotFound, wantExists: false},
		{name: "forbidden", status: http.StatusForbidden, wantErr: true, wantInErr: "ViewViewVms or ControlViewVms"},
		{name: "unauthorized", status: http.StatusUnauthorized, wantErr: true, wantInErr: "credentials"},
		{name: "server error", status: http.StatusInternalServerError, wantErr: true, wantInErr: "500"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := vmExistsFromStatus(tt.status, vmID)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("vmExistsFromStatus(%d) error = nil, want an error", tt.status)
				}
				if got := errors.Is(err, ErrVMReadForbidden); got != (tt.status == http.StatusForbidden) {
					t.Errorf("errors.Is(err, ErrVMReadForbidden) = %v for status %d", got, tt.status)
				}
				if !strings.Contains(err.Error(), tt.wantInErr) || !strings.Contains(err.Error(), vmID.String()) {
					t.Errorf("error %q should contain %q and the VM id", err, tt.wantInErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("vmExistsFromStatus(%d) error = %v", tt.status, err)
			}
			if exists != tt.wantExists {
				t.Errorf("vmExistsFromStatus(%d) = %v, want %v", tt.status, exists, tt.wantExists)
			}
		})
	}
}
