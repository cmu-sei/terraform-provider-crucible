---
page_title: "crucible_player_application_instance Resource"
description: |-
  Assigns a Crucible Player application to a team.
---

# crucible_player_application_instance

Manages one application instance assigned to a team.

> **Warning:** Do not manage the same instance through a nested `app_instance` block. Mixed ownership is unsupported and can cause perpetual differences, overwritten display order, or deletion.

```hcl
resource "crucible_player_application_instance" "terminal" {
  team_id        = crucible_player_team.students.id
  application_id = crucible_player_application.terminal.id
  display_order  = 0
}
```

## Arguments

- `team_id` - (Required, Forces replacement) Owning team UUID.
- `application_id` - (Required, Forces replacement) Application UUID.
- `display_order` - (Optional) Display order within the team. Defaults to `0`.

## Attributes

- `id` - Application-instance UUID.

## Import

The Player API's individual instance response omits its team, so import requires both UUIDs:

```shell
terraform import crucible_player_application_instance.terminal <team_uuid>/<instance_uuid>
```

Deleting the parent team, application, or view cascades to the instance.
