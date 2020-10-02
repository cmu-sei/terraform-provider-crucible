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
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

// Provider returns an instance of the provider
func Provider() *schema.Provider {
	return &schema.Provider{
		ResourcesMap: map[string]*schema.Resource{
			"crucible_player_virtual_machine":      playerVirtualMachine(),
			"crucible_player_view":                 playerView(),
			"crucible_player_application_template": applicationTemplate(),
			"crucible_player_user":                 user(),
		},
		Schema: map[string]*schema.Schema{
			"username": {
				Type:     schema.TypeString,
				Required: true,
				DefaultFunc: func() (interface{}, error) {
					return os.Getenv("TF_USERNAME"), nil
				},
			},
			"password": {
				Type:     schema.TypeString,
				Required: true,
				DefaultFunc: func() (interface{}, error) {
					return os.Getenv("TF_PASSWORD"), nil
				},
			},
			"auth_url": {
				Type:     schema.TypeString,
				Required: true,
				DefaultFunc: func() (interface{}, error) {
					return os.Getenv("TF_AUTH_URL"), nil
				},
			},
			"token_url": {
				Type:     schema.TypeString,
				Required: true,
				DefaultFunc: func() (interface{}, error) {
					return os.Getenv("TF_PLAYER_TOK_URL"), nil
				},
			},
			"vm_api_url": {
				Type:     schema.TypeString,
				Required: true,
				DefaultFunc: func() (interface{}, error) {
					return os.Getenv("TF_VM_API_URL"), nil
				},
			},
			"player_api_url": {
				Type:     schema.TypeString,
				Required: true,
				DefaultFunc: func() (interface{}, error) {
					return os.Getenv("TF_PLAYER_API_URL"), nil
				},
			},
			"client_id": {
				Type:     schema.TypeString,
				Required: true,
				DefaultFunc: func() (interface{}, error) {
					return os.Getenv("TF_PLAYER_CLIENT_ID"), nil
				},
			},
			"client_secret": {
				Type:     schema.TypeString,
				Required: true,
				DefaultFunc: func() (interface{}, error) {
					return os.Getenv("TF_CLIENT_SECRET"), nil
				},
			},
		},
		ConfigureFunc: config,
	}
}

// This will read in the key-value pairs supplied in the provider block of the config file.
// The map that is returned can be accessed in the CRUD functions in a _server.go file via the m parameter.
func config(r *schema.ResourceData) (interface{}, error) {
	user := r.Get("username")
	pass := r.Get("password")
	auth := r.Get("auth_url")
	playerTok := r.Get("token_url")
	vmAPI := r.Get("vm_api_url")
	playerAPI := r.Get("player_api_url")
	id := r.Get("client_id")
	sec := r.Get("client_secret")

	if user == nil || pass == nil || auth == nil || playerTok == nil || vmAPI == nil || id == nil || sec == nil ||
		playerAPI == nil {
		return nil, nil
	}

	m := make(map[string]string)
	m["username"] = user.(string)
	m["password"] = pass.(string)
	m["auth_url"] = auth.(string)
	m["player_token_url"] = playerTok.(string)
	m["vm_api_url"] = vmAPI.(string)
	m["player_api_url"] = playerAPI.(string)
	m["client_id"] = id.(string)
	m["client_secret"] = sec.(string)
	return m, nil
}

