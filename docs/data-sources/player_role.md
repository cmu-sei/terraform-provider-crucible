---
page_title: "crucible_player_role Data Source"
description: |-
  Looks up a system role in the Crucible Player API by name.
---

# crucible_player_role

Looks up a Player system role by name.

```hcl
data "crucible_player_role" "content_developer" {
  name = "Content Developer"
}
```

## Arguments

- `name` - (Required) Role name to look up.
- `case_insensitive` - (Optional) Match the role name without regard to case. Defaults to `false`.

## Attributes

- `id` - Role UUID.
- `all_permissions` - Whether the role grants every system permission.
- `immutable` - Whether the role is immutable.
- `permissions` - Unordered set of permission UUIDs assigned to the role.
