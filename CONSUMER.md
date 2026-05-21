# api7_consumer

管理 API7 EE 中的 Consumer。

## 示例

```terraform
resource "api7_consumer" "example" {
  username = "alice"
  desc     = "示例 consumer"
}
```

完整示例见 [`examples/resources/api7_consumer/resource.tf`](examples/resources/api7_consumer/resource.tf)。

## 参数说明

### 必填

- `username`（String）— Consumer 唯一标识，变更会触发资源重建。

### 可选

- `desc`（String）— 描述。
- `plugins`（String，JSON）— 插件配置，JSON 字符串格式。

### 只读

- `id`（String）— 与 `username` 相同。

## 导入

```bash
terraform import api7_consumer.<名称> <username>
```

## 已知限制

API7 EE 要求认证插件（`key-auth`、`jwt-auth` 等）通过 Credential 子资源管理，不支持直接写在 Consumer 的 `plugins` 字段中。
