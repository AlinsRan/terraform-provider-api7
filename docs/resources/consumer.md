# api7_consumer

Manages a consumer in API7 EE.

## Example Usage

```terraform
resource "api7_consumer" "example" {
  username = "alice"
  desc     = "Example consumer"
}
```

## Schema

### Required

- `username` (String) — Unique consumer name. Changing this forces a new resource.

### Optional

- `desc` (String) — Description.
- `plugins` (String, JSON) — Plugin configuration as a JSON string.

### Read-Only

- `id` (String) — Consumer identifier (same as `username`).

## Import

```bash
terraform import api7_consumer.<name> <username>
```

## Known Limitations

Authentication plugins (e.g. `key-auth`, `jwt-auth`) cannot be set directly on the consumer's `plugins` field in API7 EE. They must be managed through the Credential sub-resource in the Dashboard.
