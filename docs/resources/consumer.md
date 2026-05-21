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

- `username` (String) — Unique consumer name.

### Optional

- `desc` (String) — Description.
- `plugins` (String, JSON) — Plugin configuration as a JSON string.

### Read-Only

- `id` (String) — Consumer identifier (same as `username`).
