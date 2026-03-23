// Copyright 2022 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.

package provider

import (
	"fmt"
	"github.com/cmu-sei/terraform-provider-crucible/internal/api"
	"github.com/cmu-sei/terraform-provider-crucible/internal/structs"
	"github.com/cmu-sei/terraform-provider-crucible/internal/util"
	"log"
	"sort"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func playerViewNetwork() *schema.Resource {
	return &schema.Resource{
		Create: playerViewNetworkCreate,
		Read:   playerViewNetworkRead,
		Update: playerViewNetworkUpdate,
		Delete: playerViewNetworkDelete,

		Schema: map[string]*schema.Schema{
			"view_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"provider_type": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"Unknown", "Vsphere", "Proxmox", "Azure"}, false),
			},
			"provider_instance_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"network_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"team_ids": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func playerViewNetworkCreate(d *schema.ResourceData, m interface{}) error {
	if m == nil {
		return fmt.Errorf("error configuring provider")
	}

	casted := m.(map[string]string)

	tIDs := d.Get("team_ids").([]interface{})
	teamIds := *util.ToStringSlice(&tIDs)

	network := &structs.ViewNetworkInfo{
		ViewID:             d.Get("view_id").(string),
		ProviderType:       d.Get("provider_type").(string),
		ProviderInstanceId: d.Get("provider_instance_id").(string),
		NetworkId:          d.Get("network_id").(string),
		Name:               d.Get("name").(string),
		TeamIds:            teamIds,
	}

	result, err := api.CreateViewNetwork(network, casted)
	if err != nil {
		return err
	}

	d.SetId(result.ID)

	log.Printf("! ViewNetwork created with ID %s", d.Id())
	return playerViewNetworkRead(d, m)
}

func playerViewNetworkRead(d *schema.ResourceData, m interface{}) error {
	id := d.Id()
	casted := m.(map[string]string)
	viewID := d.Get("view_id").(string)

	exists, err := api.ViewNetworkExists(viewID, id, casted)
	if err != nil {
		return err
	}
	if !exists {
		d.SetId("")
		return nil
	}

	network, err := api.GetViewNetwork(viewID, id, casted)
	if err != nil {
		return err
	}

	err = d.Set("view_id", network.ViewID)
	if err != nil {
		return err
	}

	err = d.Set("provider_type", network.ProviderType)
	if err != nil {
		return err
	}

	err = d.Set("provider_instance_id", network.ProviderInstanceId)
	if err != nil {
		return err
	}

	err = d.Set("network_id", network.NetworkId)
	if err != nil {
		return err
	}

	err = d.Set("name", network.Name)
	if err != nil {
		return err
	}

	// Sort team IDs for consistent state
	sort.Strings(network.TeamIds)
	err = d.Set("team_ids", network.TeamIds)
	if err != nil {
		return err
	}

	return nil
}

func playerViewNetworkUpdate(d *schema.ResourceData, m interface{}) error {
	if m == nil {
		return fmt.Errorf("error configuring provider")
	}

	casted := m.(map[string]string)

	tIDs := d.Get("team_ids").([]interface{})
	teamIds := *util.ToStringSlice(&tIDs)

	network := &structs.ViewNetworkInfo{
		ID:                 d.Id(),
		ViewID:             d.Get("view_id").(string),
		ProviderType:       d.Get("provider_type").(string),
		ProviderInstanceId: d.Get("provider_instance_id").(string),
		NetworkId:          d.Get("network_id").(string),
		Name:               d.Get("name").(string),
		TeamIds:            teamIds,
	}

	err := api.UpdateViewNetwork(network, casted)
	if err != nil {
		return err
	}

	return playerViewNetworkRead(d, m)
}

func playerViewNetworkDelete(d *schema.ResourceData, m interface{}) error {
	if m == nil {
		return fmt.Errorf("error configuring provider")
	}

	id := d.Id()
	casted := m.(map[string]string)
	viewID := d.Get("view_id").(string)

	exists, err := api.ViewNetworkExists(viewID, id, casted)
	if err != nil {
		return err
	}

	if !exists {
		return nil
	}

	return api.DeleteViewNetwork(viewID, id, casted)
}
