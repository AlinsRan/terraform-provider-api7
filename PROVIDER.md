# Provider 配置

## 示例

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

## 参数说明

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `endpoint` | string | ✅ | API7 EE 控制面地址，如 `https://127.0.0.1:7443` |
| `api_key` | string（敏感） | ✅ | API Token，通过 `X-API-KEY` 头传递 |
| `gateway_group_id` | string | ✅ | Gateway Group ID，当前 workspace 下所有资源共用 |
| `insecure` | bool | ❌ | 跳过 TLS 证书验证，仅用于本地/测试环境 |

## 获取 API Token

API7 EE 的 root（admin）用户无法通过 API 创建 Token，可通过以下方式获取：

**方式 A（推荐）**：使用非 root 用户登录 Dashboard → 个人设置 → Token 中创建。

**方式 B（开发/测试环境）**：直接写入 PostgreSQL：

```bash
# 1. 生成 token 及哈希
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

# 2. 写入数据库（替换下方占位符）
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
