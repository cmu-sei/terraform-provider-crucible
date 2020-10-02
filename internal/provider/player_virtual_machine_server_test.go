package provider_test

// import (
// 	"crucible_provider/internal/api"
// 	"crucible_provider/internal/provider"
// 	"crucible_provider/internal/util"
// 	"fmt"
// 	"reflect"
// 	"regexp"
// 	"sort"
// 	"strings"
// 	"testing"

// 	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
// 	"github.com/hashicorp/terraform-plugin-sdk/terraform"
// )

// // Note: I had to roll my own function to check the local state of slice types. Built in ones did not work for some reason.

// // Test case for a normal creation/deployment of a VM. VM fields are set properly, as are API credentials.
// //
// // Execution steps:
// // 1. Terraform will automatically call apply based on what is passed to the Config field.
// // 2. Test to make sure both local and remote states have been updated accordingly.
// // 3. After state is checked, Terraform verifies that calling plan again does not stage any changes.
// // 4. After everything is verified, terraform destroys the resource.
// //
// // Expected behavior:
// // Resource is successfully created and destroyed. No errors should be thrown.
// func TestAccBasicSuccessful(t *testing.T) {
// 	resource.Test(t, resource.TestCase{
// 		Providers: map[string]terraform.ResourceProvider{
// 			"crucible": provider.Provider(),
// 		},
// 		Steps: []resource.TestStep{
// 			{
// 				Config: correctCreds + configVMNormal,
// 				Check: resource.ComposeTestCheckFunc(
// 					// Verify local state
// 					testAccVerifyLocal("crucible_player_virtual_machine.test", "6a7ec409-d275-4b31-94d3-a51cb61d2519", "http://example.com",
// 						"foo", "8694c78c-1c49-421b-8ed8-689b46834878", []string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),

// 					// Verify remote state
// 					testAccRemoteEquals("6a7ec409-d275-4b31-94d3-a51cb61d2519", "http://example.com", "foo", "8694c78c-1c49-421b-8ed8-689b46834878",
// 						[]string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),
// 				),
// 			},
// 		},
// 	})
// }

// // Test case for a misconfigured VM. The ID field of the VM is set improperly but everything else, including credentials,
// // is set correctly.
// //
// // Execution steps
// // 1. Terraform attempts to create the resource. This should fail.
// // 2. Verify that no state was set either locally or remotely
// // 3. Terraform attempts to destroy the resource. This should *not* throw an error.
// //
// // Expected behavior:
// // When trying to create the resource, terraform should report that status code 400 was returned by the API.
// // No state should be saved, locally or remotely. The destroy operation should not error.
// func TestAccBasicFail(t *testing.T) {
// 	resource.Test(t, resource.TestCase{
// 		Providers: map[string]terraform.ResourceProvider{
// 			"crucible": provider.Provider(),
// 		},
// 		Steps: []resource.TestStep{
// 			{
// 				Config:      correctCreds + configVMIncorrectUserID,
// 				ExpectError: regexp.MustCompile("Request returned with status code 400"),
// 				Check: resource.ComposeTestCheckFunc(
// 					// Verify local state
// 					// Checking the team_ids and allowed_networks fields works here, although I suspect that is because
// 					// it *never* thinks those fields are set, not that they have actually been verified as not being set.
// 					resource.TestCheckNoResourceAttr("crucible_player_virtual_machine.test", "vm_id"),
// 					resource.TestCheckNoResourceAttr("crucible_player_virtual_machine.test", "url"),
// 					resource.TestCheckNoResourceAttr("crucible_player_virtual_machine.test", "name"),
// 					resource.TestCheckNoResourceAttr("crucible_player_virtual_machine.test", "user_id"),
// 					resource.TestCheckNoResourceAttr("crucible_player_virtual_machine.test", "team_ids"),
// 					resource.TestCheckNoResourceAttr("crucible_player_virtual_machine.test", "allowed_networks"),
// 					// Verify that remote state is not set, ie that a VM with this id does not exist
// 					testAccRemoteNotSet("33605140-f28f-4722-b161-8540e97e6bab"),
// 				),
// 			},
// 		},
// 	})
// }

// // Test case for a VM that is created and then updated.
// //
// // Execution steps:
// // 1. Terraform creates a VM according to the first Config field
// // 2. Verify local and remote states are set as expected
// // 3. Terraform updates the VM according to a second Config field - name field is changed
// // 4. Verify local and remote states again
// // 5. Terraform destroys the resource.
// //
// // Expected behavior:
// //
// // A VM is created and updated withour error. Both local and remote state match the state defined in the config string
// // both before and after the update is performed
// func TestAccUpdate(t *testing.T) {
// 	resource.Test(t, resource.TestCase{
// 		Providers: map[string]terraform.ResourceProvider{
// 			"crucible": provider.Provider(),
// 		},
// 		Steps: []resource.TestStep{
// 			// The first step is identical to the basic test. This test is technically a superset of that one
// 			// so they could be combined, but for now I like the logical arrangement of splitting them up
// 			{
// 				Config: correctCreds + configVMNormal,
// 				Check: resource.ComposeTestCheckFunc(
// 					// Verify local state
// 					testAccVerifyLocal("crucible_player_virtual_machine.test", "6a7ec409-d275-4b31-94d3-a51cb61d2519", "http://example.com",
// 						"foo", "8694c78c-1c49-421b-8ed8-689b46834878", []string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),

// 					// Verify remote state
// 					testAccRemoteEquals("6a7ec409-d275-4b31-94d3-a51cb61d2519", "http://example.com", "foo", "8694c78c-1c49-421b-8ed8-689b46834878",
// 						[]string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),
// 				),
// 			},
// 			// The next step here is almost the same, except we must check for the updated value of name
// 			{
// 				Config: correctCreds + configVMNormalUpdated,
// 				Check: resource.ComposeTestCheckFunc(
// 					// Verify local state
// 					testAccVerifyLocal("crucible_player_virtual_machine.test", "6a7ec409-d275-4b31-94d3-a51cb61d2519", "http://example.com",
// 						"bar", "8694c78c-1c49-421b-8ed8-689b46834878", []string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),

// 					// Verify remote state
// 					testAccRemoteEquals("6a7ec409-d275-4b31-94d3-a51cb61d2519", "http://example.com", "bar", "8694c78c-1c49-421b-8ed8-689b46834878",
// 						[]string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),
// 				),
// 			},
// 		},
// 	})
// }

// // Test case for the creation of multiple VMs
// //
// // Execution steps:
// //
// // 1. Terraform creates two VMs, according to two different config strings
// // 2. The local and remote state for both of the VMs is verified
// // 3. Terraform destroys both VMs
// //
// // Expected behavior:
// //
// // Both VMs are created without error. Both of their local and remote states match what is defined by the config
// // The VMs can be destroyed without error.
// func TestAccMultipleCreate(t *testing.T) {
// 	resource.Test(t, resource.TestCase{
// 		Providers: map[string]terraform.ResourceProvider{
// 			"crucible": provider.Provider(),
// 		},
// 		Steps: []resource.TestStep{
// 			// Verify states
// 			{
// 				Config: correctCreds + configVMFirst + configVMSecond,
// 				Check: resource.ComposeTestCheckFunc(
// 					// Verify local state for first VM
// 					testAccVerifyLocal("crucible_player_virtual_machine.first", "1d0b5b53-e034-492d-95c6-714379a4f51e", "http://example.com",
// 						"first", "8694c78c-1c49-421b-8ed8-689b46834878", []string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),

// 					// Verify remote state for first VM
// 					testAccRemoteEquals("1d0b5b53-e034-492d-95c6-714379a4f51e", "http://example.com", "first", "8694c78c-1c49-421b-8ed8-689b46834878",
// 						[]string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),

// 					// Verify local state for second VM
// 					testAccVerifyLocal("crucible_player_virtual_machine.second", "3faebb23-d896-410b-9fcb-a17d9d37427d", "http://example.com",
// 						"second", "8694c78c-1c49-421b-8ed8-689b46834878", []string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),

// 					// Verify remote state for second VM
// 					testAccRemoteEquals("3faebb23-d896-410b-9fcb-a17d9d37427d", "http://example.com", "second", "8694c78c-1c49-421b-8ed8-689b46834878",
// 						[]string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),
// 				),
// 			},
// 		},
// 	})
// }

// // Test case for creating and updating multiple VMs.
// //
// // Execution steps:
// // 1. Terraform creates two VMs according to their configs
// // 2. Verify local and remote state
// // 3. Update both VMs with new configs
// // 4. Verify that the changes are reflected in both local and remote states
// // 5. Terraform destroys both VMs
// //
// // Expected behavior:
// // The VMs are created without error and their states are correct. The VMs are updated without error, and the updates
// // are reflected in their states. They are destroyed without error.
// func TestAccMultipleUpdateAll(t *testing.T) {
// 	resource.Test(t, resource.TestCase{
// 		Providers: map[string]terraform.ResourceProvider{
// 			"crucible": provider.Provider(),
// 		},
// 		Steps: []resource.TestStep{
// 			// Verify states before update
// 			{
// 				Config: correctCreds + configVMFirst + configVMSecond,
// 				Check: resource.ComposeTestCheckFunc(
// 					// Verify local state for first VM
// 					testAccVerifyLocal("crucible_player_virtual_machine.first", "1d0b5b53-e034-492d-95c6-714379a4f51e", "http://example.com",
// 						"first", "8694c78c-1c49-421b-8ed8-689b46834878", []string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),

// 					// Verify remote state for first VM
// 					testAccRemoteEquals("1d0b5b53-e034-492d-95c6-714379a4f51e", "http://example.com", "first", "8694c78c-1c49-421b-8ed8-689b46834878",
// 						[]string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),

// 					// Verify local state for second VM
// 					testAccVerifyLocal("crucible_player_virtual_machine.second", "3faebb23-d896-410b-9fcb-a17d9d37427d", "http://example.com",
// 						"second", "8694c78c-1c49-421b-8ed8-689b46834878", []string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),

// 					// Verify remote state for second VM
// 					testAccRemoteEquals("3faebb23-d896-410b-9fcb-a17d9d37427d", "http://example.com", "second", "8694c78c-1c49-421b-8ed8-689b46834878",
// 						[]string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),
// 				),
// 			},
// 			{
// 				// Update both VMs
// 				Config: correctCreds + configVMFirstUpdated + configVMSecondUpdated,
// 				Check: resource.ComposeTestCheckFunc(
// 					// Verify local state for first VM
// 					testAccVerifyLocal("crucible_player_virtual_machine.first", "1d0b5b53-e034-492d-95c6-714379a4f51e", "http://example.com",
// 						"firstUpdated", "8694c78c-1c49-421b-8ed8-689b46834878", []string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),

// 					// Verify remote state for first VM
// 					testAccRemoteEquals("1d0b5b53-e034-492d-95c6-714379a4f51e", "http://example.com", "firstUpdated", "8694c78c-1c49-421b-8ed8-689b46834878",
// 						[]string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),

// 					// Verify local state for second VM
// 					testAccVerifyLocal("crucible_player_virtual_machine.second", "3faebb23-d896-410b-9fcb-a17d9d37427d", "http://example.com",
// 						"secondUpdated", "8694c78c-1c49-421b-8ed8-689b46834878", []string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),

// 					// Verify remote state for second VM
// 					testAccRemoteEquals("3faebb23-d896-410b-9fcb-a17d9d37427d", "http://example.com", "secondUpdated", "8694c78c-1c49-421b-8ed8-689b46834878",
// 						[]string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),
// 				),
// 			},
// 		},
// 	})
// }

// // Test case for creation of multiple VMs, and updating of only one of those
// //
// // Execution steps:
// // 1. Terraform deploys two VMs
// // 2. Verify local and remote state
// // 3. Terraform updates ONE VM with a new config
// // 4. Verify local and remote states
// // 5. Terraform destroys both VMs
// //
// // Expected behavior:
// // The VMs are created and have their states verified without error. After being updated, their states are
// // verified without error. Updating one should not effect the other.
// func TestAccMultipleUpdateSome(t *testing.T) {
// 	resource.Test(t, resource.TestCase{
// 		Providers: map[string]terraform.ResourceProvider{
// 			"crucible": provider.Provider(),
// 		},
// 		Steps: []resource.TestStep{
// 			// Verify states
// 			{
// 				Config: correctCreds + configVMFirst + configVMSecond,
// 				Check: resource.ComposeTestCheckFunc(
// 					// Verify local state for first VM
// 					testAccVerifyLocal("crucible_player_virtual_machine.first", "1d0b5b53-e034-492d-95c6-714379a4f51e", "http://example.com",
// 						"first", "8694c78c-1c49-421b-8ed8-689b46834878", []string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),

// 					// Verify remote state for first VM
// 					testAccRemoteEquals("1d0b5b53-e034-492d-95c6-714379a4f51e", "http://example.com", "first", "8694c78c-1c49-421b-8ed8-689b46834878",
// 						[]string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),

// 					// Verify local state for second VM
// 					testAccVerifyLocal("crucible_player_virtual_machine.second", "3faebb23-d896-410b-9fcb-a17d9d37427d", "http://example.com",
// 						"second", "8694c78c-1c49-421b-8ed8-689b46834878", []string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),

// 					// Verify remote state for second VM
// 					testAccRemoteEquals("3faebb23-d896-410b-9fcb-a17d9d37427d", "http://example.com", "second", "8694c78c-1c49-421b-8ed8-689b46834878",
// 						[]string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),
// 				),
// 			},
// 			{
// 				// Update only the first VM
// 				Config: correctCreds + configVMFirstUpdated + configVMSecond,
// 				Check: resource.ComposeTestCheckFunc(
// 					// Verify local state for first VM
// 					testAccVerifyLocal("crucible_player_virtual_machine.first", "1d0b5b53-e034-492d-95c6-714379a4f51e", "http://example.com",
// 						"firstUpdated", "8694c78c-1c49-421b-8ed8-689b46834878", []string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),

// 					// Verify remote state for first VM
// 					testAccRemoteEquals("1d0b5b53-e034-492d-95c6-714379a4f51e", "http://example.com", "firstUpdated", "8694c78c-1c49-421b-8ed8-689b46834878",
// 						[]string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),

// 					// Verify local state for second VM
// 					testAccVerifyLocal("crucible_player_virtual_machine.second", "3faebb23-d896-410b-9fcb-a17d9d37427d", "http://example.com",
// 						"second", "8694c78c-1c49-421b-8ed8-689b46834878", []string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),

// 					// Verify remote state for second VM
// 					testAccRemoteEquals("3faebb23-d896-410b-9fcb-a17d9d37427d", "http://example.com", "second", "8694c78c-1c49-421b-8ed8-689b46834878",
// 						[]string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),
// 				),
// 			},
// 		},
// 	})
// }

// // Test case for incorrect API credentials
// //
// // Execution steps:
// //
// // 1. Terraform attempts to create a VM. This should fail.
// // 2. Verify that no state, local or remote, was saved
// //
// // Expected behavior:
// // Terraform tries and fails to create a VM. No state is saved
// func TestAccIncorrectCredentials(t *testing.T) {
// 	resource.Test(t, resource.TestCase{
// 		Providers: map[string]terraform.ResourceProvider{
// 			"crucible": provider.Provider(),
// 		},
// 		Steps: []resource.TestStep{
// 			{
// 				Config:      incorrectCreds + configVMIncorrectUserID,
// 				ExpectError: regexp.MustCompile("Request returned with status code 401"),
// 				Check: resource.ComposeTestCheckFunc(
// 					// Verify local state
// 					resource.TestCheckNoResourceAttr("crucible_player_virtual_machine.test", "vm_id"),
// 					resource.TestCheckNoResourceAttr("crucible_player_virtual_machine.test", "url"),
// 					resource.TestCheckNoResourceAttr("crucible_player_virtual_machine.test", "name"),
// 					resource.TestCheckNoResourceAttr("crucible_player_virtual_machine.test", "user_id"),
// 					resource.TestCheckNoResourceAttr("crucible_player_virtual_machine.test", "team_ids"),
// 					resource.TestCheckNoResourceAttr("crucible_player_virtual_machine.test", "allowed_networks"),
// 					// Verify that remote state is not set, ie that a VM with this id does not exist
// 					testAccRemoteNotSet("33605140-f28f-4722-b161-8540e97e6bab"),
// 				),
// 			},
// 		},
// 	})
// }

// // Test case for moving a VM between teams
// //
// // Execution steps:
// // 1. Create a VM that is on multiple teams
// // 2. Remove it from one team
// // 3. Verify state
// // 4. Add it back to that team
// // 5. Verify state
// //
// // Expected behavior:
// // VMs can be added to/removed from teams without error
// func TestAccMoveTeams(t *testing.T) {
// 	resource.Test(t, resource.TestCase{
// 		Providers: map[string]terraform.ResourceProvider{
// 			"crucible": provider.Provider(),
// 		},
// 		Steps: []resource.TestStep{
// 			// VM on two teams
// 			{
// 				Config: configVMMultiTeams,
// 				Check: resource.ComposeTestCheckFunc(
// 					testAccVerifyLocal("crucible_player_virtual_machine.test", "6a7ec409-d275-4b31-94d3-a51cb61d2519", "http://example.com",
// 						"foo", "8694c78c-1c49-421b-8ed8-689b46834878", []string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761", "8efdcbd3-daa5-4cb4-b62b-338fe7bf3351"}),

// 					testAccRemoteEquals("6a7ec409-d275-4b31-94d3-a51cb61d2519", "http://example.com", "foo", "8694c78c-1c49-421b-8ed8-689b46834878",
// 						[]string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761", "8efdcbd3-daa5-4cb4-b62b-338fe7bf3351"}),
// 				),
// 			},
// 			// Removed from one team
// 			{
// 				Config: configVMNormal,
// 				Check: resource.ComposeTestCheckFunc(
// 					testAccVerifyLocal("crucible_player_virtual_machine.test", "6a7ec409-d275-4b31-94d3-a51cb61d2519", "http://example.com",
// 						"foo", "8694c78c-1c49-421b-8ed8-689b46834878", []string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),

// 					testAccRemoteEquals("6a7ec409-d275-4b31-94d3-a51cb61d2519", "http://example.com", "foo", "8694c78c-1c49-421b-8ed8-689b46834878",
// 						[]string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761"}),
// 				),
// 			},
// 			// Back to two teams
// 			{
// 				Config: configVMMultiTeams,
// 				Check: resource.ComposeTestCheckFunc(
// 					testAccVerifyLocal("crucible_player_virtual_machine.test", "6a7ec409-d275-4b31-94d3-a51cb61d2519", "http://example.com",
// 						"foo", "8694c78c-1c49-421b-8ed8-689b46834878", []string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761", "8efdcbd3-daa5-4cb4-b62b-338fe7bf3351"}),

// 					testAccRemoteEquals("6a7ec409-d275-4b31-94d3-a51cb61d2519", "http://example.com", "foo", "8694c78c-1c49-421b-8ed8-689b46834878",
// 						[]string{"c0a1ebb6-f549-43fb-8d79-63fe1c3dd761", "8efdcbd3-daa5-4cb4-b62b-338fe7bf3351"}),
// 				),
// 			},
// 		},
// 	})
// }

// // -------------------- helper functions --------------------

// // Verify that the remote state of the VM with ID *id* matches the parameters passed
// func testAccRemoteEquals(id, url, name, userID string, teamIDs []string) resource.TestCheckFunc {
// 	return func(s *terraform.State) error {
// 		m := getMap()
// 		info, err := api.GetVMInfo(id, m)
// 		if err != nil {
// 			return err
// 		}

// 		if info.ID != id {
// 			return fmt.Errorf("Expected: " + id + " did not match actual: " + info.ID + " for field ID")
// 		}
// 		if info.Name != name {
// 			return fmt.Errorf("Expected: " + name + " did not match actual: " + info.Name + " for field name")
// 		}
// 		if info.URL != url {
// 			return fmt.Errorf("Expected: " + url + " did not match actual: " + info.URL + " for field URL")
// 		}
// 		// If user id is nil, this will panic but user id is not nil in any test case
// 		if info.UserID != userID {
// 			return fmt.Errorf("Expected: " + userID + " did not match actual: " + info.UserID.(string) + " for field userID")
// 		}

// 		// Sort both teamIDs fields so we don't get a false negative due to ordering
// 		sort.Slice(info.TeamIDs, func(i, j int) bool {
// 			return info.TeamIDs[i] < info.TeamIDs[j]
// 		})
// 		sort.Slice(teamIDs, func(i, j int) bool {
// 			return teamIDs[i] < teamIDs[j]
// 		})
// 		if !reflect.DeepEqual(info.TeamIDs, teamIDs) {
// 			return fmt.Errorf("Expected: " + strings.Join(teamIDs, " ") + " did not match actual: " + strings.Join(info.TeamIDs, " ") + " for field Team IDs")
// 		}

// 		return nil
// 	}
// }

// // Verify that remote state is not set. The VM API does not seem to have a concept of partial state, so we will consider
// // state to not be set if the VM pointed to by id does not exist.
// func testAccRemoteNotSet(id string) resource.TestCheckFunc {
// 	return func(s *terraform.State) error {
// 		m := getMap()
// 		exists, err := api.VMExists(id, m)
// 		if err != nil {
// 			return fmt.Errorf("Error when checking remote state for VM " + id)
// 		}
// 		if exists {
// 			return fmt.Errorf("Remote state for VM " + id + " is set.")
// 		}
// 		return nil
// 	}
// }

// // Verify local state. The built-in functions work for string fields, but I had to make a custom function to check
// // that slice fields are correctly set
// func testAccVerifyLocal(vm, id, url, name, userID string, expectedTeamIDs []string) resource.TestCheckFunc {
// 	return resource.ComposeTestCheckFunc(
// 		resource.TestCheckResourceAttr(vm, "vm_id", id),
// 		resource.TestCheckResourceAttr(vm, "name", name),
// 		resource.TestCheckResourceAttr(vm, "url", url),
// 		resource.TestCheckResourceAttr(vm, "user_id", userID),
// 		func(s *terraform.State) error {
// 			var ret error
// 			// Look for the resource we want to check
// 			for i := range s.Modules {
// 				str := findState(vm, s.Modules[i])

// 				if str == "" {
// 					continue
// 				}

// 				check := checkLocalSlices(str, vm, expectedTeamIDs)

// 				if check {
// 					ret = nil
// 				} else {
// 					ret = fmt.Errorf("Slice types set improperly in local state")
// 				}
// 			}
// 			return ret
// 		},
// 	)
// }

// // Verify that the local state of slice fields is set properly
// func checkLocalSlices(state, vm string, expectedTeamIDs []string) bool {
// 	state = strings.ReplaceAll(state, vm+":", "")
// 	lines := strings.Split(state, "\n")

// 	teamMatches := 0
// 	currTeam := 0

// 	// fmt.Printf("Expected:%+v\n\n\n", expectedTeamIDs)
// 	// fmt.Printf("Actual:%+v\n\n\n", lines)

// 	for _, line := range lines {
// 		line = strings.Trim(line, " ")
// 		split := strings.Split(line, " = ")
// 		if split[0] == fmt.Sprintf("%s%d", "team_ids.", currTeam) {
// 			if util.StrSliceContains(&expectedTeamIDs, split[1]) {
// 				teamMatches++
// 			}
// 			currTeam++
// 		}
// 	}

// 	return teamMatches == len(expectedTeamIDs)
// }

// func findState(vm string, state *terraform.ModuleState) string {
// 	if strings.HasPrefix(state.String(), vm) {
// 		return state.String()
// 	}
// 	return ""
// }
