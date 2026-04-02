# AGENTS.md

本文件用于指导 Codex 在 `resd-mini` 项目中进行测试、调试与维护，优先保证 iStoreOS/OpenWrt 网关场景可用。

## 1. 项目目标与关键约束

- 后端为 Go，前端为 Vue3 + Vite。
- `main.go` 使用 `//go:embed all:web/dist`，所以 **后端编译前必须先生成 `web/dist`**。
- 当前主要运行场景：iStoreOS/OpenWrt 网关设备（无 GUI、无 gsettings、通常无 sudo）。
- 线上可用优先：修复必须以“网关可部署 + API 可调用 + 前端可操作”为验收标准。

## 2. 本地开发与构建流程

1. 前端依赖安装（首次或依赖变更后）
   - `cd web`
   - `npm install`
2. 前端构建（生成 `web/dist`）
   - `npm run build-only`
3. 返回根目录构建后端

## 3. iStoreOS 部署标准流程

使用仓库根目录脚本：

- `./deploy_istoreos_hardened.ps1 -Router "root@192.168.1.254"`

脚本职责：

- 自动识别网关架构并编译
- 上传二进制与 init/config
- CRLF 转 LF 兼容处理
- 启停服务并健康检查
- 校验上传文件哈希

## 4. 已踩坑总结（必须记住）

### 4.1 Linux/iStoreOS 兼容坑

- `sudo` 不存在：Linux 命令执行需在 `os.Geteuid()==0` 时直接执行，不要强制 `sudo -S`。
- `gsettings` 不存在：网关模式下 `setProxy/unsetProxy` 必须 no-op 成功。
- `update-ca-certificates` 不存在：OpenWrt/iStoreOS 证书安装不走 Debian 机制；网关模式允许 `/api/install` 直接成功返回，客户端通过 `/api/cert` 下载证书。
- init 脚本 CRLF 会导致 ash 语法异常（`expecting "}"`）：部署后必须做 `sed -i 's/\r$//'`。

### 4.2 运行时行为坑

- 下载完成状态被覆盖：进度协程可能晚于完成事件写入 `running`，需等待进度协程收敛。
- 未知 `Content-Length` 的进度百分比可能异常（负值/无意义）：`totalSize <= 0` 时不要算 `%`，改显示字节数。
- 已完成任务不应再可 cancel：完成后需从 `tasks` 映射移除。
- `/api/*` 返回体语义：`code=1` 才是成功，`code=0` 为失败；前端提示与测试断言必须按此语义判断。

### 4.3 前端交互坑

- `navigator.clipboard` 在非 HTTPS/IP 场景可能失效：必须提供 `execCommand('copy')` 回退。
- 复制类功能统一走 `copyToClipboard`，禁止散落直接调用 `navigator.clipboard.writeText`。

### 4.4 Windows 测试与命令坑

- 在 PowerShell + SSH 场景下，用 `curl -d '{\"k\":\"v\"}'` 这类写法容易被二次转义污染，导致后端报 JSON 解析错误（如 `invalid character '\\' ...`）。
- 需要发 JSON 请求时，优先使用 `Invoke-RestMethod` + `ConvertTo-Json -Compress`，避免转义问题。
- 若必须在路由器上发 JSON，优先 `curl -H 'Content-Type: application/json' --data-binary ...`，并先单独验证 payload。

## 5. 调试方法（推荐顺序）

1. 先看 API 健康：`/api/v1/health`
2. 再看服务状态与日志：
   - `/etc/init.d/resd-mini status`
   - `logread -f | grep resd-mini`
3. 再看配置是否生效：`/api/v1/config`（重点看 Host/Port/SaveDirectory）
4. 若前端异常，先排除缓存：浏览器强刷（Ctrl+F5）
5. 若部署成功但外部访问失败，优先用网关本机 loopback 先验证服务是否正常

## 6. 代码维护守则

- 小步提交：一次修复只解决一类问题，便于回归。
- 不破坏桌面平台行为：Linux 网关兼容分支需与 Windows/macOS 逻辑隔离。
- 接口向后兼容：保留原 `/api/*`，新增能力放 `/api/v1/*`。
- 每次改动后至少执行：
  - `npm run build-only`
  - `go build ./...`
  - 部署脚本
  - 健康检查 + 1 条下载链路验证

## 7. 常用命令清单

- 前端构建：`cd web && npm run build-only`
- 后端构建：`go build ./...`
- 一键部署：`./deploy_istoreos_hardened.ps1 -Router "root@192.168.1.254"`
- 健康检查：`curl -sS http://127.0.0.1:8899/api/v1/health`
- 配置查看：`curl -sS http://127.0.0.1:8899/api/v1/config`
- 日志查看：`logread -f | grep resd-mini`
- Windows JSON 请求（推荐）：
  - `$body = @{ content = 'smoke-export' } | ConvertTo-Json -Compress`
  - `Invoke-RestMethod -Method Post -Uri 'http://192.168.1.254:8899/api/batch-export' -ContentType 'application/json' -Body $body`

## 8. 对 Codex 的执行要求

- 任何变更前先确认是否影响 iStoreOS 网关模式。
- 任何“已修复”必须附带可复现验证步骤或实际命令输出。
- 遇到网络/权限限制时，优先切换到“SSH 到网关本机执行测试”。
- 发现新错误后，先最小化复现，再修复，再回归冒烟，最后更新本文件。
