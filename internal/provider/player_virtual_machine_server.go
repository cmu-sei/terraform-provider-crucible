// Copyright 2021 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider

import (
	"crucible_provider/internal/api"
	"crucible_provider/internal/structs"
	"crucible_provider/internal/util"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

// Maps the required operations to the functions defined below.
// The map of strings to Schema pointers defines the properties of a resource.
func playerVirtualMachine() *schema.Resource {
	return &schema.Resource{
		Create: playerVirtualMachineCreate,
		Read:   playerVirtualMachineRead,
		Update: playerVirtualMachineUpdate,
		Delete: playerVirtualMachineDelete,

		Schema: map[string]*schema.Schema{
			"vm_id": {
				Type:     schema.TypeString,
				Optional: true,
				// This makes it so, if no id is included, terraform will not try to update this field
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return new == ""
				},
				ForceNew: true,
			},
			"url": {
				Type:     schema.TypeString,
				Optional: true,
				// The API adds extra information on to the end of the url, so consider the url unchanged if it starts
				// with the url in the configuration
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return strings.HasPrefix(old, new) && new != ""
				},
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"team_ids": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"user_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"console_connection_info": {
				Type:     schema.TypeList,
				Optional: true,
				Default:  nil,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"hostname": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"port": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"protocol": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"username": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"password": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

/*
For create and update, we do the necessary operations, then call read to ensure everything worked
These functions should *never* panic or call os.Exit, just return an error if something goes wrong
Rules for updating state (summarized from docs at https://www.terraform.io/docs/extend/writing-custom-providers.html#error-handling-amp-partial-state):

	1. Regardless of whether or not an error is returned from Create, state will be saved if SetID is called, and will
	not be saved if SetID is not called. That is, state is saved if and only if SetID is called. Important: if there is
	an error, state will still be saved if SetID is called.

	2. If the Update function returns with or without an error, the full state is saved. If the ID becomes blank, the
	resource is destroyed (even within an update, though this shouldn't happen except in error scenarios).

	3. If the Destroy function returns without an error, the resource is assumed to be destroyed, and all state is removed.
	If it returns with an error, all prior state is preserved.

	4. If partial mode is enabled when a create or update returns, only the explicitly enabled configuration keys are
	persisted, resulting in a partial state.
*/

/*
get id, URL, name, teamIds, userId, and allowedNetworks via d.Get(). These are the parameters needed in the
API's POST call to create a new VM.
With the data, construct a JSON object and use it to call API
check for error in API response
Set d's properties using the above data.
call read to ensure everything worked properly
*/
func playerVirtualMachineCreate(d *schema.ResourceData, m interface{}) error {
	log.Printf("! In create function")
	if m == nil {
		return fmt.Errorf("error configuring provider")
	}

	// We have to convert []interface{} to []string manually
	tIDs := d.Get("team_ids").([]interface{})
	convertedTeamIDs := util.ToStringSlice(&tIDs)

	var uid interface{}
	if d.Get("user_id").(string) == "" {
		uid = nil
	} else {
		uid = d.Get("user_id")
	}

	var vmID string
	if d.Get("vm_id") == "" {
		vmID = uuid.New().String()
	} else {
		vmID = d.Get("vm_id").(string)
	}

	// Grab the console connection info block if one exists
	connectionGeneric := d.Get("console_connection_info").([]interface{})
	log.Printf("! In create, console connection info = %v", connectionGeneric)
	var connection *structs.ConsoleConnection
	if len(connectionGeneric) > 0 {
		connection = structs.ConnectionFromMap(connectionGeneric[0].(map[string]interface{}))
	} else {
		connection = nil
	}

	reqBody := &structs.VMInfo{
		ID:         vmID,
		URL:        d.Get("url").(string),
		Name:       d.Get("name").(string),
		TeamIDs:    *convertedTeamIDs,
		UserID:     uid,
		Connection: connection,
	}
	log.Printf("! VM to be created with the following fields:\n %+v", reqBody)

	casted := m.(map[string]string)
	log.Printf("! In create function, calling create API wrapper")
	err := api.CreateVM(reqBody, casted)
	if err != nil {
		return err
	}

	// If no errors occurred, set the properties of d. This tells terraform the resource was created
	d.SetId(vmID)
	err = d.Set("url", reqBody.URL)
	if err != nil {
		return err
	}
	err = d.Set("name", reqBody.Name)
	if err != nil {
		return err
	}
	err = d.Set("team_ids", reqBody.TeamIDs)
	if err != nil {
		return err
	}
	err = d.Set("user_id", reqBody.UserID)
	if err != nil {
		return err
	}

	log.Printf("! In create function, calling read function")
	return playerVirtualMachineRead(d, m)
}

/*
Call API to get resource. Arg to API function is the ID.
If VM does not exist, set ID to "" and return nil.
Take the data structure returned by the API and use it to update d using err = d.Set()
if err != nil {
    return err
}
*/
func playerVirtualMachineRead(d *schema.ResourceData, m interface{}) error {
	log.Printf("! In read function")
	if m == nil {
		return fmt.Errorf("error configuring provider")
	}

	id := d.Id()
	casted := m.(map[string]string)
	log.Printf("! In read function, calling vmExists function")
	exists, err := api.VMExists(id, casted)
	if err != nil {
		return err
	}
	if !exists {
		log.Printf("! In read function, VM does not exist")
		d.SetId("")
		return nil
	}

	log.Printf("! In read function, calling read API wrapper")
	info, err := api.GetVMInfo(id, casted)
	if err != nil {
		return err
	}
	log.Printf("! In read, remote state was:\n %+v", info)

	// Team IDs must be sorted alphabetically to prevent unnecessary updates
	sort.Slice(info.TeamIDs, func(i, j int) bool {
		return info.TeamIDs[i] < info.TeamIDs[j]
	})

	d.SetId(info.ID)
	err = d.Set("vm_id", info.ID)
	if err != nil {
		return err
	}
	err = d.Set("url", info.URL)
	if err != nil {
		return err
	}
	err = d.Set("name", info.Name)
	if err != nil {
		return err
	}
	err = d.Set("user_id", info.UserID)
	if err != nil {
		return err
	}
	err = d.Set("team_ids", info.TeamIDs)
	if err != nil {
		return err
	}

	if info.Connection != nil {
		err = d.Set("console_connection_info", []interface{}{info.Connection.ToMap()})
		if err != nil {
			return err
		}
	}

	log.Printf("! Returning from read function without error")
	return nil
}

/*
Read data from .tf file using d.Get() (as in Create)
Here we only need url, name, userId, and allowedNetworks
Update d using this new data with err = d.Set()
if err != nil {
    return err
}
Update the VM using update API function
If successful, return nil else return error
*/
func playerVirtualMachineUpdate(d *schema.ResourceData, m interface{}) error {
	log.Printf("! In update function")
	if m == nil {
		return fmt.Errorf("error configuring provider")
	}

	// Add and remove VM to/from teams as necessary
	if d.HasChange("team_ids") {
		oldGeneric, currGeneric := d.GetChange("team_ids")

		old := oldGeneric.([]interface{})
		curr := currGeneric.([]interface{})

		oldStr := util.ToStringSlice(&old)
		currStr := util.ToStringSlice(&curr)

		// Find the teams this VM should be removed from (in old but not in curr)
		toRemove := new([]string)
		for _, team := range *oldStr {
			if !util.StrSliceContains(currStr, team) {
				*toRemove = append(*toRemove, team)
			}
		}

		// Find the teams this VM should be added to (in curr but not in old)
		toAdd := new([]string)
		for _, team := range *currStr {
			if !util.StrSliceContains(oldStr, team) {
				*toAdd = append(*toAdd, team)
			}
		}

		casted := m.(map[string]string)
		log.Printf("! Teams to remove VM from: %+v", toRemove)
		log.Printf("! Teams to add VM to: %+v", toAdd)

		err := api.RemoveVMFromTeams(toRemove, d.Id(), casted)
		if err != nil {
			return err
		}
		err = api.AddVMToTeams(toAdd, d.Id(), casted)
		if err != nil {
			return err
		}

		log.Printf("! In update, setting team_ids to: %+v", curr)

		// Team IDs must be sorted alphabetically to prevent unnecessary updates
		sort.Slice(curr, func(i, j int) bool {
			return curr[i].(string) < curr[j].(string)
		})

		err = d.Set("team_ids", curr)
		if err != nil {
			return err
		}
	}

	// Update other fields
	var uid interface{}
	if d.Get("user_id").(string) == "" {
		uid = nil
	} else {
		uid = d.Get("user_id")
	}

	connectionGeneric := d.Get("console_connection_info").([]interface{})
	log.Printf("! console connection from data: %v", connectionGeneric)
	log.Printf("! len = %v", len(connectionGeneric))
	var connection *structs.ConsoleConnection
	if len(connectionGeneric) != 0 {
		connection = structs.ConnectionFromMap(connectionGeneric[0].(map[string]interface{}))
	}

	// The ID and TeamIDs parameters will be ignored by the API.
	reqBody := &structs.VMInfo{
		ID:         "",
		URL:        d.Get("url").(string),
		Name:       d.Get("name").(string),
		TeamIDs:    []string{""},
		UserID:     uid,
		Connection: connection,
	}

	casted := m.(map[string]string)
	log.Printf("! In update function, calling update API wrapper")
	err := api.UpdateVM(reqBody, d.Id(), casted)
	if err != nil {
		return err
	}

	// Set the local state to reflect the update
	err = d.Set("url", reqBody.URL)
	if err != nil {
		return err
	}
	err = d.Set("name", reqBody.Name)
	if err != nil {
		return err
	}
	err = d.Set("user_id", reqBody.UserID)
	if err != nil {
		return err
	}

	log.Printf("! Calling read from update")
	return playerVirtualMachineRead(d, m)
}

/*
Check if VM has already done destroyed using get VM by ID API function.
If it's already been destroyed, return nil (no error)
If it still exists, call the API delete function
If delete is successful, return nil
If there is an error with deletion, return an error

d.SetID("") is called implicitly, no need to call it here
*/
func playerVirtualMachineDelete(d *schema.ResourceData, m interface{}) error {
	log.Printf("! In delete function")
	if m == nil {
		return fmt.Errorf("error configuring provider")
	}

	id := d.Id()
	casted := m.(map[string]string)
	log.Printf("! In delete function, calling vmExists")
	exists, err := api.VMExists(id, casted)

	if err != nil {
		return err
	}

	if !exists {
		log.Printf("! In delete function, VM does not exist")
		return nil
	}

	log.Printf("! In delete function, calling delete API wrapper")
	// We can return the result of the function call directly because it is nil on success or some error value on failure
	return api.DeleteVM(id, casted)
}
