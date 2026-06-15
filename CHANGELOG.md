# 更新日志

## 2026-06-15 — Seedance 视频模型调用示例适配

### 背景
京东聚合的 Seedance 视频模型（`drrfsmvr2.0` 等）走异步任务接口 `/v1/video/generations`，
请求体为 `{model, prompt, duration, size}`，与 OpenAI chat 格式不同。
模型广场的「调用示例」原本对所有模型统一套 chat 模板，导致视频模型展示的示例错误，
用户「复制即用」不可行。

### 改动
- **web/default/src/features/pricing/components/model-details-api.tsx**
  - 新增 `buildVideoSample()`：渲染 `/v1/video/generations` 的 cURL / Python / TypeScript / JavaScript
    示例（字段 `model/prompt/duration/size`），URL 用站点「服务器地址(server_address)」自动拼接。
  - 新增 `isVideoEndpoint()`：按端点类型(`video`)、路径(含 `/video/`)、或模型名(`seedance|drrfsmvr`)
    识别视频模型；`buildSample()` 优先走视频模板。
- **web/default/rsbuild.config.ts**
  - `server.port` 支持通过 `DEV_PORT` 环境变量指定（未设时行为不变，仅本地开发便利）。

### 影响
- 视频模型在模型广场显示正确的调用示例，用户复制后换 API Key 与提示词即可调用。
- 非视频模型不受影响。
- 前提：需确认「系统设置 → 服务器地址」为对外公开域名（如 `https://www.ahatime.net`），
  否则示例里的地址会是错误/本地地址。
