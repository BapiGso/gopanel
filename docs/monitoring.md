# 只读探针与全球监控

GoPanel 的监控页提供全球地图、国家/区域筛选、分组、搜索、排序、实时节点列表和有限的历史采样。页面位于 `/status/monitor`，原来的 `/admin/monitor` 也使用同一套框架页面。

页面采用 Tailwind + daisyUI 组件，地图使用内置的 jsVectorMap 1.7.0。组件和两套主题的 CSS 在开发时预编译，脚本、样式及世界轮廓随二进制嵌入，不访问外部 CDN，也不在浏览器中运行 Tailwind 编译器。页面只显示真实采样，没有多节点预览或演示数据。

**新部署使用人工指定汇总机的拉取模式**：汇总机填其他机器的 IP 和一份共享监控密钥，探针只填相同密钥。没有自动发现、候选列表维护或自动选主。旧候选/主动上报配置仅保留兼容。

## 采集与角色

每个进程只有一个可取消的采集循环。HTTP 请求和 SSE 都读取缓存，不再为每个访问者阻塞一秒重新采样。

- CPU、内存和网络总计通过 OpenTelemetry Go 的 `instrumentation/host` 封装采集。
- 上游封装暂不提供磁盘和运行时间，因此这些指标保留少量 gopsutil 跨平台适配。
- CPU 百分比由连续 CPU 时间计数计算；网络速率用本机单调时间的采样间隔计算，重连、浏览器暂停和计数回绕不会产生负值或下溢尖峰。
- 第一帧还没有速率/CPU基线时显示不可用，而不是伪造零使用率。
- 不上报真实 IP、系统主机名、进程列表、网卡地址、磁盘路径或应用配置。名称、标签、价格及位置是操作者提供的公开展示信息，请勿填写秘密。

角色通常无需手填，由简化配置推断：

| `role` | 行为 |
| --- | --- |
| `standalone` | 默认模式，面板只展示本机采样，保持原有管理功能的独立认证 |
| `master` | 有 `peers` 时自动使用；人工指定汇总机，本机采样并主动拉取其他机器，保留面板管理功能 |
| `probe` | 只有 `key` 时自动使用；采样并在独立 9555 端口提供只读指标，不初始化或启动 Caddy/FRP 等应用服务 |
| `candidate` | 旧模式兼容：接收上报、缓存所有节点，并向其他候选上报自己的状态 |

当前仍是同一个 GoPanel 二进制，探针模式不是裁剪后的独立小程序。只读探针的 HTTP 服务不注册登录、文件、终端、Caddy 或任何其他管理路由。退出或收到 SIGINT/SIGTERM 会取消采集、拉取并关闭探针监听。旧的带 `candidates` 的 `probe` 仍按原来的出站上报方式运行，不新增监听。

## 配置

监控配置使用 `GOPANEL_CONFIG_DIR` 下的 `gopanel_monitor.json`，与面板账号配置分开。第一次启动只生成持久化的 `node_id`，不会写出一大份默认参数。自己准备配置时也可以省略 `node_id`，启动后会原子写回唯一身份。已有的明确身份会保留，损坏配置或显式空身份会报错，不悄悄重置。未填写名称时显示 `Node-<身份前缀>`，不公开真实主机名。修改配置需要重启对应进程。

默认设置为：

- `role: standalone`、`public: true`，公开的只有经过筛选的只读监控资源。
- `sample_seconds: 2`、`offline_seconds: 15`。
- `history_samples: 300`、`maximum_nodes: 500`。历史和节点缓存只在内存中，容量有上限。
- 空 `country` 不会自动查询第三方 GeoIP；配置 ISO 两位大写国家代码后地图即可点亮对应区域。
- `latitude` / `longitude` 可选，必须成对提供。未配置精确坐标的节点只参与区域着色，不制造定位点。
- `interfaces` 可选，指定需要统计的真实出口网卡名称；默认使用上游封装的接口累计总计。

### 汇总机：只填其他机器的 IP

人工挑选一台服务器作为汇总机，配置如下。`role`、`node_id`、`cluster_id`、候选列表和自身公网地址都不用填写：

```json
{
  "peers": ["192.0.2.10", "198.51.100.20", "2001:db8::30"],
  "key": "REPLACE_WITH_32_OR_MORE_RANDOM_CHARACTERS"
}
```

`peers` 中填探针 IP，不用重复填汇总机自己；本机采样会自动加入汇总。裸 IP 使用固定 TCP 9555 端口，IPv4 和 IPv6 都支持。修改节点列表只需修改汇总机这一个文件，探针不维护任何成员列表。

### 探针：只有一个身份密钥

每台其他机器使用相同密钥：

```json
{
  "key": "REPLACE_WITH_THE_SAME_RANDOM_MONITOR_KEY"
}
```

通过原有方式启动 GoPanel 即可，不新增 CLI。每台机器在自己目录启动，不要复制启动后生成的 `node_id`；复制带身份的整份配置会产生身份冲突，汇总端会拒绝它。

名字、分组和地图位置可以选填，不影响接入：

```json
{
  "key": "REPLACE_WITH_THE_SAME_RANDOM_MONITOR_KEY",
  "name": "Frankfurt",
  "country": "DE",
  "tags": ["Service"]
}
```

国家代码用于地图区域着色，精确定位点需另填成对的 `latitude` / `longitude`。系统不会凭空猜测真实地理位置。

如果这台机器还要保留原来的管理面板，显式添加 `"role": "standalone"` 和同一个 `key` 即可同时提供只读探针端口。普通探针不需要这样做。

### 网络与身份

- 汇总机需要能够访问每台探针的 **TCP 9555**；探针不需要知道汇总机 IP，也不再主动上报。建议防火墙只允许汇总机 IP 或使用私网/VPN；NAT 后无法被访问的探针需要先打通网络。
- 需要自定义端口时，探针可填 `"listen": ":9666"`，汇总机对应填 `"peers": ["192.0.2.10:9666"]`。默认无需这两个改动。汇总机不另外开启 9555 监听，它的页面仍位于原有 HTTPS 面板端口的 `/status/monitor`。
- 密钥只用于 HMAC-SHA256 监控身份校验，不是管理员密码。请求签名包含时间戳和随机挑战，响应签名绑定这次挑战及完整报告，防止篡改或拿旧响应冒充新采样。密钥本身不在 HTTP 中发送。
- **身份校验不等于加密**：默认 HTTP 的指标内容仍可被网络观察者读取。需要指标保密时使用私网/VPN，或配置有效证书的 HTTPS 入口，并在 `peers` 中填对应 HTTPS origin；`ca_file` 可指定私有 CA。不关闭 TLS 校验，不向探针下发面板管理员凭据。
- 同一共享密钥内的探针相互信任，不是每节点证书体系；持有密钥的成员有能力伪造组内指标。不要把密钥交给不可信成员。每个 IP 首次连接后绑定其节点身份，运行期间换成不同身份会被拒绝，确认替换机器后重启汇总机即可重新建立绑定。
- 响应最多 32 KiB，只允许规范化指标；并发拉取有上限，不累计无限任务。时钟校验允许一分钟偏差，各机应正常同步时间。
- 首次连接失败的 IP 不会被伪造成有指标的节点；页面的探针可达数量和诊断提示会反映未连接项。收到有效采样后自动进入节点列表，之后失联保留最后采样并标记离线。

共享密钥只需生成一次再分发，PowerShell 7 可用：

```powershell
[Convert]::ToHexString([Security.Cryptography.RandomNumberGenerator]::GetBytes(32))
```

汇总机不会自动迁移或选举。要换汇总机，把相同的 `peers` 和 `key` 配置放到另一台机器即可，探针无需修改；公开入口的域名/IP切换仍由操作者处理。

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

### 旧候选聚合节点（仅兼容）

以下不是新部署的必填项，仅用于已有配置继续运行。旧候选及上报探针仍使用相同的 `cluster_id`、`cluster_key` 和相同顺序的 `candidates`。每台机器必须使用不同的 `node_id`。新部署不要与 `peers` / `key` 混用。

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

### 旧主动上报探针（仅兼容）

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

### 旧模式的 TLS

上报默认验证 HTTPS 证书，不提供关闭验证的开关。私有 CA 可通过 `ca_file` 配置 PEM 证书文件，相对路径以配置目录为基准。

面板自动生成的自签名证书只包含 `localhost`，不能直接作为远程域名的有效证书。真实多机部署应使用有正确域名/IP SAN 的证书，或现有 HTTPS 反向代理。使用私有 CA 时，既需要信任该 CA，也需要证书名称匹配候选地址。

仅本机回环 HTTP 测试无需额外开关；其他明文 HTTP origin 必须显式设置 `allow_http: true`。该开关只允许上报客户端使用 HTTP，不会把现有管理端口改成明文 HTTP。不要在公网通过 HTTP 分发或发送集群密钥。

## 旧模式的简化选主与故障切换

这一节仅适用于已有 `candidate` / `candidates` 配置，新模式不进行选主。旧模式选择的是**只读优先汇总节点，不是共识主节点**：

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
- `GET /api/v1/monitor/cluster`：只读汇总信息；新模式返回 `coordination: manual`、`peer_count`、`healthy_peers`，旧模式保留候选信息。
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

新模式由汇总端读取探针独立端口的 `GET /api/v1/probe/report`，该端口只有这一个读取路由。普通 LLM/页面仍通过汇总端的 JSON 和 SSE API 查看数据，不需要实现探针签名协议。公开 API 不返回共享密钥、探针 IP 列表或候选地址。

## 前端性能与开发

- 实时数据只使用一条 SSE 连接，初始帧就是当前快照，不额外重复请求节点列表；连续帧合并到浏览器下一次绘制。
- 节点对象和排序后的列表在身份/顺序未变化时保持引用，只更新发生变化的指标。
- 地图实例不进入 Alpine 深度响应式代理；只有位置、名称、筛选或在线区域变化时更新地图，CPU/流量变化不再重建标记或重新测量容器。
- 切到后台标签时关闭 SSE，恢复时重新获取最新状态；关闭详情时取消历史请求并释放详情视图。
- 列表每页最多渲染 50 台，地图和总计仍使用所有节点。保留 daisyUI 的布局、分页和主题组件，不手写一套 UI 框架。

CSS 构建工具位于 `frontend/`。修改监控模板/脚本的类名后运行 `npm ci --ignore-scripts` 和 `npm run build:monitor`，提交生成的 `assets/css/observatory.min.css`。`npm run test:monitor` 验证节点引用、分页、地图更新、SSE合并及演示代码移除。普通 Go 构建直接嵌入现成资源，不需要 Node.js。

## 数据边界

- 没有持久化监控历史、账期流量或节点库存；候选重启后靠新的报告重建缓存，此前离线的节点不会凭空恢复。
- 总流量是当前报告的接口累计计数，通常自系统/网卡启动以来累计；不是月度账单。接口重置会使计数下降，速率重新建立基线。
- 默认接口总计可能包含虚拟、隧道或回环接口；需要真实出口统计时显式指定 `interfaces`，不要重复计入同一流量路径。
- 公开 SSE 并发有上限，HTTP 读取及上报分别限速。只读不等于没有数据泄露或资源消耗风险。
- 地图位置由配置提供，不自动发送服务器 IP 到 GeoIP 服务。缺少国家或坐标时仍显示节点，不制造地理位置。

后续其他前端页可逐步复用框架主题和布局；这次不同时重写所有管理模块。
