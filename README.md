## res-mini
## 本项目为[res-downloader](https://github.com/putyy/res-downloader)项目的精简版，使用系统默认浏览器展示UI界面，核心功能与res-downloader一致！
### 🎉 爱享素材下载器

> 一款基于 Go 的跨平台资源下载工具，简洁易用，支持多种资源嗅探与下载。

---

## ✨ 功能特色

- 🚀 **简单易用**：操作简单，界面清晰美观
- 🖥️ **多平台支持**：Windows / macOS / Linux
- 🌐 **多资源类型支持**：视频 / 音频 / 图片 / m3u8 / 直播流等
- 📱 **平台兼容广泛**：支持微信视频号、小程序、抖音、快手、小红书、酷狗音乐、QQ音乐等
- 🌍 **代理抓包**：支持设置代理获取受限网络下的资源

---

## 📚 文档 & 版本

- 📘 [在线文档](https://res.putyy.com/)
- 🧩 [Go wails版(推荐)](https://github.com/putyy/res-downloader) ｜[Mini版 UI使用默认浏览器展示](https://github.com/putyy/resd-mini) ｜ [Electron旧版 支持Win7](https://github.com/putyy/res-downloader/tree/old)
- 💬 [加入交流群](https://www.putyy.com/app/admin/upload/img/20250418/6801d9554dc7.webp)
  > *群满时可加微信 `AmorousWorld`，请备注“来源”*

---

## 🧩 下载地址

- 🆕 [蓝奏云下载 密码:ftlv](https://wwjv.lanzoum.com/b00l1q2mnc)
- 🆕 [GitHub 下载](https://github.com/putyy/resd-mini/releases)
--- 

## 🚀 使用方法

> 请按以下步骤操作以正确使用软件：

1. 安装时务必 **允许安装证书文件** 并 **允许网络访问**
2. 打开软件 → 首页左上角点击 **“启动代理”**
3. 选择要获取的资源类型（默认全部）
4. 在外部打开资源页面（如视频号、小程序、网页等）
5. 返回软件首页，即可看到资源列表

---

## 🖼️ 软件截图

![软件截图](https://raw.githubusercontent.com/putyy/res-downloader/master/docs/images/show.webp)

---

## ❓ 常见问题

### 📺 m3u8 视频资源

- 在线预览：[m3u8play](https://m3u8play.com/)
- 视频下载：[m3u8-down](https://m3u8-down.gowas.cn/)

### 📡 直播流资源

- 推荐使用 [OBS](https://obsproject.com/) 进行录制（教程请百度）

### 🐢 下载慢、大文件失败？

- 推荐工具：
  - [Neat Download Manager](https://www.neatdownloadmanager.com/index.php/en/)
  - [Motrix](https://motrix.app/download)
- 视频号资源下载后可在操作项点击 `视频解密（视频号）`

### 🧩 软件无法拦截资源？

- 检查是否正确设置系统代理：  
  地址：127.0.0.1
  端口：8899

### 🌐 关闭软件后无法上网？

- 手动关闭系统代理设置

### 🧠 更多问题

- [GitHub Issues](https://github.com/putyy/res-downloader/issues)
- [爱享论坛讨论帖](https://s.gowas.cn/d/4089)

---

## 💡 实现原理 & 初衷

本工具通过代理方式实现网络抓包，并筛选可用资源。与 Fiddler、Charles、浏览器 DevTools 原理类似，但对资源进行了更友好的筛选、展示和处理，大幅度降低了使用门槛，更适合大众用户使用。

---

## ⚠️ 免责声明

> 本软件仅供学习与研究用途，禁止用于任何商业或违法用途。  
如因此产生的任何法律责任，概与作者无关！

---

## 🧱 iStoreOS / OpenWrt 常驻后台运行

Linux 构建版本已支持无 GUI 常驻模式，直接以前台进程运行即可由 `procd` 托管为后台服务。

### 1. 编译（示例）

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o resd-mini
```

### 2. 部署到网关

```bash
scp .\\resd-mini root@iStoreOS:/usr/bin/resd-mini
scp .\\build\\istoreos\\resd-mini.init root@iStoreOS:/etc/init.d/resd-mini
scp .\\build\\istoreos\\resd-mini.config.example root@iStoreOS:/etc/config/resd-mini
```

### 3. 启用服务

```bash
ssh root@iStoreOS
chmod +x /etc/init.d/resd-mini
mkdir -p /root/downloads/resd-mini
/etc/init.d/resd-mini enable
/etc/init.d/resd-mini start
logread -f | grep resd-mini
```

服务脚本通过环境变量控制监听与下载路径：

- `RESD_HOST`（建议网关场景设为 `0.0.0.0`）
- `RESD_PORT`（默认 `8899`）
- `RESD_SAVE_DIR`

---

## 🔌 Web API（自动化脚本）

自动化接口统一前缀：`/api/v1/*`

OpenAPI 规范文件：[`openapi.yaml`](./openapi.yaml)

### 接口清单

| Method | Path | 说明 |
|---|---|---|
| GET | `/api/v1/health` | 健康检查 |
| GET | `/api/v1/config` | 读取当前配置 |
| POST/PUT | `/api/v1/config` | 更新配置 |
| GET | `/api/v1/proxy/status` | 查询系统代理是否开启 |
| POST | `/api/v1/proxy/open` | 开启系统代理 |
| POST | `/api/v1/proxy/unset` | 关闭系统代理 |
| GET | `/api/v1/resources` | 获取资源列表 |
| GET | `/api/v1/resource?id=<id>` | 获取单个资源详情 |
| POST | `/api/v1/download` | 创建下载任务 |
| POST | `/api/v1/cancel` | 取消下载任务 |
| POST | `/api/v1/clear` | 清空资源缓存 |
| POST | `/api/v1/delete` | 按 `urlSign` 删除资源 |
| POST | `/api/v1/set-type` | 设置抓取类型过滤 |
| POST | `/api/v1/wx-file-decode` | 视频号文件解密 |

### 通用约定

- Base URL：`http://127.0.0.1:8899`
- Content-Type：`application/json`（GET 可省略）
- 返回结构固定：

```json
{
  "code": 1,
  "message": "ok",
  "data": {}
}
```

- `code=1` 成功，`code=0` 失败。
- 当前实现大多数情况下 HTTP 状态码为 `200`，应优先根据 `code/message` 判定业务是否成功。

### `data` 字段模型说明

#### 1) Config（`/api/v1/config` 的 data）

| 字段 | 类型 | 作用 |
|---|---|---|
| Theme | string | 主题名，如 `lightTheme` |
| Locale | string | 语言，如 `zh`/`en` |
| Host | string | 本地服务监听地址，默认 `127.0.0.1` |
| Port | string | 本地服务端口，默认 `8899` |
| Quality | number | 视频号清晰度：`0`默认、`1`超清、`2`高、`3`中、`4`低 |
| SaveDirectory | string | 下载保存目录 |
| FilenameLen | number | 用描述生成文件名时的最大长度，`0` 表示不限制该规则 |
| FilenameTime | boolean | 下载文件名是否追加时间戳 |
| UpstreamProxy | string | 上游代理地址，如 `http://127.0.0.1:7890` |
| OpenProxy | boolean | 抓包转发是否使用上游代理 |
| DownloadProxy | boolean | 下载任务是否使用上游代理 |
| AutoProxy | boolean | 程序启动时是否自动开启系统代理 |
| WxAction | boolean | 视频号抓取模式：全量拦截或仅详情拦截 |
| TaskNumber | number | 单任务分片下载并发数（连接数） |
| DownNumber | number | 同时下载任务数量（队列并发） |
| UserAgent | string | 下载请求默认 UA |
| UseHeaders | string | 下载时允许透传的请求头；`default` 表示内置安全过滤策略 |
| InsertTail | boolean | 新资源是否追加到列表尾部 |
| MimeMap | object | MIME 到资源类型与后缀映射 |
| Rule | string | 抓包域名规则，多行文本，支持 `*`、`*.qq.com`、`!static.qq.com` |

`MimeMap` 子项结构：

| 字段 | 类型 | 作用 |
|---|---|---|
| Type | string | 资源分类（如 `video`/`audio`/`image`/`m3u8`/`live`） |
| Suffix | string | 保存时使用的文件后缀（如 `.mp4`） |

#### 2) MediaInfo（资源对象）

`/api/v1/resources` 返回数组项、`/api/v1/resource` 的 `data`、`/api/v1/download`/`cancel`/`wx-file-decode` 请求体都使用该结构（可按接口需要只传部分字段）。

| 字段 | 类型 | 作用 |
|---|---|---|
| Id | string | 资源唯一 ID，下载/取消任务的主键 |
| Url | string | 资源下载链接 |
| UrlSign | string | 资源签名（URL 的 md5），用于去重与删除 |
| CoverUrl | string | 封面图 URL（部分资源有值） |
| Size | number | 资源大小（字节），未知时可能为 `0` |
| Domain | string | 资源所属主域名 |
| Classify | string | 资源分类，如 `image`/`audio`/`video`/`m3u8`/`live` 等 |
| Suffix | string | 文件后缀，如 `.mp4` |
| SavePath | string | 下载完成后的本地路径 |
| Status | string | 状态：`ready`/`running`/`error`/`done`/`handle` |
| DecodeKey | string | 视频号解密 key（有值时可用于解密） |
| Description | string | 资源描述，用于辅助命名 |
| ContentType | string | 原始 MIME 类型 |
| OtherData | object | 扩展数据字典 |

`OtherData` 常见键：

| 键名 | 类型 | 作用 |
|---|---|---|
| headers | string(JSON) | 抓包时记录的请求头 JSON 字符串，下载时可回放请求头 |
| wx_file_formats | string | 视频号可选清晰度标识，`#` 分隔 |
| last_message | string | 最近一次下载状态消息 |

### 每个 API 具体用法

#### `GET /api/v1/health`

作用：健康检查与基础状态探测。  
请求体：无。

`data` 字段：

| 字段 | 类型 | 作用 |
|---|---|---|
| status | string | 固定 `ok` |
| name | string | 应用名 |
| port | string | 当前服务端口 |
| proxy | boolean | 当前系统代理是否已开启 |

```bash
curl http://127.0.0.1:8899/api/v1/health
```

#### `GET /api/v1/config`

作用：获取完整配置。  
请求体：无。  
响应 `data`：完整 `Config` 对象（见上文字段表）。

```bash
curl http://127.0.0.1:8899/api/v1/config
```

#### `POST|PUT /api/v1/config`

作用：更新配置。  
请求体：`Config` 对象。  
注意：必须提交完整对象，不支持局部 patch；缺失字段可能被覆盖为零值。

```bash
curl -X PUT http://127.0.0.1:8899/api/v1/config \
  -H "Content-Type: application/json" \
  -d '{
    "Theme":"lightTheme",
    "Locale":"zh",
    "Host":"127.0.0.1",
    "Port":"8899",
    "Quality":0,
    "SaveDirectory":"C:/Users/xxx/Downloads",
    "FilenameLen":20,
    "FilenameTime":true,
    "UpstreamProxy":"",
    "OpenProxy":false,
    "DownloadProxy":false,
    "AutoProxy":false,
    "WxAction":true,
    "TaskNumber":8,
    "DownNumber":3,
    "UserAgent":"Mozilla/5.0 ...",
    "UseHeaders":"default",
    "InsertTail":true,
    "MimeMap":{
      "video/mp4":{"Type":"video","Suffix":".mp4"}
    },
    "Rule":"*"
  }'
```

#### `GET /api/v1/proxy/status`

作用：查询系统代理状态。  
请求体：无。  
响应 `data` 字段：

| 字段 | 类型 | 作用 |
|---|---|---|
| value | boolean | `true` 表示系统代理已开启 |

```bash
curl http://127.0.0.1:8899/api/v1/proxy/status
```

#### `POST /api/v1/proxy/open`

作用：开启系统代理。  
请求体：无。  
响应 `data.value`：当前代理状态。

```bash
curl -X POST http://127.0.0.1:8899/api/v1/proxy/open
```

#### `POST /api/v1/proxy/unset`

作用：关闭系统代理。  
请求体：无。  
响应 `data.value`：当前代理状态。

```bash
curl -X POST http://127.0.0.1:8899/api/v1/proxy/unset
```

#### `GET /api/v1/resources`

作用：获取当前资源列表。  
请求体：无。  
响应 `data` 字段：

| 字段 | 类型 | 作用 |
|---|---|---|
| items | MediaInfo[] | 当前已捕获或已记录的资源数组 |

```bash
curl http://127.0.0.1:8899/api/v1/resources
```

#### `GET /api/v1/resource?id=<id>`

作用：按 ID 获取单条资源。  
查询参数：

| 参数 | 必填 | 说明 |
|---|---|---|
| id | 是 | 资源 ID（`MediaInfo.Id`） |

成功时 `data` 为 `MediaInfo`；`id` 缺失或不存在会返回 `code=0`。

```bash
curl "http://127.0.0.1:8899/api/v1/resource?id=demo-id"
```

#### `POST /api/v1/download`

作用：创建下载任务。  
请求体：`MediaInfo + decodeStr`。

| 字段 | 必填 | 说明 |
|---|---|---|
| Id | 建议是 | 任务 ID，取消任务依赖该字段 |
| Url | 是 | 资源地址 |
| Suffix | 建议是 | 文件后缀，影响保存名 |
| Description | 否 | 用于生成保存文件名 |
| OtherData.headers | 否 | JSON 字符串，下载请求可回放抓包头 |
| decodeStr | 否 | base64 字符串；提供后会在下载完成后执行解密 |

```bash
curl -X POST http://127.0.0.1:8899/api/v1/download \
  -H "Content-Type: application/json" \
  -d '{
    "Id":"demo-id",
    "Url":"https://example.com/a.mp4",
    "Suffix":".mp4",
    "Description":"demo-video",
    "OtherData":{
      "headers":"{\"Referer\":[\"https://example.com\"],\"User-Agent\":[\"Mozilla/5.0\"]}"
    },
    "decodeStr":""
  }'
```

#### `POST /api/v1/cancel`

作用：取消下载任务。  
请求体最小只需：

| 字段 | 必填 | 说明 |
|---|---|---|
| Id | 是 | 需要取消的任务 ID |

```bash
curl -X POST http://127.0.0.1:8899/api/v1/cancel \
  -H "Content-Type: application/json" \
  -d '{"Id":"demo-id"}'
```

#### `POST /api/v1/clear`

作用：清空资源缓存（列表与索引）。  
请求体：无。

```bash
curl -X POST http://127.0.0.1:8899/api/v1/clear
```

#### `POST /api/v1/delete`

作用：按 `urlSign` 删除指定资源。  
请求体字段：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| sign | string[] | 是 | 资源签名数组（对应 `MediaInfo.UrlSign`） |

```bash
curl -X POST http://127.0.0.1:8899/api/v1/delete \
  -H "Content-Type: application/json" \
  -d '{"sign":["url-sign-1","url-sign-2"]}'
```

#### `POST /api/v1/set-type`

作用：设置抓取过滤类型。  
请求体字段：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| type | string | 是 | 逗号分隔类型，如 `all`、`video,audio,image` |

说明：可用类型来自当前 `MimeMap` 的 `Type` 集合，默认包含 `all`。

```bash
curl -X POST http://127.0.0.1:8899/api/v1/set-type \
  -H "Content-Type: application/json" \
  -d '{"type":"video,audio"}'
```

#### `POST /api/v1/wx-file-decode`

作用：对本地文件执行视频号解密。  
请求体字段：

| 字段 | 必填 | 说明 |
|---|---|---|
| filename | 是 | 待解密本地文件路径（通常 `.mp4`） |
| decodeStr | 是 | base64 编码的解密字节串 |
| 其他 MediaInfo 字段 | 否 | 可传可不传 |

响应 `data` 字段：

| 字段 | 类型 | 作用 |
|---|---|---|
| save_path | string | 解密后文件路径（通常为 `*_decrypt.mp4`） |

```bash
curl -X POST http://127.0.0.1:8899/api/v1/wx-file-decode \
  -H "Content-Type: application/json" \
  -d '{
    "filename":"C:/Users/xxx/Downloads/source.mp4",
    "decodeStr":"BASE64_DECODE_KEY"
  }'
```

### 建议调用流程

1. `POST /api/v1/proxy/open` 开启代理并开始抓取。
2. 轮询 `GET /api/v1/resources` 获取资源。
3. 选择资源后调用 `POST /api/v1/download`。
4. 需要中止时调用 `POST /api/v1/cancel`。
5. 按需调用 `POST /api/v1/delete` 或 `POST /api/v1/clear` 清理记录。
