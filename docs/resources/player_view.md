---
page_title: "crucible_player_view Resource"
description: |-
  Manages a view and its teams and applications in the Crucible Player API.
---

# crucible_player_view

Manages views in Crucible's Player API, including the teams and applications within them.

> **Warning:** Do not mix nested child blocks and standalone child resources for the same view. Mixed ownership is unsupported and can cause perpetual plan differences, overwritten settings, or deletion of child objects.

## Example Usage

```hcl
resource "crucible_player_view" "example" {
  name              = "example"
  description       = "This was created from Terraform!"
  status            = "Active"
  create_admin_team = true
  is_template       = true

  application {
    name               = "testApp"
    embeddable         = "false"
    load_in_background = "true"
  }

  team {
    name = "test_team"
    role = "SomeRole"
    default = true
    scoped_teams = ["observer_team"]

    user {
      user_id = "6fb5b293-668b-4eb6-b614-dfdd6b0e0acf"
    }

    app_instance {
      name          = "testApp"
      display_order = 0
    }
  }

  team {
    name = "observer_team"
  }
}
```

## Argument Reference

### View

- `name` - (Required) The name of this view.
- `description` - (Optional) A description for this view.
- `status` - (Optional) The status of this view. Defaults to `"Active"`.
- `create_admin_team` - (Optional) Whether to automatically create an Admin team. Defaults to `true`.
- `is_template` - (Optional) Whether the view is a reusable Player template. Defaults to `false`.
- `child_management` - (Optional) Child ownership mode. `"inline"` (the default) manages the complete child collection through nested blocks. `"standalone"` ignores remote children so they can be managed with standalone resources. Standalone mode requires `create_admin_team = false` and rejects `application` and `team` blocks. Use `crucible_player_view_default_team` to select a default standalone team.

An inline view with no child blocks authoritatively manages an empty child collection. Use `child_management = "standalone"` when unmanaged or standalone children must be ignored.

### Applications

The `application` block is optional and repeatable. Applications should be placed in alphabetical order by name to avoid unnecessary state changes.

~> Due to a quirk in Terraform's type system, values for `embeddable` and `load_in_background` must be wrapped in quotes (e.g., `"false"`).

- `name` - (Required) The name of this application.
- `app_id` - (Computed) The UUID of this application, generated internally.
- `v_id` - (Computed) The UUID of the view this application belongs to.
- `url` - (Optional) A URL to associate with this application.
- `icon` - (Optional) A string pointing to the icon for this application.
- `embeddable` - (Optional) Whether this application is embeddable.
- `load_in_background` - (Optional) Whether this application should load in the background.
- `app_template_id` - (Optional) The UUID of an application template to inherit from.

### Teams

The `team` block is optional and repeatable. Teams should be placed in alphabetical order by name to avoid unnecessary state changes. Users within teams should also be ordered alphabetically by `user_id`, and app instances by `name`.

- `name` - (Required) The name of this team.
- `team_id` - (Computed) The UUID of this team, assigned by the API.
- `role` - (Optional) The name of the role this team falls under. Defaults to `"View Member"`.
- `default` - (Optional) Whether this is the view's default team. Defaults to `false`. At most one inline team can be the default.
- `permissions` - (Optional) A list of permission IDs for this team.
- `scoped_teams` - (Optional) An unordered set of sibling team names onto which this team's permissions are scoped. Targets must be other `team` blocks in this view; a team cannot target itself. Defaults to `[]`. Terraform manages the complete set and removes API scope relationships not listed here.

#### `user` block (nested inside `team`)

- `user_id` - (Required) The UUID of the user.
- `role` - (Optional) The name of a role to assign to this user within the team.

#### `app_instance` block (nested inside `team`)

- `name` - (Required) The name of the application to instantiate. Must match an `application` block's name.
- `display_order` - (Optional) The display order of this application within the team. Defaults to `0`.
- `id` - (Computed) The UUID of this application instance.

## Attribute Reference

- `id` - The UUID of the view.

## Standalone Child Management

```hcl
resource "crucible_player_view" "example" {
  name              = "example"
  create_admin_team = false
  child_management  = "standalone"
}
```

The provider cannot determine whether a standalone resource using a hardcoded `view_id` targets an inline-managed view. Ensure all standalone children target a view configured with `child_management = "standalone"`.

Deleting a view cascades to its applications, teams, memberships, and application instances in the Player API even in standalone mode. Use Terraform references so child resources are destroyed before their view.

## Migrating Inline Children

Migration requires two applies:

1. Keep all nested blocks. Define matching standalone resources and import each existing application, team, membership, and application instance. Confirm the plan contains imports only.
2. Remove the nested blocks, set `child_management = "standalone"`, and set `create_admin_team = false`. Confirm the plan only detaches nested state and does not replace or delete children.

Import applications, teams, and memberships by UUID. Import application instances as `<team_uuid>/<instance_uuid>`.

`create_admin_team` is only honored when the view is created. Changing it to `false` does not delete an existing Admin team. Before migration, explicitly choose one of:

- Import the Admin team and its membership as standalone resources.
- Delete the Admin team manually before migration.
- Leave the Admin team unmanaged.

Never combine the import and ownership-transition steps in one apply.
