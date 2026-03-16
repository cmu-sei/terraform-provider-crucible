---
page_title: "crucible_player_virtual_machine Resource"
description: |-
  Manages a virtual machine in the Crucible VM API.
---

# crucible_player_virtual_machine

Manages virtual machine resources in Crucible's VM API. VMs can be created, read, updated, and destroyed using Terraform with this provider.

## Example Usage

```hcl
# vSphere VM — using the ID of an existing VM
resource "crucible_player_virtual_machine" "vsphere_example" {
  vm_id    = "6a7ec409-d275-4b31-94d3-a51cb61d2519"
  name     = "User1"
  team_ids = ["46420756-9421-41b7-99b4-1b6d2cba29b3"]
}

# Guacamole VM — with console connection info
resource "crucible_player_virtual_machine" "guacamole_example" {
  url        = "https://guac.example.com/guacamole"
  name       = "User2"
  team_ids   = ["46420756-9421-41b7-99b4-1b6d2cba29b3"]
  embeddable = false

  console_connection_info {
    hostname = "vm1.example.local"
    port     = "22"
    protocol = "ssh"
    username = "user"
    password = "example"
  }
}

# Proxmox VM
resource "crucible_player_virtual_machine" "proxmox_example" {
  name       = "User3"
  team_ids   = ["46420756-9421-41b7-99b4-1b6d2cba29b3"]
  embeddable = true

  proxmox_vm_info {
    id   = 100
    node = "pve"
  }
}
```

## Argument Reference

- `vm_id` - (Optional, ForceNew) A globally unique identifier for this VM. When creating a VM, this will generally point to the ID of a machine created using something like vSphere. If omitted, the provider will generate a UUID.

- `url` - (Optional) The URL to the virtual machine console. Can be any valid HTTP/HTTPS URL. If omitted, the API will use the default URL for the virtual machine's type.

- `name` - (Required) The display name of the VM as shown in the view.

- `user_id` - (Optional) A UUID corresponding to the user of this VM.

- `team_ids` - (Required) A list of UUIDs corresponding to the teams who should have access to this machine. At least one team ID is required.

- `embeddable` - (Optional) Whether the UI should allow opening this VM's console in the embedded view. If `false`, the UI should only allow opening the console in a new tab. Defaults to `true`.

- `console_connection_info` - (Optional) Configuration for connecting to this VM's console through a web-based service like Guacamole.

  - `hostname` - (Optional) The internal hostname or address that Guacamole should connect to.
  - `port` - (Optional) The port to connect to.
  - `protocol` - (Optional) The protocol to use for the connection (`ssh`, `vnc`, `rdp`).
  - `username` - (Optional) Username to connect with.
  - `password` - (Optional) Password to connect with.

- `proxmox_vm_info` - (Optional) Additional metadata required for a virtual machine on a Proxmox hypervisor.

  - `id` - (Optional, ForceNew) The integer ID of the virtual machine within Proxmox.
  - `node` - (Optional) The name of the node that the virtual machine is running on.
  - `type` - (Optional) The type of virtual machine (`QEMU`, `LXC`). Defaults to `QEMU`.

## Attribute Reference

- `id` - The UUID of the virtual machine.
- `default_url` - Whether the URL was computed by the API (i.e., no explicit URL was provided).
