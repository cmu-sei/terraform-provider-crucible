/*
Crucible
Copyright 2020 Carnegie Mellon University.
NO WARRANTY. THIS CARNEGIE MELLON UNIVERSITY AND SOFTWARE ENGINEERING INSTITUTE MATERIAL IS FURNISHED ON AN "AS-IS" BASIS. CARNEGIE MELLON UNIVERSITY MAKES NO WARRANTIES OF ANY KIND, EITHER EXPRESSED OR IMPLIED, AS TO ANY MATTER INCLUDING, BUT NOT LIMITED TO, WARRANTY OF FITNESS FOR PURPOSE OR MERCHANTABILITY, EXCLUSIVITY, OR RESULTS OBTAINED FROM USE OF THE MATERIAL. CARNEGIE MELLON UNIVERSITY DOES NOT MAKE ANY WARRANTY OF ANY KIND WITH RESPECT TO FREEDOM FROM PATENT, TRADEMARK, OR COPYRIGHT INFRINGEMENT.
Released under a MIT (SEI)-style license, please see license.txt or contact permission@sei.cmu.edu for full terms.
[DISTRIBUTION STATEMENT A] This material has been approved for public release and unlimited distribution.  Please see Copyright notice for non-US Government use and distribution.
Carnegie Mellon(R) and CERT(R) are registered in the U.S. Patent and Trademark Office by Carnegie Mellon University.
DM20-0181
*/

package provider

import (
	"crucible_provider/internal/api"
	"crucible_provider/internal/structs"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func applicationTemplate() *schema.Resource {
	return &schema.Resource{
		Create: applicationTemplateCreate,
		Read:   applicationTemplateRead,
		Update: applicationTemplateUpdate,
		Delete: applicationTemplateDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"url": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"icon": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"embeddable": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"load_in_background": {
				Type:     schema.TypeBool,
				Optional: true,
			},
		},
	}
}

// get properties from d
// call API
// set local state
// call read
func applicationTemplateCreate(d *schema.ResourceData, m interface{}) error {
	if m == nil {
		return fmt.Errorf("Error configuring provider")
	}

	template := &structs.AppTemplate{
		Name:             d.Get("name").(string),
		URL:              d.Get("url").(string),
		Icon:             d.Get("icon").(string),
		Embeddable:       d.Get("embeddable").(bool),
		LoadInBackground: d.Get("load_in_background").(bool),
	}

	log.Printf("! In template create, template is %+v", template)

	casted := m.(map[string]string)
	id, err := api.CreateAppTemplate(template, casted)
	if err != nil {
		return err
	}

	d.SetId(id)
	err = d.Set("name", template.Name)
	if err != nil {
		return err
	}
	err = d.Set("url", template.URL)
	if err != nil {
		return err
	}
	err = d.Set("icon", template.Icon)
	if err != nil {
		return err
	}
	err = d.Set("embeddable", template.Embeddable)
	if err != nil {
		return err
	}
	err = d.Set("load_in_background", template.LoadInBackground)
	if err != nil {
		return err
	}

	return applicationTemplateRead(d, m)
}

// Check if resource exists
// If yes, call API to get remote state
// Use it to set local state
func applicationTemplateRead(d *schema.ResourceData, m interface{}) error {
	if m == nil {
		return fmt.Errorf("error configuring provider")
	}

	casted := m.(map[string]string)

	exists, err := api.AppTemplateExists(d.Id(), casted)
	if err != nil {
		return err
	}
	if !exists {
		d.SetId("")
		return nil
	}

	template, err := api.AppTemplateRead(d.Id(), casted)
	if err != nil {
		return err
	}

	err = d.Set("name", template.Name)
	if err != nil {
		return err
	}
	err = d.Set("url", template.URL)
	if err != nil {
		return err
	}
	err = d.Set("icon", template.Icon)
	if err != nil {
		return err
	}
	err = d.Set("embeddable", template.Embeddable)
	if err != nil {
		return err
	}
	err = d.Set("load_in_background", template.LoadInBackground)
	if err != nil {
		return err
	}

	return nil
}

// Get state from d
// Use it to call API update function
func applicationTemplateUpdate(d *schema.ResourceData, m interface{}) error {
	if m == nil {
		return fmt.Errorf("error configuring provider")
	}

	template := &structs.AppTemplate{
		Name:             d.Get("name").(string),
		URL:              d.Get("url").(string),
		Icon:             d.Get("icon").(string),
		Embeddable:       d.Get("embeddable").(bool),
		LoadInBackground: d.Get("load_in_background").(bool),
	}

	casted := m.(map[string]string)
	err := api.AppTemplateUpdate(d.Id(), template, casted)
	if err != nil {
		return err
	}

	err = d.Set("name", template.Name)
	if err != nil {
		return err
	}
	err = d.Set("url", template.URL)
	if err != nil {
		return err
	}
	err = d.Set("icon", template.Icon)
	if err != nil {
		return err
	}
	err = d.Set("embeddable", template.Embeddable)
	if err != nil {
		return err
	}
	err = d.Set("load_in_background", template.LoadInBackground)
	if err != nil {
		return err
	}

	return applicationTemplateRead(d, m)
}

// Check if template exists
// Call API to delete it
func applicationTemplateDelete(d *schema.ResourceData, m interface{}) error {
	if m == nil {
		return fmt.Errorf("Error configuring provider")
	}

	id := d.Id()
	casted := m.(map[string]string)
	exists, err := api.AppTemplateExists(id, casted)

	if err != nil {
		return err
	}

	if !exists {
		return nil
	}

	return api.DeleteAppTemplate(id, casted)
}

