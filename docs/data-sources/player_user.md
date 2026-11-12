---
page_title: "crucible_player_user Data Source"
description: |-
  Looks up one user in the Crucible Player API by name, by identity attributes, or both.
---

# crucible_player_user

Looks up one Player user and returns its id. Match by name, by identity attributes, or by both. The lookup fails unless exactly one user matches.

Names can collide. If Player is configured to store identity attributes, look the user up by an attribute that is unique, such as email.

```hcl
data "crucible_player_user" "alice" {
  identity_attributes = {
    email = "alice@example.com"
  }
  case_insensitive = true
}

resource "crucible_player_team_user" "alice" {
  team_id = crucible_player_team.blue.id
  user_id = data.crucible_player_user.alice.id
}
```

Look up by name when names are unique:

```hcl
data "crucible_player_user" "bob" {
  name = "Bob Smith"
}
```

## Identity attributes

Player records identity attributes from a user's identity provider token when it refreshes that user's claims during the user's authenticated requests. Player only stores the attributes listed in its `ClaimsTransformation:UserAttributes` setting. For example, this setting stores the email claim under the key `email`:

```json
"ClaimsTransformation": {
  "UserAttributes": [
    { "Key": "email", "Name": "Email", "ClaimPath": "email" }
  ]
}
```

As environment variables:

```
ClaimsTransformation__UserAttributes__0__Key=email
ClaimsTransformation__UserAttributes__0__Name=Email
ClaimsTransformation__UserAttributes__0__ClaimPath=email
```

Keys in `identity_attributes` are these `Key` values. Player only returns identity attributes to accounts with the `ViewUsers` system permission, so the provider account needs it to look users up by attribute. A user who has not used Player since the attribute was configured has no value for it and cannot be found by it.

## Arguments

Set at least one of `name` or `identity_attributes`.

- `name` - (Optional) User name to match.
- `identity_attributes` - (Optional) Map of identity attribute values to match, keyed by attribute key. The user must match every entry.
- `case_insensitive` - (Optional) Match the name and identity attribute values without regard to case. Defaults to `false`.

## Attributes

- `id` - User UUID.
- `name` - User name.
- `role_id` - UUID of the user's system role, or empty when the user has none.
- `role_name` - Name of the user's system role, or empty when the user has none.
- `attributes` - Map of every identity attribute Player holds a value for on the user, keyed by attribute key.
