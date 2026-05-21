# api7_service

管理 API7 EE 中的已发布服务（Published Service）。

## 示例

```terraform
resource "api7_service" "example" {
  name = "httpbin"
  desc = "HTTPBin 服务"

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

完整示例见 [`examples/resources/api7_service/resource.tf`](examples/resources/api7_service/resource.tf)。

## 参数说明

### 必填

- `name`（String）— 服务名称。
- `upstream`（Attributes）— Upstream 配置。
  - `nodes`（List，必填）— 后端节点列表，每项包含：
    - `host`（String，必填）
    - `port`（Number，必填）
    - `weight`（Number，必填）

### 可选

- `desc`（String）— 描述。
- `upstream.scheme`（String）— 协议：`http`、`https`、`grpc`、`grpcs`。
- `upstream.type`（String）— 负载均衡算法：`roundrobin`、`chash`、`least_conn`、`ewma`。
- `plugins`（String，JSON）— 插件配置，JSON 字符串格式。

### 只读

- `id`（String）— 服务 ID。

## 导入

```bash
terraform import api7_service.<名称> <service-id>
```

## 已知限制

API7 EE 的 Service GET 接口响应不包含 upstream 字段。执行 `terraform import` 后需手动补充 upstream 配置并运行一次 `terraform apply` 以同步到网关。
