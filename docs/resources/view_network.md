---
page_title: "crucible_view_network Resource"
description: |-
  Manages a network entry within a view in the Crucible VM API.
---

# crucible_view_network

Manages allowed network entries for a view in Crucible's VM API. View networks control which teams can access specific virtual networks, scoped by provider type and instance.

## Example Usage

```hcl
resource "crucible_view_network" "example" {
  view_id              = var.view_id
  provider_type        = "Vsphere"
  provider_instance_id = var.vcenter
  network_id           = data.vsphere_network.example.id
  name                 = data.vsphere_network.example.name
  team_ids             = [var.team1_id, var.team2_id]
}
```

## Argument Reference

- `view_id` - (Required, ForceNew) The UUID of the view this network belongs to.

- `provider_type` - (Required) The virtualization provider type. Must be one of: `Unknown`, `Vsphere`, `Proxmox`, `Azure`.

- `provider_instance_id` - (Required) Identifier for the provider instance (e.g., vCenter FQDN).

- `network_id` - (Required) The provider-specific network identifier (e.g., vSphere dvportgroup ID).

- `name` - (Required) Display name for the network.

- `team_ids` - (Optional) List of team UUIDs allowed to use this network. Defaults to empty (no teams).

~> Changing `view_id` will destroy and recreate the resource.

## Attribute Reference

- `id` - The UUID of the view network entry.
