# terraform-provider-api7

Terraform Provider for [API7 Enterprise](https://api7.ai), built with [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) + [terraform-plugin-framework](https://github.com/hashicorp/terraform-plugin-framework).

## 支持的资源

| Resource | Create | Read | Update | Delete | Import |
|----------|--------|------|--------|--------|--------|
| `api7_consumer` | ✅ | ✅ | ✅ | ✅ | ✅ |
| `api7_service` | ✅ | ✅ | ✅ | ✅ | ✅ |
| `api7_route` | ✅ | ✅ | ✅ | ✅ | ✅ |

---

## 快速开始

### 前置条件

- Terraform >= 1.0
- Go >= 1.21（仅构建时需要）
- API7 Enterprise 3.x 实例

### 1. 构建 Provider

```bash
git clone https://github.com/AlinsRan/terraform-provider-api7.git
cd terraform-provider-api7

# 生成客户端代码（如已有 client.gen.go 可跳过）
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1
oapi-codegen --config oapi-codegen.yaml openapi-subset.yaml

# 编译
go build -o terraform-provider-api7 .
```

### 2. 安装到本地 Terraform

```bash
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
PLUGIN_DIR="$HOME/.terraform.d/plugins/registry.terraform.io/api7/api7/0.1.0/${OS}_${ARCH}"

mkdir -p "$PLUGIN_DIR"
cp terraform-provider-api7 "$PLUGIN_DIR/"
```

配置 `~/.terraformrc` 使用本地 mirror：

```hcl
provider_installation {
  filesystem_mirror {
    path    = "/home/<YOUR_USER>/.terraform.d/plugins"
    include = ["registry.terraform.io/api7/api7"]
  }
  direct {
    exclude = ["registry.terraform.io/api7/api7"]
  }
}
```

### 3. 获取 API Token

> **说明**：API7 EE 的 root 用户无法通过 API 创建 token，需通过以下方式之一获取：
>
> **方式 A（推荐）**：在 Dashboard → 个人设置 → Token 中创建（使用非 root 用户登录）。
>
> **方式 B**：直接插入数据库（仅开发/测试环境）：

```bash
# 1. 生成 token 哈希
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

### 4. 编写 Terraform 配置

```hcl
# main.tf
terraform {
  required_providers {
    api7 = {
      source  = "registry.terraform.io/api7/api7"
      version = "0.1.0"
    }
  }
}

provider "api7" {
  endpoint         = "https://127.0.0.1:7443"
  api_key          = var.api7_api_key
  gateway_group_id = "default"
  insecure         = true   # 仅本地自签名证书时使用
}

variable "api7_api_key" {
  type      = string
  sensitive = true
}
```

```hcl
# service.tf
resource "api7_service" "httpbin" {
  name = "httpbin"
  desc = "HTTPBin service"

  upstream = {
    nodes = [
      {
        host   = "httpbin.org"
        port   = 80
        weight = 100
      }
    ]
    scheme = "http"
    type   = "roundrobin"
  }
}
```

```hcl
# route.tf
resource "api7_route" "get_anything" {
  name       = "get-anything"
  service_id = api7_service.httpbin.id
  paths      = ["/anything/*"]
  methods    = ["GET"]
}
```

```hcl
# consumer.tf
resource "api7_consumer" "alice" {
  username = "alice"
  desc     = "Test consumer"
  # 注意：认证插件（key-auth 等）需通过 API7 Dashboard 的
  # Credential 子资源管理，不能直接写在 plugins 字段中。
}
```

### 5. 运行

```bash
export TF_VAR_api7_api_key="<your-token>"

terraform init
terraform plan
terraform apply

# 验证幂等性（apply 后再 plan，应显示 No changes）
terraform plan

# 导入已有资源
terraform import api7_consumer.bob bob
terraform import api7_service.existing <service-id>
terraform import api7_route.existing <route-id>

# 销毁
terraform destroy
```

---

## Provider 配置参考

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `endpoint` | string | ✅ | Dashboard 地址，如 `https://127.0.0.1:7443` |
| `api_key` | string | ✅ | API Token（X-API-KEY 头） |
| `gateway_group_id` | string | ✅ | 所有资源所属的 Gateway Group ID |
| `insecure` | bool | ❌ | 跳过 TLS 证书验证，仅用于本地测试 |

## 资源配置参考

### `api7_consumer`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `username` | string | ✅ | 唯一标识，变更会触发资源重建 |
| `desc` | string | ❌ | 描述 |
| `plugins` | string (JSON) | ❌ | 插件配置（JSON 字符串）。注意 API7 EE 不支持直接在 consumer 绑定认证插件，需使用 Credential 子资源 |

Import：`terraform import api7_consumer.<name> <username>`

### `api7_service`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | ✅ | 服务名称 |
| `desc` | string | ❌ | 描述 |
| `upstream` | object | ✅ | upstream 配置（见下） |
| `upstream.nodes` | list | ✅ | 后端节点列表，每项含 `host`、`port`、`weight` |
| `upstream.scheme` | string | ❌ | 协议：`http`、`https`、`grpc`、`grpcs` |
| `upstream.type` | string | ❌ | 负载均衡算法：`roundrobin`、`chash`、`least_conn`、`ewma` |
| `plugins` | string (JSON) | ❌ | 插件配置（JSON 字符串） |

> **已知限制**：API7 EE 的 Service GET 响应不返回 upstream 字段。`Read` 时 upstream 从本地 state 保留，`import` 后需执行一次 `apply` 以同步 upstream 配置。

Import：`terraform import api7_service.<name> <service-id>`

### `api7_route`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | ✅ | 路由名称 |
| `service_id` | string | ✅ | 所属 Service ID，变更会触发资源重建 |
| `paths` | list(string) | ✅ | 匹配路径，如 `["/api/*"]` |
| `methods` | list(string) | ❌ | HTTP 方法，如 `["GET", "POST"]`，空表示全部 |
| `plugins` | string (JSON) | ❌ | 插件配置（JSON 字符串） |

Import：`terraform import api7_route.<name> <route-id>`

---

## 维护指南

### 项目结构

```
terraform-provider-api7/
├── main.go                          # Provider 入口
├── oapi-codegen.yaml                # 客户端生成配置
├── openapi-subset.yaml              # OpenAPI 规范子集（手动维护）
├── internal/
│   ├── client/
│   │   └── client.gen.go           # oapi-codegen 自动生成，勿手动修改
│   └── provider/
│       ├── provider.go             # Provider 框架
│       ├── consumer_resource.go    # Consumer 资源
│       ├── service_resource.go     # Service 资源
│       └── route_resource.go       # Route 资源
└── examples/                        # 示例 HCL 配置
```

### 当 API7 OpenAPI 规范发生更新时

**Step 1：确认变更范围**

```bash
# 在 api7ee-3-control-plane 仓库中查看变更
git diff openapi/routers/consumer.yaml
git diff openapi/schemas/service.yaml
git diff openapi/schemas/route.yaml
```

**Step 2：更新 openapi-subset.yaml**

`openapi-subset.yaml` 是从完整规范中手动裁剪的子集，只包含 Provider 用到的路径。根据实际变更，在文件中同步修改对应的 request/response schema。

> 完整规范位于 `internal/pkg/consts/manifests/openapi.generated.yaml`（执行 `make openapi` 后生成）。

**Step 3：重新生成客户端**

```bash
oapi-codegen --config oapi-codegen.yaml openapi-subset.yaml
```

**Step 4：修复编译错误**

```bash
go build ./...
```

常见错误及处理方式：

| 错误 | 原因 | 处理 |
|------|------|------|
| `undefined: client.XxxJSONRequestBody` | 生成的类型名变了 | 搜索新类型名替换 |
| `cannot use X as type Y` | 字段类型变更 | 调整 resource 文件中的类型转换 |
| `apiResp.JSON200.Value.XxxField undefined` | 响应结构变更 | 更新 `build*ModelFromResponse` 函数 |

**Step 5：验证**

```bash
go build ./...
cd examples && terraform plan
```

---

### 新增字段（最常见）

以 Service 新增 `timeout` 字段为例：

**1. `internal/provider/service_resource.go` — Schema 中加字段**

```go
"timeout": schema.Int64Attribute{
    Optional:    true,
    Computed:    true,
    Description: "Upstream timeout in seconds.",
},
```

**2. Model struct 中加字段**

```go
type ServiceResourceModel struct {
    // ...existing fields...
    Timeout types.Int64 `tfsdk:"timeout"`
}
```

**3. `buildServiceRequestBody` 中加字段**

```go
if !m.Timeout.IsNull() && !m.Timeout.IsUnknown() {
    body["timeout"] = m.Timeout.ValueInt64()
}
```

**4. `buildServiceModelFromResponse` 中加字段**

```go
if v, ok := m["timeout"].(float64); ok {
    state.Timeout = types.Int64Value(int64(v))
}
```

---

### 新增资源

以新增 `api7_ssl` 资源为例：

```bash
# 1. 在 openapi-subset.yaml 中添加 SSL 相关路径
# 2. 重新生成客户端
oapi-codegen --config oapi-codegen.yaml openapi-subset.yaml

# 3. 参考 consumer_resource.go 创建新文件
cp internal/provider/consumer_resource.go internal/provider/ssl_resource.go
# 修改类型名、字段、API 调用方法

# 4. 在 provider.go 中注册
# Resources() 函数中加入 NewSSLResource
```

---

### 版本对应关系

| API7 EE 版本 | Provider 版本 | 备注 |
|-------------|--------------|------|
| 3.9.x | 0.1.x | 当前 |
| 3.10.x+ | 0.2.x+ | 按需更新 |

---

## 已知限制

1. **Service upstream 不可从 API 读回**：API7 EE 的 Service GET 接口响应不包含 upstream 字段，`import` 后需手动补充 upstream 配置再执行 `apply`。
2. **Consumer 认证插件**：API7 EE 要求认证插件（`key-auth`、`jwt-auth` 等）通过 Credential 子资源管理，不支持直接写在 Consumer 的 `plugins` 字段。
3. **Gateway Group 固定**：当前 Provider 仅支持单个 Gateway Group，每个 Terraform workspace 对应一个 Group。

---

## 开发

```bash
# 运行单元测试（暂无）
go test ./...

# 本地构建并安装
make install   # 或手动执行上方安装步骤
```
