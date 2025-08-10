# Headscale 管理模块

## 概述

Headscale 是一个开源的、自托管的 Tailscale 控制服务器实现。本模块提供了在 GoPanel 中管理 Headscale 服务的功能。

## 主要功能

1. **配置管理**
   - 通过 Web 界面配置 Headscale 参数
   - 配置持久化到 `/etc/headscale/config.json`
   - 支持所有主要配置选项

2. **服务管理**
   - 启动/停止/重启 Headscale 服务
   - 实时状态监控
   - 自动启动功能（通过 viper 配置）

3. **网络配置**
   - IPv4/IPv6 地址范围配置
   - MagicDNS 基础域名设置
   - DERP 服务器配置

## 文件结构

```
core/headscale/
├── handler.go       # HTTP 请求处理器
├── manager.go       # 服务管理器
├── init.go         # 初始化和自动启动
├── unavailable.go  # 非支持平台的处理
├── handler_test.go # 单元测试
└── README.md       # 本文档
```

## API 端点

- `GET /admin/headscale` - 显示配置页面
- `POST /admin/headscale/config` - 保存配置
- `POST /admin/headscale/start` - 启动服务
- `POST /admin/headscale/stop` - 停止服务
- `POST /admin/headscale/restart` - 重启服务
- `GET /admin/headscale/status` - 获取服务状态

## 配置参数

### 必需参数

- `server_url`: Headscale 服务器的公共 URL
- `listen_addr`: 服务监听地址
- `ipv4_prefix`: IPv4 地址分配范围（必须在 100.64.0.0/10 内）
- `ipv6_prefix`: IPv6 地址分配范围（必须在 fd7a:115c:a1e0::/48 内）
- `base_domain`: MagicDNS 基础域名

### 可选参数

- `metrics_listen_addr`: Metrics 端点监听地址
- `grpc_listen_addr`: gRPC API 监听地址
- `private_key_path`: Noise 私钥文件路径
- `database_type`: 数据库类型（默认 sqlite）
- `sqlite_path`: SQLite 数据库文件路径

## 使用说明

### 1. 首次配置

1. 访问 `/admin/headscale` 页面
2. 填写必要的配置信息
3. 点击"保存配置"按钮
4. 点击"启动 Headscale"按钮

### 2. 客户端连接

启动 Headscale 后，客户端可以通过以下方式连接：

```bash
# 使用 tailscale 客户端连接
tailscale up --login-server=https://your-headscale-server.com:443
```

### 3. 管理节点

可以通过 Headscale CLI 工具管理节点：

```bash
# 列出所有节点
headscale nodes list

# 创建用户
headscale users create myuser

# 生成预授权密钥
headscale preauthkeys create --user myuser
```

## 故障排除

### 服务无法启动

1. 检查配置是否正确
2. 确保端口没有被占用
3. 检查日志文件获取详细错误信息

### 客户端无法连接

1. 确保服务器 URL 可访问
2. 检查防火墙规则
3. 验证 TLS 证书配置

### 数据库错误

1. 确保数据库文件路径可写
2. 检查磁盘空间
3. 验证数据库权限

## 安全建议

1. **使用 HTTPS**: 始终使用 HTTPS 作为服务器 URL
2. **限制访问**: 使用防火墙限制 gRPC 和 metrics 端口的访问
3. **定期备份**: 定期备份配置和数据库文件
4. **更新维护**: 保持 Headscale 版本更新

## 依赖项

- github.com/juanfont/headscale
- github.com/labstack/echo/v4
- github.com/spf13/viper

## 注意事项

1. Headscale 仅在 Linux、Darwin 和 FreeBSD 系统上可用
2. 需要 root 权限或适当的系统权限来绑定端口
3. 配置更改需要重启服务才能生效

## 相关链接

- [Headscale 官方文档](https://github.com/juanfont/headscale)
- [Tailscale 文档](https://tailscale.com/kb/)
- [WireGuard 协议](https://www.wireguard.com/)