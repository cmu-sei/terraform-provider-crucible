---
page_title: "crucible_player_team Resource"
description: |-
  Manages a standalone team in a Crucible Player view.
---

# crucible_player_team

Manages one team, its role, permissions, and permission scopes.

> **Warning:** Do not use this resource with nested `team` blocks on the same view. Mixed ownership is unsupported and can cause perpetual differences, overwritten settings, or deletion.

```hcl
resource "crucible_player_team" "students" {
  view_id = crucible_player_view.example.id
  name    = "students"
  role    = "View Member"

  permissions     = ["00000000-0000-0000-0000-000000000001"]
  scoped_team_ids = [crucible_player_team.observers.id]
}
```

`permissions` and `scoped_team_ids` are authoritative unordered sets. Omission means an empty set and removes unlisted relationships.

## Arguments

- `view_id` - (Required, Forces replacement) Owning view UUID.
- `name` - (Required) Team name.
- `role` - (Optional) Team role name. Defaults to `"View Member"`.
- `permissions` - (Optional) Complete set of team-permission UUIDs. Defaults to `[]`.
- `scoped_team_ids` - (Optional) Complete set of target team UUIDs. Defaults to `[]`.

## Attributes

- `id` - Team UUID.

## Import

```shell
terraform import crucible_player_team.students <team_uuid>
```

Deleting the parent view cascades to the team, memberships, and application instances. The provider cannot validate ownership when `view_id` is hardcoded.
