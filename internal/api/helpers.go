// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package api

// Small pointer/deref helpers shared by the generated-client adapters in this
// package. The oapi-codegen models use pointers for optional fields; these keep
// the conversion code terse. derefStr/derefBool/toUUIDs live in vm_api.go.

func strPtr(s string) *string { return &s }

func boolPtr(b bool) *bool { return &b }

// ifaceStrPtr converts an optional interface{} field (string or nil, as used by
// structs.AppInfo / TeamInfo / UserInfo) into a *string, returning nil for nil
// or empty-string so the field is omitted from the request.
func ifaceStrPtr(v interface{}) *string {
	if s, ok := v.(string); ok && s != "" {
		return &s
	}
	return nil
}

// ifaceBoolPtr converts an optional interface{} field into a *bool. The provider
// stores these as strings ("true"/"false") or bools; nil/empty yields nil.
func ifaceBoolPtr(v interface{}) *bool {
	switch t := v.(type) {
	case bool:
		return &t
	case string:
		if t == "true" {
			b := true
			return &b
		}
		if t == "false" {
			b := false
			return &b
		}
	}
	return nil
}

// strOrNil returns the string value of p, or nil when p is nil/empty — matching
// the interface{} optional-field convention of structs.AppInfo (empty -> nil).
func strOrNil(p *string) interface{} {
	if p != nil && *p != "" {
		return *p
	}
	return nil
}

// boolOrNil returns the bool value of p as a string ("true"/"false") to match
// the structs.AppInfo convention, or nil when p is nil.
func boolOrNil(p *bool) interface{} {
	if p == nil {
		return nil
	}
	if *p {
		return "true"
	}
	return "false"
}

// derefFloat32 returns the float32 value of p, or 0 when nil.
func derefFloat32(p *float32) float32 {
	if p != nil {
		return *p
	}
	return 0
}
