// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

// Always-run cleanup + prior-run self-heal for the acceptance tests. The
// terraform-plugin-testing framework runs its own destroy + CheckDestroy at the
// end of resource.Test; these helpers are a safety net that runs AFTER that (via
// t.Cleanup) and a sweep that runs BEFORE a test, so a crashed or
// failed-teardown run never leaves resources dangling on the live stack.
//
// Invariant: cleanup never turns a broken run green. CheckDestroy stays the
// pass/fail signal; registerCleanup additionally FAILS the test if it had to
// delete something (teardown was incomplete). The pre-run sweep only logs — it
// heals a previous run, not the current one.
package provider_test

import (
	"strconv"
	"testing"

	"github.com/cmu-sei/terraform-provider-crucible/internal/api"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// registerCleanup runs after the framework's destroy/CheckDestroy. If the
// resource still exists, teardown was incomplete: delete it AND fail the test.
// A clean teardown leaves nothing to do. A stack-down error is logged, not
// failed, so it never masks the real failure.
func registerCleanup(t *testing.T, label string, exists func() (bool, error), del func() error) {
	t.Helper()
	t.Cleanup(func() {
		ok, err := exists()
		if err != nil {
			t.Logf("cleanup: %s existence check failed (stack down?): %v", label, err)
			return
		}
		if !ok {
			return
		}
		if err := del(); err != nil {
			t.Errorf("cleanup: %s leaked and could not be deleted: %v", label, err)
			return
		}
		t.Errorf("cleanup: %s leaked (Terraform teardown did not complete) — deleted it", label)
	})
}

// sweep deletes a leak from a PRIOR run before the current test runs. It only
// logs (never fails): it is healing a previous run, not asserting anything about
// this one.
func sweep(t *testing.T, label string, exists func() (bool, error), del func() error) {
	t.Helper()
	ok, err := exists()
	if err != nil || !ok {
		return
	}
	if err := del(); err != nil {
		t.Logf("pre-sweep: %s existed and could not be deleted: %v", label, err)
		return
	}
	t.Logf("pre-sweep: cleared leaked %s from a prior run", label)
}

// --- fixed-handle resources (VM, user): delete directly by their known id ---

func cleanupVM(t *testing.T, vmID string) {
	registerCleanup(t, "vm "+vmID,
		func() (bool, error) { return api.VMExists(vmID, getMap()) },
		func() error { return api.DeleteVM(vmID, getMap()) })
}

func sweepVM(t *testing.T, vmID string) {
	sweep(t, "vm "+vmID,
		func() (bool, error) { return api.VMExists(vmID, getMap()) },
		func() error { return api.DeleteVM(vmID, getMap()) })
}

func cleanupUser(t *testing.T, userID string) {
	registerCleanup(t, "user "+userID,
		func() (bool, error) { return api.UserExists(userID, getMap()) },
		func() error { return api.DeleteUser(userID, getMap()) })
}

func sweepUser(t *testing.T, userID string) {
	sweep(t, "user "+userID,
		func() (bool, error) { return api.UserExists(userID, getMap()) },
		func() error { return api.DeleteUser(userID, getMap()) })
}

// --- computed-id resources: clean up an id captured during the run ---

// cleanupView registers cleanup for a view whose id is captured (via captureID)
// into idPtr during the create step. Deleting a view cascades its teams,
// app-instances, and networks.
func cleanupView(t *testing.T, idPtr *string) {
	registerCleanup(t, "view (captured)",
		func() (bool, error) {
			if *idPtr == "" {
				return false, nil
			}
			return api.ViewExists(*idPtr, getMap())
		},
		func() error { return api.DeleteView(*idPtr, getMap()) })
}

func cleanupAppTemplate(t *testing.T, idPtr *string) {
	registerCleanup(t, "app_template (captured)",
		func() (bool, error) {
			if *idPtr == "" {
				return false, nil
			}
			// The Player API returns 200 with an empty body for a missing
			// template (not 404), so AppTemplateExists is always true. Use the
			// read + empty-Name signal the destroy check uses instead.
			tmpl, err := api.AppTemplateRead(*idPtr, getMap())
			if err != nil {
				return false, nil // unreadable => treat as gone
			}
			return tmpl != nil && tmpl.Name != "", nil
		},
		func() error { return api.DeleteAppTemplate(*idPtr, getMap()) })
}

func cleanupViewNetwork(t *testing.T, viewIDPtr, idPtr *string) {
	registerCleanup(t, "view_network (captured)",
		func() (bool, error) {
			if *viewIDPtr == "" || *idPtr == "" {
				return false, nil
			}
			return api.ViewNetworkExists(*viewIDPtr, *idPtr, getMap())
		},
		func() error { return api.DeleteViewNetwork(*viewIDPtr, *idPtr, getMap()) })
}

func cleanupVlan(t *testing.T, idPtr *string) {
	registerCleanup(t, "vlan (captured)",
		func() (bool, error) {
			if *idPtr == "" {
				return false, nil
			}
			// A vlan record persists in the pool after release; only InUse
			// indicates an actual leak (still acquired). DeleteVlan (release) on
			// an already-released vlan would error, so gate on InUse.
			vlan, err := api.ReadVlan(*idPtr, getMap())
			if err != nil {
				return false, nil // unreadable => treat as gone
			}
			return vlan.InUse, nil
		},
		func() error { return api.DeleteVlan(*idPtr, getMap()) })
}

// --- prior-run self-heal for computed-id resources, by known fixed handle ---

// sweepViewByName clears a leaked view (and its cascade) located by its fixed
// configured name.
func sweepViewByName(t *testing.T, name string) {
	sweep(t, "view "+name,
		func() (bool, error) {
			id, err := api.FindViewByName(name, getMap())
			return id != "", err
		},
		func() error {
			id, err := api.FindViewByName(name, getMap())
			if err != nil {
				return err
			}
			if id == "" {
				return nil
			}
			return api.DeleteView(id, getMap())
		})
}

// cleanupViewByName registers post-run cleanup for a view located by its fixed
// configured name. Used for the view tests, whose ids are computed but whose
// names are stable. Deleting the view cascades teams/app-instances/networks.
func cleanupViewByName(t *testing.T, name string) {
	registerCleanup(t, "view "+name,
		func() (bool, error) {
			id, err := api.FindViewByName(name, getMap())
			return id != "", err
		},
		func() error {
			id, err := api.FindViewByName(name, getMap())
			if err != nil || id == "" {
				return err
			}
			return api.DeleteView(id, getMap())
		})
}

// sweepVlanByNumber releases a leaked vlan located by its number in the given
// partition (empty partitionID => default partition).
func sweepVlanByNumber(t *testing.T, number int, partitionID string) {
	sweep(t, vlanLabel(number),
		func() (bool, error) {
			id, err := api.FindVlanByNumber(number, partitionID, getMap())
			return id != "", err
		},
		func() error {
			id, err := api.FindVlanByNumber(number, partitionID, getMap())
			if err != nil {
				return err
			}
			if id == "" {
				return nil
			}
			return api.DeleteVlan(id, getMap())
		})
}

func vlanLabel(number int) string {
	return "vlan #" + strconv.Itoa(number)
}

// captureID returns a TestCheckFunc that records a resource's Terraform state id
// into out, so cleanup can delete it even after state is gone. attrKey selects
// which attribute to capture ("" => the resource's primary id).
func captureID(res, attrKey string, out *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[res]
		if !ok {
			return nil // resource not in state (e.g. step errored) — nothing to capture
		}
		if attrKey == "" {
			*out = rs.Primary.ID
		} else {
			*out = rs.Primary.Attributes[attrKey]
		}
		return nil
	}
}
