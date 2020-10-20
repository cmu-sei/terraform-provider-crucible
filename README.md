# Crucible Terraform Provider

[Terraform](https://www.terraform.io/) is an Infrastructure as Code tool for managing cloud-based infrastructure. A provider is a plugin to Terraform that allows for the management of a given resource type. That is, a provider supplies the logic needed to manage infrastructure. There are three main resource types managed by this provider: virtual machines, views, and application templates. They are detailed below.

In order to use the provider, several environment variables must be set:
``` 
TF_USERNAME=<your username>
TF_PASSWORD=<your password>
TF_AUTH_URL=<the url to the authentication service>
TF_PLAYER_TOK_URL=<the url where you get your player authentication token>
TF_CLIENT_ID=<your client ID for authentication>
TF_CLIENT_SECRET=<your client secret for authentication>
TF_VM_API_URL=<the url to the VM API>
TF_PLAYER_API_URL=<the url to the Player API>
``` 

## Virtual Machines

The provider can interact with Crucible's VM API in order to manage virtual machine resources. VMs can be created, read, updated, and destroyed using Terraform with this provider. An example config for a single virtual machine is defined below.

```
resource "crucible_player_virtual_machine" "example" {
	vm_id = "6a7ec409-d275-4b31-94d3-a51cb61d2519"
	url = "http://example.com"
	name = "foobarboz"
	team_ids = ["46420756-9421-41b7-99b4-1b6d2cba29b3",]
}
```
The name of the resource type - the first string after the word "resource" - must be `crucible_player_virtual_machine`. This tells Terraform what kind of infrastructure is being managed. The name of the specific configuration block ("example" in the above configuration) can be any string. The fields within the configuration are detailed below.

<ul>
<li> vm_id: This must be a globally unique identifier (GUID) not shared by any other machines in the same configuration. When creating a VM, this will generally point to the ID of a machine created using something like VSphere. If this is omitted, the provider will generate a GUID for this field. 
<li> url: The URL to the virtual machine console. This can be any string, including a URL that does not actually point to a VM, which is useful for testing purposes. When creating a VM, this should point to the machine whose ID was used in the vm_id field.
<li> name: The name of the VM. This is the name that will show up in the view where the VM is created. It can be any string.
<li> user_id: This is an optional field that, if set, must be a GUID corresponding to the ID of the user of this VM.
<li> team_ids: A list of GUIDs corresponding to the IDs of the teams who should be given access to this machine.
</ul>

## Player Views

The Provider can also interact with Crucible's Player API in order to manage views and the things that live within them such as teams and applications. An example configuration is outlined below. 

An example configuration:
```
resource "crucible_player_view" "example" {
	name = "example"
	description = "This was created from terraform!"
	status = "Active"

	application	{
		name = "testApp"
		embeddable = "false"
		load_in_background = "true"
	}

	team {
		name = "test_team"
		role = SomeRole
		user {
			user_id = "6fb5b293-668b-4eb6-b614-dfdd6b0e0acf"
		}
		app_instance {
			name = "testApp"
			display_order = 0
		}
	}
}
```
As before, the name of this resource type _must_ be the first string after the word "resource" - ie it must be "crucible_player_view". The name of this specific block can, again, be any string.

Inside of the resource block, there is information to configure both the view itself as well as the resources that live inside the view. Only the information about the view itself is required. Thus, it is possible to simply create an empty view. Each field is outlined below.

### The view itself
<ul>
<li> name: The name of this view. This can be any string. Required.
<li> description: A description for this view. This can be any string. Optional.
<li> status: The status of this view. That is, whether it is active. This field is a string. Optional.
</ul>

### Applications
There do not have to be any applications within a view, so no application blocks are required. However, for each application block that is set, certain fields within it must be set. See the above configuration example for the the syntax of creating an application block. The fields of an application are outlined below. Applications should be placed in the configuration in alphabetical order by name. Note that numbers come before letters in an alphabetical ordering. 

Due to a quirk in Terraform's type system, values for `embeddable` and `load_in_background` must be wrapped in quotes. They are still boolean fields; their values just need to be wrapped in quotes as is shown above.

<ul>
<li> app_id: The GUID of this application. Optional. If not set, it will be generated internally.
<li> name: The name of this application. Required.
<li> v_id: The GUID of the view this application will be created under. Optional. If not set, it will automatically be set to the ID of the view this application is under.
<li> url: A URL to associate with this application. Optional.
<li> icon: A string pointing to the icon for this application. Optional.
<li> embeddable: A boolean stating whether this application is embeddable. Optional.
<li> load_in_background: A boolean stating whether this application should be loaded in the background. Optional.
<li> app_template_id: The GUID of an application template to inherit from. Optional.
</ul> 

### Teams
As with applications, there do not need to be any teams within a view. However, for each team that is defined, some fields must be set. The fields of a team block are outlined below. Teams should be placed in the config file in alphabetical order by name. Otherwise, Terraform will try to update the state when changes are not necessary. The users within teams should also be placed in alphabetical order by their user_id. App instances should be placed in alphabetical order by name.
<ul>
<li> name: The name of this team, which can be any string. Required.
<li> role: The name of the role this team falls under, if any. Optional.
<li> permissions: A list of GUIDs corresponding to this team's permissions. Optional.
<li> user: Any users that are in this team. Optional.
	<ul>
	<li> user_id: The GUID of this user. Required.
	<li> role: The name of a role to assign this user. Optional.
	</ul>
<li> app_instance: App application assigned to this team. This field must be set for a team to have access to a given application.
	<ul>
	<li> Name: Required. The name of the application to instantiate.
	<li> display_order: Optional. Determines the order in which applications within a team will be displayed. A value of 0 will make this the first application shown. If not set, this will default to 0.
	</ul>  
</ul>

## Application Templates
The provider can also interact with the Player API to create application templates. These are a distinct resource because application templates do not exist underneath views. An example configuration for an application template is outlined below. 

```
resource "crucible_player_application_template" "template" {
	name = "Example"
	url = "http://example.com"
	icon = "https://www.cs.cmu.edu/sites/default/files/fall10p05_sm_0.jpg"
	embeddable = false
	load_in_background = false
}
```

As before, this resource needs to have a specific name - `crucible_player_application_template`. The name of the instance, however, can be anything. The configuration fields are defined below.
<ul>
<li> name: The name of the application template. Required. 
<li> url: The url this application template should point to. Optional.
<li> icon: The URL to an image to use as the template's icon. Optional.
<li> embeddable: Boolean flag specifying whether this template is embeddable. Optional.
<li> load_in_background: Boolean flags specifying whether this template should load in the background. Optional.
</ul> 

## Users
The provider can also create users within Player. Note that this is distinct from a `user` block inside of a `view`. The block inside of a view assumes a user with the given id already exists, whereas this resource type _creates_ the user. This is intended to be used in conjunction with the Identity provider to create accounts and add corresponding users to Player. Once created, users can be used within teams and views. An example configuration for a user is below.
```
resource "crucible_player_user" "test" {
    user_id = identity_account.test.global_id
    # Make sure the user does not have the email domain in their username
    name = regex("(.*)(@.*)", identity_account.test.username)[0]
    role = "TestRole"
}
```
User properties

- user_id: The GUID to create this user with. Will probably point to an Identity account's GUID. Required.
- name: The name to assign this user. Required.
- role: A role to give this user. Optional.
- is_system_admin: Whether this user is a system admin. Computer.

## Reporting bugs and requesting features

Think you found a bug? Please report all Crucible bugs - including bugs for the individual Crucible apps - in the [cmu-sei/crucible issue tracker](https://github.com/cmu-sei/crucible/issues). 

Include as much detail as possible including steps to reproduce, specific app involved, and any error messages you may have received.

Have a good idea for a new feature? Submit all new feature requests through the [cmu-sei/crucible issue tracker](https://github.com/cmu-sei/crucible/issues). 

Include the reasons why you're requesting the new feature and how it might benefit other Crucible users.	
