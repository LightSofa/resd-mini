2026-4-1
-----------
• 已完成冒烟测试，结果如下（通过 SSH 在网关本机执行）：

  1. GET /api/v1/health：通过，返回 status=ok。
  2. GET /api/v1/config：通过，确认 Host=0.0.0.0、Port=8899。
  3. GET /api/v1/resources：通过。
  4. POST /api/v1/clear：通过。
  5. POST /api/v1/download：通过（任务创建成功）。
  6. GET /api/v1/resource?id=...：通过（可查到任务）。
  7. POST /api/v1/cancel：已修复后行为正确，完成后任务返回 task not found，不再误删已完成文件。

  本次为测试而修复的代码：

  - /E:/project/resd-mini/core/resource.go:214（下载完成后从 tasks 移除，避免“已完成任务仍可被 cancel”）

  部署与测试脚本（已可跑通）：

  - /E:/project/resd-mini/deploy_istoreos_hardened.ps1
  - /E:/project/resd-mini/build/istoreos/resd-mini.init
