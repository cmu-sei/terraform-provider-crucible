// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package api

// Small pointer/deref helpers shared by the generated-client adapters in this
// package. The oapi-codegen models use pointers for optional fields; these keep
// the conversion code terse. derefStr/derefBool/toUUIDs live in vm_api.go.

func strPtr(s string) *string { return &s }

func boolPtr(b bool) *bool { return &b }
