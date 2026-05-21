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

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `endpoint` | string | ✅ | API7 EE control plane address, e.g. `https://127.0.0.1:7443`. |
| `api_key` | string (sensitive) | ✅ | API token sent as `X-API-KEY`. |
| `gateway_group_id` | string | ✅ | Gateway Group ID shared by all resources in this workspace. |
| `insecure` | bool | ❌ | Skip TLS certificate verification. For local/test environments only. |

## Obtaining an API Token

API7 EE's root (`admin`) user cannot create tokens via API. Use one of:

**Option A (recommended):** Dashboard → Profile → Tokens (log in as a non-root user).

**Option B (dev/test only):** Insert directly into PostgreSQL:

```bash
# 1. Generate token hash
python3 -c "
import hashlib, base64, secrets, uuid
token_id = str(uuid.uuid4())
random_part = secrets.token_urlsafe(24)[:32]
uuid_no_dash = token_id.replace('-', '')
token = f'a7ee-{random_part}-{uuid_no_dash}'
salt = secrets.token_urlsafe(18)[:18]
hashed = hashlib.pbkdf2_hmac('sha256', token.encode(), salt.encode(), 4096, 32)
print(f'Token ID : {token_id}')
print(f'Token    : {token}')
print(f'Salt     : {salt}')
print(f'Hashed   : {base64.b64encode(hashed).decode()}')
"

# 2. Write to DB
docker exec api7-ee-postgresql-1 env PGPASSWORD=changeme psql -U api7ee -c "
INSERT INTO tokens (id, name, token, salt, org_id, user_id, expires_at, created_at, updated_at)
VALUES (
  '<TOKEN_ID>',
  'terraform',
  '<HASHED>',
  '<SALT>',
  'default',
  (SELECT id FROM users WHERE username='admin'),
  '2099-12-31 23:59:59+00',
  NOW(), NOW()
);"
```
