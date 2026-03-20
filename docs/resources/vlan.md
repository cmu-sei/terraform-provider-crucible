---
page_title: "crucible_vlan Resource"
description: |-
  Acquires and releases VLANs in the Crucible Caster API.
---

# crucible_vlan

Acquires and releases VLAN resources in Crucible's Caster API. The `vlan_id` of the returned VLAN is a number marked as in use at the Caster API, allowing it to be used in a Terraform configuration as the ID of a VLAN without risk of collisions.

Caster groups each set of 4096 VLANs into Pools, which can be further sub-divided into Partitions.

## Example Usage

```hcl
# Use the default partition
resource "crucible_vlan" "default" {}

# Use the partition assigned to a project
resource "crucible_vlan" "project" {
  project_id = var.project_id
}

# Use a specific partition
resource "crucible_vlan" "partition" {
  partition_id = var.partition_id
}

# Use a specific vlan_id in a specific partition
resource "crucible_vlan" "specific_id" {
  partition_id = var.partition_id
  vlan_id      = 10
}

# Use a VLAN with a specific tag in a project's partition
resource "crucible_vlan" "tagged" {
  project_id = var.project_id
  tag        = "red"
}
```

## Argument Reference

- `project_id` - (Optional, ForceNew) The ID of a Project in Caster. If this Project exists and has been assigned a VLAN Partition, the requested VLAN will come from this Partition. Conflicts with `partition_id`.

- `partition_id` - (Optional, ForceNew) The ID of a Partition in Caster. If this Partition exists, the requested VLAN will come from this Partition. Conflicts with `project_id`. If neither `project_id` nor `partition_id` is set, the requested VLAN will come from the system-wide default Partition.

- `tag` - (Optional, ForceNew) If set, will return a VLAN with the specified tag, only if one with the requested tag exists and is not in use in the requested Partition. Otherwise, an error will occur.

- `vlan_id` - (Optional, ForceNew) If set, will return a VLAN with the specified ID, only if it is not in use in the requested Partition. Otherwise, an error will occur.

## Attribute Reference

- `id` - The internal UUID of the VLAN resource.
- `vlan_id` - The numeric VLAN ID (0–4095).
- `pool_id` - The UUID of the Pool this VLAN belongs to.
- `partition_id` - The UUID of the Partition this VLAN belongs to.
- `tag` - The tag assigned to this VLAN, if any.
