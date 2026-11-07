# 只读探针与全球监控

GoPanel 的监控页提供全球地图、国家/区域筛选、分组、搜索、排序、实时节点列表和有限的历史采样。页面位于 `/status/monitor`，原来的 `/admin/monitor` 也使用同一套框架页面。

页面采用现有的 Tailwind + daisyUI 组件，不引入另一套前端构建系统。主题配色由 `assets/js/ui-theme.js` 统一配置；地图使用内置的 jsVectorMap 1.7.0。地图脚本、样式及世界轮廓随二进制嵌入，不需要浏览器访问外部 CDN。多节点预览按钮只生成明确标记的浏览器演示数据，不会写入或上报真实节点。

## 采集与角色

每个进程只有一个可取消的采集循环。HTTP 请求和 SSE 都读取缓存，不再为每个访问者阻塞一秒重新采样。

- CPU、内存和网络总计通过 OpenTelemetry Go 的 `instrumentation/host` 封装采集。
- 上游封装暂不提供磁盘和运行时间，因此这些指标保留少量 gopsutil 跨平台适配。
- CPU 百分比由连续 CPU 时间计数计算；网络速率用本机单调时间的采样间隔计算，重连、浏览器暂停和计数回绕不会产生负值或下溢尖峰。
- 第一帧还没有速率/CPU基线时显示不可用，而不是伪造零使用率。
- 不上报真实 IP、系统主机名、进程列表、网卡地址、磁盘路径或应用配置。名称、标签、价格及位置是操作者提供的公开展示信息，请勿填写秘密。

支持三种角色：

| `role` | 行为 |
| --- | --- |
| `standalone` | 默认模式，面板只展示本机采样，保持原有管理功能的独立认证 |
| `candidate` | 本机采样、接收探针上报、缓存所有节点，并向其他候选上报自己的状态 |
| `probe` | 仅采集和上报，不监听 HTTP，也不初始化或启动 Caddy/FRP 等应用服务 |

当前仍是同一个 GoPanel 二进制，探针模式不是裁剪后的独立小程序。`probe` 模式退出或接收到 SIGINT/SIGTERM 会取消采集和正在进行的上报。

## 配置

监控配置使用 `GOPANEL_CONFIG_DIR` 下的 `gopanel_monitor.json`，与面板自身的账号配置分开，复用同一文件存储层的安全创建能力。第一次启动自动生成持久化的 `node_id`；已有配置损坏时明确报错，不重置节点身份。修改配置需要重启对应进程。

默认设置为：

- `role: standalone`、`public: true`，公开的只有经过筛选的只读监控资源。
- `sample_seconds: 2`、`offline_seconds: 15`。
- `history_samples: 300`、`maximum_nodes: 500`。历史和节点缓存只在内存中，容量有上限。
- 空 `country` 不会自动查询第三方 GeoIP；配置 ISO 两位大写国家代码后地图即可点亮对应区域。
- `latitude` / `longitude` 可选，必须成对提供。未配置精确坐标的节点只参与区域着色，不制造定位点。
- `interfaces` 可选，指定需要统计的真实出口网卡名称；默认使用上游封装的接口累计总计。

### 本机

可以直接使用自动生成的配置，或使用以下简化内容；未提供的普通选项使用默认值，`node_id` 必须保持唯一且稳定：

```json
{
  "node_id": "local-panel-01",
  "name": "我的服务器",
  "country": "CN",
  "region": "Shanghai",
  "tags": ["Service"],
  "public": true
}
```

设置 `public: false` 后，公开页面和公开 API 返回 404；已认证的管理页和私有监控 API 仍然可用。

### 候选聚合节点

建议先配置两到三个候选。所有候选及探针使用相同的 `cluster_id`、同一随机 `cluster_key` 和相同顺序的 `candidates`。每台机器必须使用不同的 `node_id`。

```json
{
  "node_id": "tokyo-candidate-01",
  "name": "JP-Tokyo.Aggregator",
  "role": "candidate",
  "country": "JP",
  "region": "Tokyo",
  "latitude": 35.68,
  "longitude": 139.69,
  "tags": ["Relay"],
  "cluster_id": "my-monitor-fleet",
  "cluster_key": "REPLACE_WITH_32_OR_MORE_RANDOM_CHARACTERS",
  "advertise_url": "https://status-a.example.com",
  "candidates": [
    "https://status-a.example.com",
    "https://status-b.example.com",
    "https://status-c.example.com"
  ]
}
```

`advertise_url` 必须出现在候选列表中。URL 必须是普通 HTTP(S) origin，不包含子路径、query、fragment、内嵌账号或结尾斜线。

探针密钥是独立的监控凭据，不要使用面板管理员密码。可在本机 PowerShell 7 中生成随机密钥，再安全分发，不要提交到仓库或发送到公开提示词：

```powershell
[Convert]::ToHexString([Security.Cryptography.RandomNumberGenerator]::GetBytes(32))
```

### 普通探针

```json
{
  "node_id": "frankfurt-probe-01",
  "name": "DE-Frankfurt.Cloud",
  "role": "probe",
  "country": "DE",
  "region": "Frankfurt",
  "tags": ["Service"],
  "cluster_id": "my-monitor-fleet",
  "cluster_key": "REPLACE_WITH_THE_SAME_RANDOM_CLUSTER_KEY",
  "candidates": [
    "https://status-a.example.com",
    "https://status-b.example.com",
    "https://status-c.example.com"
  ]
}
```

仍通过原有方式启动二进制，不新增 action/param CLI。探针不需要公网入站端口。

### TLS

上报默认验证 HTTPS 证书，不提供关闭验证的开关。私有 CA 可通过 `ca_file` 配置 PEM 证书文件，相对路径以配置目录为基准。

面板自动生成的自签名证书只包含 `localhost`，不能直接作为远程域名的有效证书。真实多机部署应使用有正确域名/IP SAN 的证书，或现有 HTTPS 反向代理。使用私有 CA 时，既需要信任该 CA，也需要证书名称匹配候选地址。

仅本机回环 HTTP 测试无需额外开关；其他明文 HTTP origin 必须显式设置 `allow_http: true`。该开关只允许上报客户端使用 HTTP，不会把现有管理端口改成明文 HTTP。不要在公网通过 HTTP 分发或发送集群密钥。

## 简化选主与故障切换

这里选择的是**只读优先汇总节点，不是共识主节点**：

1. 探针向所有配置好的候选发送同一份最新报告，候选都保持自己的热缓存。
2. 收到有效报告响应后记录候选的可达状态，按配置顺序选择第一个近期可达的候选。
3. 优先候选失联、超过健康期限后，选择下一个可达候选；其他候选已有报告，不需要等待旧主恢复或传输数据库。
4. 有界上报循环读取最新快照，不累计无限重试队列，也不自动向候选列表之外的地址重定向凭据。

网络分区时允许出现多个不同的只读视图；页面和 API 用 `coordination: best_effort` 明确标记。不得把这个机制用作配置主节点、告警唯一执行器、分布式锁或远程命令调度器。不需要 Raft、成员 gossip 或数据库。

任意候选都可以访问自己的 `/status/monitor` 查看缓存。选主不迁移域名、IP或浏览器地址；需要一个长期稳定的入口时，仍应使用有健康检查的反代/负载均衡或准备多个候选地址。

## HTTP 协议

私有读取仍使用面板 Cookie / HTTP Basic：

- `GET /api/v1/monitor/status`：本机缓存，第一帧未准备好时返回 503。
- `GET /api/v1/monitor/nodes`：汇总、集群健康和节点列表。
- `GET /api/v1/monitor/nodes/{id}`：节点详情。
- `GET /api/v1/monitor/nodes/{id}/history`：有界内存历史。
- `GET /api/v1/monitor/cluster`：只读候选/优先汇总信息。
- `GET /api/v1/monitor/events`：SSE 全量快照。

公开读取在 `/api/v1/public/monitor` 下提供对应的 `nodes`、节点详情、`history`、`cluster`、`events`，另有 `GET /config` 返回不含密钥的展示信息。公开资源没有配置、执行或删除方法，也不接受 query 动作参数。

上报使用 `POST /api/v1/monitor/reports`，仅候选接收，要求：

```text
Content-Type: application/json
Authorization: Bearer <cluster_key>
X-GoPanel-Cluster: <cluster_id>
```

报告最大 32 KiB。请求类型由 OpenAPI 中的 `reportMonitorMetrics` 描述，包含 `node`、`boot_id`、`sequence`、`collected_at` 和规范化 `metrics`。不接受未知字段或命令字段；监控密钥不能访问面板管理 API。

按节点 ID、启动会话和序列去重。重复/旧报告不延长在线时间，过期启动会话的重放不替换新状态。在线时间以候选收到新报告的本机时间判断，而不是信任探针时钟；报告时间仍需要处于合理时钟偏差窗口，因此机器应启用正常的时间同步。

同一集群密钥对应一个相互信任的监控组，持有它的成员可上报展示数据，不能执行管理操作。它不是每节点独立证书体系；不要把密钥交给不可信探针。远程报告也不能覆盖候选自身的节点 ID。

## 数据边界

- 没有持久化监控历史、账期流量或节点库存；候选重启后靠新的报告重建缓存，此前离线的节点不会凭空恢复。
- 总流量是当前报告的接口累计计数，通常自系统/网卡启动以来累计；不是月度账单。接口重置会使计数下降，速率重新建立基线。
- 默认接口总计可能包含虚拟、隧道或回环接口；需要真实出口统计时显式指定 `interfaces`，不要重复计入同一流量路径。
- 公开 SSE 并发有上限，HTTP 读取及上报分别限速。只读不等于没有数据泄露或资源消耗风险。
- 地图位置由配置提供，不自动发送服务器 IP 到 GeoIP 服务。缺少国家或坐标时仍显示节点，不制造地理位置。

后续其他前端页可逐步复用框架主题和布局；这次不同时重写所有管理模块。
