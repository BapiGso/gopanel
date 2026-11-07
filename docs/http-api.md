# GoPanel Agent HTTP API

GoPanel 的管理 API 统一位于 `/api/v1`。网页和 Agent 调用同一套接口、管理同一批进程内服务实例，没有新增 CLI 或子进程管理器。

## 调用约定

- `/admin/*` 只负责页面渲染；原来的 `?action=`、`?status=`、`?type=`、`?mode=` 操作入口已移除。
- 资源标识放在 URL 路径中，例如 `/cron/jobs/backup`、`/docker/containers/{id}`。
- 写操作使用 JSON 对象，并设置 `Content-Type: application/json`。不再混用表单、裸文本、`param` 或 `option` 包装。未知字段、错误类型和未声明的 query 参数会被拒绝。
- `GET` 用于读取；`POST` 创建资源或执行明确动作；`PUT` 替换配置或内容；`PATCH` 修改明确字段；`DELETE` 删除资源。
- 生命周期动作单独命名，例如 `POST /caddy/reload`，而不是 `PUT /website?status=restart`。
- 布尔值使用 `true` / `false`，不接受 `"on"` 或 `"true"`。`PATCH` 保留未提供的字段，重复设置同一个布尔值不会再次翻转状态。
- JSON 请求体上限为 4 MiB。上传、下载、SSE 和 WebSocket 是明确的协议例外，不混入普通 JSON CRUD。
- 浏览器的管理写操作和终端 WebSocket 必须来自同源页面；后端 HTTP 工具不需要伪造 `Origin`。

成功响应：

```json
{"data":{"state":"running","autostart":false,"pending_changes":false}}
```

错误响应：

```json
{"error":{"code":"revision_conflict","message":"Configuration changed; read it again before saving"}}
```

参数校验错误的 `error.details` 包含字段、规则和规则参数，方便 Agent 修正请求。状态码约定：`400` JSON 或协议错误、`401` 未认证、`403` 权限不足、`404` 资源不存在、`409` 状态或版本冲突、`413` 请求过大、`415` 类型不支持、`422` 字段或应用配置无效、`501` 当前实现不支持、`503` 管理后端不可用。

## 认证和发现

API 支持现有面板登录产生的 `panel_token` Cookie，也支持使用面板用户名和密码的 HTTP Basic Authentication。必须通过 HTTPS 调用；不要把明文密码放进提示词、URL、代码仓库或请求日志。

- `GET /api/v1`：API 版本及 OpenAPI 地址。
- `GET /api/v1/openapi.json`：受认证保护的 OpenAPI 3.1 文档，直接由注册的路由及 Go 请求/响应类型生成。
- 网页仍通过配置好的安全入口登录，API 不另建用户、密钥或配置状态库。

PowerShell 7 示例：

```powershell
$credential = Get-Credential
$base = 'https://localhost:8443/api/v1'
$spec = Invoke-RestMethod -Uri "$base/openapi.json" -Authentication Basic -Credential $credential
$document = Invoke-RestMethod -Uri "$base/caddy/config" -Authentication Basic -Credential $credential
$body = @{
    content = $document.data.content
    revision = $document.data.revision
} | ConvertTo-Json
Invoke-RestMethod -Uri "$base/caddy/config" -Method Put -ContentType 'application/json' -Body $body -Authentication Basic -Credential $credential
Invoke-RestMethod -Uri "$base/caddy/reload" -Method Post -Authentication Basic -Credential $credential
```

面板默认自签名证书需要先在客户端建立信任。本地开发可临时使用 PowerShell 的 `-SkipCertificateCheck`，不要把跳过证书校验作为正常部署方案。

这是完整的面板管理员权限，不是带权限范围限制的 Agent Token。应把凭据交给可信的 HTTP 工具/执行环境保管；项目仍不建议用于生产环境。

## 配置与运行状态

配置文件资源统一返回 `content` 和 `revision`：

```json
{"data":{"content":"{\nadmin off\n}\n:2015\nrespond hello\n","revision":"64-character-sha256"}}
```

`PUT /caddy/config`、`PUT /frp/{service}/config` 和 `PUT /headscale/config` 的请求体都是：

```json
{"content":"application configuration text","revision":"the revision read previously"}
```

先读取再修改，提交读取时的 `revision`。过期版本返回 `409 revision_conflict`，不会覆盖其他用户或 Agent 的修改；配置采用同目录临时文件和重命名保存。收到冲突后重新读取、合并，而不是盲目重试。

面板自身的持久化设置同样需要修订检查：`PATCH /panel/settings`、`PATCH /webdav/settings`、`PATCH /caddy/settings`、`PATCH /frp/{service}/settings` 必须带 `revision`。面板/WebDAV 的 GET 返回 `revision`；Caddy/FRP 的状态 GET 返回 `settings_revision`，设置 PATCH 使用该值，而不是原生配置的 `revision` 或 `applied_revision`。这些设置共享面板配置文件的一个修订值。

```powershell
$settings = Invoke-RestMethod -Uri "$base/webdav/settings" -Authentication Basic -Credential $credential
$body = @{ enable = $false; revision = $settings.data.revision } | ConvertTo-Json
Invoke-RestMethod -Uri "$base/webdav/settings" -Method Patch -ContentType 'application/json' -Body $body -Authentication Basic -Credential $credential
```

配置读取不会隐式创建新账号来覆盖损坏配置，保存失败不会发布候选内存状态。配置目录、旧格式迁移和存储边界见 [配置存储说明](configuration.md)。

**保存不等于应用**：保存不会启动、重载或重启服务。Caddy/FRP 保存时检查配置格式，实际启动时仍可能因为端口、证书、权限或远端连接失败而报错。应用是单独的生命周期请求。

`autostart` 只控制下次面板启动时是否自动启动服务，不代表当前已经运行。`applied_revision` 表示加载的配置版本，`pending_changes` 表示磁盘配置尚未加载。服务循环启动成功也不等于远端可达或所有端口均已就绪；FRPC 和 Headscale 的异步启动返回 `202`，需要随后读取状态和错误。

## 资源入口

下表中的路径相对于 `/api/v1`，字段和详细响应以 OpenAPI 为准。

| 模块 | 读取 | 修改与动作 |
| --- | --- | --- |
| Caddy | `GET /caddy/config`、`GET /caddy/status` | `PUT /caddy/config`、`PATCH /caddy/settings`、`POST /caddy/start`、`POST /caddy/reload`、`POST /caddy/stop` |
| FRP | `GET /frp/{service}/config`、`GET /frp/{service}/status` | `PUT /frp/{service}/config`、`PATCH /frp/{service}/settings`、`POST /frp/{service}/start`、`POST /frp/{service}/reload`、`POST /frp/{service}/stop` |
| 面板 | `GET /panel/settings` | `PATCH /panel/settings`、`POST /panel/update`、`POST /panel/restart` |
| WebDAV | `GET /webdav/settings` | `PATCH /webdav/settings` |
| 定时任务 | `GET /cron/jobs`、`GET /cron/jobs/{name}` | `POST /cron/jobs`、`PUT/PATCH/DELETE /cron/jobs/{name}`、`POST /cron/jobs/{name}/run` |
| 文件 | `GET /files/{path}` | `POST /files`、`PUT/PATCH/DELETE /files/{path}` |
| Docker | `GET /docker/status`、`GET /docker/containers`、`GET /docker/images` | `POST /docker/containers/{id}/{action}`（`start`、`stop`、`restart`、`pause`、`unpause`）、`DELETE /docker/containers/{id}`、`DELETE /docker/images/{id}` |
| 防火墙 | `GET /firewall/rules` | `POST /firewall/rules`、`DELETE /firewall/rules/{id}` |
| SSH 终端 | `GET /terminals`、`GET /terminals/{id}` | `POST /terminals`、`PATCH /terminals/{id}`、`DELETE /terminals/{id}` |
| 监控 | `GET /monitor/status`、`GET /monitor/nodes`、节点详情/历史、`GET /monitor/cluster` | 无管理写操作；人工指定汇总机主动读取探针，旧候选模式的 `POST /monitor/reports` 保留兼容 |
| Headscale | `GET /headscale/config`、`GET /headscale/status` | `PUT /headscale/config`、`POST /headscale/start`、受限制的 `POST /headscale/stop` |

`{service}` 只能是 `frps` 或 `frpc`。Caddy 保留原生 Caddyfile，FRP 保留现有原生配置文件，未增加网站/代理/证书的另一套配置 DSL。

### Caddy 安全边界

Caddy 是面板进程中的同一个实例；HTTP 的启动、重载和停止都通过 `caddy.Load` / `caddy.Stop` 操作它。为避免原生 Admin API 的 `/stop` 退出整个面板进程，加载时始终禁用 Caddy Admin API。也禁用 Caddy JSON 自动保存，以面板维护的 Caddyfile 作为配置来源。

### 定时任务

使用稳定的任务名称，而不是会在删除后变化的数组下标。名称匹配 `[A-Za-z0-9._-]{1,64}`，重复创建返回 `409`。创建/替换使用标准五字段 cron 表达式或 `interval_seconds`，两者只能选一个：

```json
{"name":"backup","schedule":"0 2 * * *","script":"tar -czf /backup/home.tar.gz /home"}
```

```json
{"name":"poll","interval_seconds":300,"script":"curl -f https://example.com/health"}
```

`PATCH /cron/jobs/backup` 的请求体为 `{"paused":true}` 或 `{"paused":false}`。暂停任务需要先恢复再手动运行；`run` 返回接受执行，不保证脚本成功。任务列表目前仍存于内存，重启后丢失；脚本使用现有的 `sh` 运行方式。

### 文件及协议例外

逻辑路径相对于原有文件管理根目录，`GET /files/` 读取根目录。路径中的各个片段需要 URL 编码，例如 `/files/home/user/a%20b.txt`，不要把整个路径当作一个编码后的片段；路径只解码一次，拒绝 `..` 越界。

- `POST /files`：`{"path":"/home/user/note.txt","type":"file"}`，`type` 也可以是 `directory`；不会覆盖已存在资源。
- `PUT /files/home/user/note.txt`：`{"content":"new UTF-8 text"}`，允许空内容。
- `PATCH /files/home/user/note.txt`：`{"permissions":"0600"}` 或 `{"destination":"/home/user/renamed.txt"}`，一次只做一个操作。
- `DELETE /files/{path}` 递归删除文件或目录，禁止删除根目录。
- `GET /downloads/{path}` 下载二进制内容，不返回 JSON。
- `POST /uploads/{directory}` 使用 `multipart/form-data` 的 `files` 字段，整个请求最大 32 MiB；同名文件会被覆盖，多文件上传不是事务。
- `GET /monitor/events`、`GET /docker/events` 使用 SSE，每个 `data:` 帧仍是 `{"data":...}` 或 `{"error":...}`。
- `GET /terminals/{id}/data` 升级到 WebSocket；断开连接释放现有 SSH 会话，不是普通 JSON CRUD。创建终端使用 `host`、`port`、`username`、`password`、`rows`、`cols`，不再混用 `user` / `pwd` 和 query 参数。

监控提供普通 JSON 快照，Agent 无需打开长连接。Docker 和系统库返回的原生对象保留其原生字段，面板自身的资源字段使用明确的 JSON 名称。

监控读取后台采集缓存，不再为每个请求重新阻塞采样；`/monitor/status` 返回规范化的本机节点视图，旧的底层 gopsutil 对象字段不再作为监控协议暴露。默认公开只读页面 `/status/monitor`，公开 JSON/SSE 位于 `/api/v1/public/monitor`。新部署只在汇总机配置 `peers` IP 列表和共享 `key`，探针只配置相同的 `key`，角色和节点身份自动补齐；`/monitor/cluster` 的 `coordination: manual` 表示人工指定汇总机，不再自动选主。

探针默认在独立的 TCP 9555 端口提供 `GET /api/v1/probe/report`，不注册任何面板管理路由。请求及响应使用带时间戳/随机挑战的 HMAC-SHA256 身份校验，密钥不在 HTTP 中明文传输；HTTP 指标本身不加密，需要保密时使用私网或 HTTPS。这个独立协议由汇总端实现，普通 LLM HTTP 工具仍通过汇总端的 JSON 读取 API 查看节点，无需自己实现签名。其他管理资源仍需原有面板认证；监控密钥不能用于面板管理。部署、TLS、地图位置、旧配置兼容和内存历史边界见 [探针与全球监控说明](monitoring.md)。

### 已知边界

- 面板端口和安全入口路径保存后，需要显式重启；保存不会中途断开当前请求。用户名和密码修改保留现有认证实现。读取面板/WebDAV 设置不返回密码，只返回 `password_configured`。
- 自动重启仅支持 Linux；其他平台返回 `501 not_supported`，需要手动重启。
- FRP 重载会验证新配置后停止旧实例，再创建新实例，不保证端口切换无间断或失败自动回滚。
- Headscale 保留现有实验性的配置字段，持久化在 `gopanel_headscale.json`，**不是完整的上游 YAML 配置能力**。可先 `GET /headscale/config` 查看现有字段。Windows 不支持其进程内运行。
- 上游 Headscale 没有安全的嵌入式 Shutdown 入口，运行中请求停止返回 `501 not_supported`，不再假装停止成功或发送可能退出面板的进程信号。此模块目前不提供自动启动设置或安全重载。
- 自更新、服务启动、SSH、Docker 和防火墙调用涉及真实资源或网络条件；HTTP 参数和协议校验通过，不代表操作一定成功。

## 后续模块接入

在模块中定义只有 JSON 字段的请求结构，使用 `httpapi.Decode`、`httpapi.OK/Respond`、`httpapi.NewError`；通过 `httpapi.Router.Handle` 同时注册路由和 `operationId`、请求/响应类型。页面 Handler 只渲染 HTML，业务操作仍使用模块原有实例或服务实现。这样无需另写一份 function-call 参数映射或维护与路由脱节的接口清单。
