---
page_title: "crucible_player_view_default_team Resource"
description: |-
  Manages the default team for a Crucible Player view.
---

# crucible_player_view_default_team

Manages the single default-team association for a Player view.

```hcl
resource "crucible_player_view_default_team" "students" {
  view_id = crucible_player_view.example.id
  team_id = crucible_player_team.students.id
}
```

The team must belong to the specified view. Manage at most one
`crucible_player_view_default_team` resource for each view.
Destroying the resource clears the default only when the view still references
the managed team.

## Arguments

- `view_id` - (Required, Forces replacement) Player view UUID.
- `team_id` - (Required) UUID of the team to use as the view's default. Changing this value switches the default team in place.

## Attributes

- `id` - View UUID.

## Import

Import the association using the view UUID:

```shell
terraform import crucible_player_view_default_team.students <view_uuid>
```

The view must already have a default team. Import discovers its team UUID from
the Player API.
