---
page_title: "crucible_player_team_role Data Source"
description: |-
  Looks up a team role in the Crucible Player API by name.
---

# crucible_player_team_role

Looks up a Player team role by name.

```hcl
data "crucible_player_team_role" "observer" {
  name = "Observer"
}
```

## Arguments

- `name` - (Required) Team-role name to look up.
- `case_insensitive` - (Optional) Match the team-role name without regard to case. Defaults to `false`.

## Attributes

- `id` - Team-role UUID.
- `all_permissions` - Whether the role grants every team permission.
- `immutable` - Whether the role is immutable.
- `permissions` - Unordered set of team-permission UUIDs assigned to the role.
