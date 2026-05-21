# 快速开始示例

演示如何用 terraform-provider-api7 在 API7 EE 上创建 Consumer、Service 和 Route。

## 前置条件

1. 已按照根目录 [README](../../README.md) 完成 provider 构建与安装。
2. 已配置 `~/.terraformrc` 指向本地 mirror。
3. 已准备好 API7 EE 实例和 API Token（获取方式见 [PROVIDER.md](../../PROVIDER.md)）。

## 运行步骤

```bash
cd examples/quickstart

# 初始化
terraform init

# 预览变更
terraform plan -var="api7_api_key=<your-token>"

# 应用
terraform apply -var="api7_api_key=<your-token>"

# 验证幂等性（再次 plan 应显示 No changes）
terraform plan -var="api7_api_key=<your-token>"

# 销毁
terraform destroy -var="api7_api_key=<your-token>"
```

也可以通过环境变量传入 token，避免在命令行中暴露：

```bash
export TF_VAR_api7_api_key="<your-token>"
terraform apply
```

## 示例资源说明

| 文件 | 资源 | 说明 |
|------|------|------|
| `main.tf` | provider | Provider 配置，连接本地 API7 EE |
| `consumer.tf` | `api7_consumer.alice` | 创建名为 alice 的 consumer |
| `service.tf` | `api7_service.httpbin` | 创建指向 httpbin.org 的服务 |
| `route.tf` | `api7_route.get_anything` | 在 httpbin 服务上创建 GET /anything/* 路由 |
