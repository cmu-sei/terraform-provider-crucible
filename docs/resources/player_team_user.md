---
page_title: "crucible_player_team_user Resource"
description: |-
  Manages a user membership in a Crucible Player team.
---

# crucible_player_team_user

Manages one user-to-team membership.

> **Warning:** Do not manage the same membership through a nested `user` block. Mixed ownership is unsupported and can cause perpetual differences, overwritten roles, or deletion.

```hcl
resource "crucible_player_team_user" "student" {
  team_id = crucible_player_team.students.id
  user_id = crucible_player_user.student.id
}
```

Omitting `role` stores a null membership role, causing the user to inherit the team role. Setting `role` creates an explicit override; removing it clears the override.

## Arguments

- `team_id` - (Required, Forces replacement) Team UUID.
- `user_id` - (Required, Forces replacement) User UUID.
- `role` - (Optional) Explicit membership role name.

## Attributes

- `id` - Membership UUID.

## Import

```shell
terraform import crucible_player_team_user.student <membership_uuid>
```

Memberships are unique by `(team_id, user_id)`. Import an existing membership instead of attempting to create a duplicate.
