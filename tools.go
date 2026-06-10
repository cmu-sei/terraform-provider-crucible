// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

//go:build tools

// Package tools pins build-time code generators so their versions are tracked
// in go.mod and reproducible. It is never compiled into the provider binary
// (guarded by the "tools" build tag).
//
// oapi-codegen generates the typed Player VM API client under internal/vmclient.
// Regenerate with `task gen-vm-client`.
//
// gotestsum wraps `go test` with a streaming, per-test formatter so the live
// acceptance suite (`task testacc`) shows progress instead of buffering each
// package's output until it finishes.
package tools

import (
	_ "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen"
	_ "gotest.tools/gotestsum"
)
