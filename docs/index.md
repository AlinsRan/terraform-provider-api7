# API7 Provider

The API7 provider manages resources in [API7 Enterprise Edition](https://api7.ai).

## Example Usage

```terraform
provider "api7" {
  endpoint         = "https://127.0.0.1:7443"
  api_key          = var.api7_api_key
  gateway_group_id = "default"
  insecure         = true
}

variable "api7_api_key" {
  type      = string
  sensitive = true
}
```

## Schema

### Required

- `endpoint` (String) — API7 EE control plane endpoint (e.g. `https://127.0.0.1:7443`).
- `api_key` (String, Sensitive) — API token for authentication.

### Optional

- `gateway_group_id` (String) — Gateway group ID. Defaults to `default`.
- `insecure` (Boolean) — Skip TLS certificate verification. Defaults to `false`.
