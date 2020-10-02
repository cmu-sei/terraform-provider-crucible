package provider

import (
	"crucible_provider/internal/api"
	"crucible_provider/internal/structs"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func user() *schema.Resource {
	return &schema.Resource{
		Create: userCreate,
		Read:   userRead,
		Update: userUpdate,
		Delete: userDelete,

		Schema: map[string]*schema.Schema{
			"user_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"role": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"is_system_admin": {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

func userCreate(d *schema.ResourceData, m interface{}) error {
	if m == nil {
		return fmt.Errorf("Error configuring provider")
	}

	user := structs.PlayerUser{
		ID:            d.Get("user_id").(string),
		Name:          d.Get("name").(string),
		Role:          d.Get("role"),
		IsSystemAdmin: d.Get("is_system_admin").(bool),
	}

	casted := m.(map[string]string)
	err := api.CreateUser(user, casted)
	if err != nil {
		return err
	}

	// Set local state
	d.SetId(user.ID)

	err = d.Set("name", user.Name)
	if err != nil {
		return err
	}

	err = d.Set("role", user.Role)
	if err != nil {
		return err
	}

	err = d.Set("is_system_admin", user.IsSystemAdmin)
	if err != nil {
		return err
	}

	return userRead(d, m)
}

func userRead(d *schema.ResourceData, m interface{}) error {
	if m == nil {
		return fmt.Errorf("Error configuring provider")
	}

	user, err := api.ReadUser(d.Id(), m.(map[string]string))
	if err != nil {
		return err
	}

	log.Printf("! Read user, state is %+v", user)

	// Set local state - no need to set ID b/c it will not change
	err = d.Set("name", user.Name)
	if err != nil {
		return err
	}

	// We want to set using the name of the role, not its id
	role, err := api.GetRoleByID(user.Role.(string), m.(map[string]string))
	if err != nil {
		return err
	}
	err = d.Set("role", role)
	if err != nil {
		return err
	}

	return d.Set("is_system_admin", user.IsSystemAdmin)
}

func userUpdate(d *schema.ResourceData, m interface{}) error {
	if m == nil {
		return fmt.Errorf("Error configuring provider")
	}

	user := structs.PlayerUser{
		ID:            d.Get("user_id").(string),
		Name:          d.Get("name").(string),
		Role:          d.Get("role"),
		IsSystemAdmin: d.Get("is_system_admin").(bool),
	}
	casted := m.(map[string]string)

	err := api.UpdateUser(user, casted)
	if err != nil {
		return err
	}

	// Set local state

	err = d.Set("name", user.Name)
	if err != nil {
		return err
	}

	err = d.Set("role", user.Role)
	if err != nil {
		return err
	}

	err = d.Set("is_system_admin", user.IsSystemAdmin)
	if err != nil {
		return err
	}

	return userRead(d, m)
}

func userDelete(d *schema.ResourceData, m interface{}) error {
	if m == nil {
		return fmt.Errorf("Error configuring provider")
	}

	id := d.Id()
	casted := m.(map[string]string)
	exists, err := api.UserExists(id, casted)

	if err != nil {
		return err
	}

	if !exists {
		return nil
	}

	return api.DeleteUser(id, casted)
}
