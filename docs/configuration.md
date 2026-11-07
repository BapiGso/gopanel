# 配置存储

GoPanel 自身的设置使用强类型 Go 结构体和 JSON 文件保存，不依赖 Koanf，也不引入面板数据库。Caddy 和 FRP 继续保存、校验其原生配置文本。

## 配置目录与启动

程序入口显式调用 `core.Initialize(directory)`，完成面板设置及应用配置文件的加载、缺失文件的创建。导入配置包或 Caddy/FRP 模块不会自动读写配置或启动服务。初始化失败会输出错误并退出，损坏文件不会被默认配置覆盖。

通过 `GOPANEL_CONFIG_DIR` 指定配置目录：

```powershell
$env:GOPANEL_CONFIG_DIR = 'C:\GoPanel\config'
.\gopanel.exe
```

未设置时使用启动时的工作目录以兼容旧部署；路径只解析一次，随后改变工作目录不会改变配置读写位置。已有配置不会自动搬迁，指定新目录前需自行复制已有文件。

该目录包含：

| 文件 | 内容 |
| --- | --- |
| `gopanel_config.json` | 面板账号、监听地址、安全入口、WebDAV 设置、服务自动启动偏好 |
| `gopanel_Caddyfile` | 原生 Caddyfile |
| `gopanel_frps.conf` | 现有 FRPS 原生配置 |
| `gopanel_frpc.conf` | 现有 FRPC 原生配置 |
| `gopanel_headscale.json` | 现有实验性 Headscale 配置 |

这个变量只指定上述配置的位置，不改变证书、日志、定时任务脚本、Headscale 数据目录或原生配置内相对路径的解析方式。

服务自动启动在初始化完成、面板准备启动时执行；自动启动失败会记录错误，仍可通过面板修改配置。保存设置或应用配置不会立即启动、停止或重载服务。

## 类型与版本

面板配置结构由 `core/config/config.go` 定义。当前文件的 `schema_version` 为 `1`，原有的 `panel`、`webdav`、`enable` 字段保留。

- 没有版本字段的旧配置按旧格式读取，在内存中迁移到版本 `1`；首次成功保存时才写入版本字段，不在读取时改写文件。
- 缺失的服务自动启动标志和 WebDAV 启用标志默认为 `false`。
- 已有文件缺失必要的面板账号或监听设置会报错，不重新随机生成账号；只有首次创建文件时生成随机凭据。
- 拒绝未知字段、错误类型、显式 `null`、多个 JSON 文档及非法监听地址、安全入口或不完整的启用 WebDAV 账号。
- 不支持的未来版本会明确报错，避免旧程序保存时丢弃新版本字段。

`schema_version` 用于格式兼容；`revision` 是当前文件内容的 SHA-256，用于检查并发写入，二者不是同一个概念。

## 读写流程

进程中只有一份已校验的配置快照，不再同时维护 Koanf 和结构体，也不使用字符串配置键。

- `config.Snapshot()` / `config.Current()`：读取内存中的最近一次有效快照；前者返回配置，后者同时返回修订值。未初始化时立即报错，不静默提供空配置。
- `config.Read()`：读取选定的文件，校验变化并更新内存快照。HTTP 设置读取使用这个入口，因此冲突后重新 GET 可以获得最新值；外部编辑无效时保留最近一次有效内存快照，但返回读取错误。
- `config.Update(revision, callback)`：复制当前配置、合并修改、完整校验、检查磁盘修订、保存成功后发布新快照。回调、校验和保存失败均不发布候选状态。

示例：

```go
state, err := config.Read()
if err != nil {
    return err
}
_, err = config.Update(state.Revision, func(next *config.Config) error {
    next.WebDAV.Enable = false
    return nil
})
return err
```

存储层位于 `core/storage`，独立于 HTTP。它使用同目录临时文件、文件 `Sync`、关闭文件及平台替换操作，写入或替换失败不会先截断旧文件。Windows 使用带替换和写入完成标志的文件移动操作；Unix 使用同文件系统重命名。此实现不承诺多文件事务、所有文件系统上的断电持久性或跨进程事务。

同一进程内，对同一路径的不同存储句柄共享写入锁；修订检查也会发现读取后已完成的外部文件修改。不要同时运行多个面板进程管理同一目录，也不要在 API 保存过程中用不协调的外部写入器修改文件。

新目录使用 `0700`、保存的配置文件使用 `0600`（Unix）。Windows 的这些权限位不能代替 ACL，应限制目录的实际访问权限。账号密码仍按现有方式保存在 JSON 中；此改动不提供密码加密，备份也需要妥善保管。

## HTTP 修订检查

所有修改持久化设置的 `PATCH` 请求都必须带读取时的 `revision`：

- `GET /panel/settings`、`GET /webdav/settings` 返回 `revision`。
- `GET /caddy/status`、`GET /frp/{service}/status` 返回 `settings_revision`，将该值作为设置 PATCH 的 `revision`。
- 这些设置都来自同一个 `gopanel_config.json`，共享一个修订值。其他模块设置的保存也可能使手里的修订过期。
- 原生配置 `/config` 返回的 `revision` 独立于设置修订；`applied_revision` 表示当前服务实际加载的原生配置版本。

过期修订返回 `409 revision_conflict`。重新读取、比较和合并修改后提交新修订，不通过直接刷新修订后盲目重试来覆盖其他修改。没有变化的保存不会无故翻转布尔值；旧修订的重试仍然被拒绝。

完整接口约定见 [Agent HTTP API](http-api.md)。
