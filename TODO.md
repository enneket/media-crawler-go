# MediaCrawler Go 转写进度与 TODO

本文用于对齐 Python 版 `/home/zjx/code/mine/MediaCrawler` 的能力，并跟踪 Go 版 `/home/zjx/code/mine/media-crawler-go` 的转写进度。

## 已完成（Go）

### 通用
- [x] CLI 入口：按平台启动爬虫（xhs/douyin）。
- [x] YAML/环境变量配置加载（viper）。
- [x] HTTP 配置：超时/重试/退避（HTTP_TIMEOUT_SEC / HTTP_RETRY_*）。
- [x] JSON Lines / CSV 追加写存储（notes/comments）。
- [x] 简单媒体下载（HTTP GET 并发）。
- [x] 代理池：支持失效切换（rate-limit/403 时 InvalidateCurrent，下次请求自动换 IP）。
- [x] Data API：浏览/预览/下载 data 目录文件（/data/files /data/download /data/stats）。

### 小红书（xhs）
- [x] Playwright persistent context 启动与登录态复用（`USER_DATA_DIR`）。
- [x] CDP 模式（对齐 Python `tools/cdp_browser.py`）：可启动/复用远程调试端口并 `connect_over_cdp`。
- [x] 代理池（对齐 Python `proxy/*`）：支持 kuaidaili / wandouhttp，从环境变量读取密钥。
- [x] Cookie 注入（`COOKIES`）与从浏览器同步 Cookie 到 HTTP Header。
- [x] 签名：通过浏览器上下文计算 `X-S/X-T/x-S-Common/X-B3-Traceid`。
- [x] 三模式：search / detail / creator（creator 支持 cursor 翻页）。
- [x] search 分页与最大笔记数限制（`START_PAGE`、`CRAWLER_MAX_NOTES_COUNT`）。
- [x] 评论分页抓取与最大评论数限制（`CRAWLER_MAX_COMMENTS_COUNT_SINGLENOTES`）。
- [x] 二级评论抓取（`ENABLE_GET_SUB_COMMENTS`）。
- [x] 并发抓取（`MAX_CONCURRENCY_NUM`，签名调用已串行化以避免 Playwright 并发 Evaluate 风险）。
- [x] 请求重试：429/5xx 自动退避重试；重试会重新签名，避免 X-T 过期。

### 抖音（douyin）
- [x] 三模式（search/detail/creator）。
- [x] detail：解析 aweme_id，调用 detail API，按 note_id 落盘。
- [x] a_bogus 生成：移植 `douyin.js` 并用 GoJS 计算签名。
- [x] 评论抓取与去重落盘（可选二级评论）。
- [x] 媒体下载（video/cover）。
- [x] 请求重试：基于 HTTP_RETRY_*（429/5xx）自动退避重试。

## 进行中（Go）
- [x] 平台编排抽象：internal/crawler + platform registry 统一分发（Runner/Request/Result）。
- [x] 更完善的错误处理：统一重试/限速/风控提示与可观测性（日志/metrics；已具备 risk_hint + failure_kinds 统计，且 403/429 分别归类 forbidden/rate_limited）。
- [x] HTTP 错误结构化：统一用 crawler.NewHTTPStatusError 封装状态码与 body snippet，便于分类统计与告警。

## 待转写（P0，高优先）
- [x] 平台接口与注册表：统一 Search/Detail/Creator 的输入输出与并发控制。
- [x] 存储后端：SQLite（纯 Go）落地，并支持全局去重（STORE_BACKEND=sqlite，SQLITE_PATH）。
- [x] Web API（对齐 Python FastAPI 能力）：启动/停止任务、状态查询、运行参数校验。
- [x] 其他平台骨架：tieba / zhihu / ks（先跑通 detail + store，再补 search/creator；bili/wb detail 已完成）。

## 待转写（P1，中优先）
- [x] WebUI：可视化配置、任务管理、日志与数据预览（内置静态页 + API）。
- [x] WebSocket：推送 logs/status（对齐 Python `/api/ws/logs`、`/api/ws/status`）。
- [x] 词云等增值能力：评论词云（读取 comments 数据生成图片）。
- [x] 测试体系：按平台/模式的可回放测试（mock HTTP + 签名模块单测）。
- [x] 存储后端扩展：Excel（XLSX）。
- [x] 存储后端扩展：MySQL/Postgres。
- [ ] 存储后端扩展：MongoDB（按需）。
- [x] Cache 抽象：memory/redis（用于跨流程去重、签名/代理缓存等按需场景）。

## 与 Python 版差异（待补齐）

### API/WebUI
- [x] 配置接口：/config/platforms、/config/options（对齐 Python WebUI 动态渲染所需的选项接口）。
- [x] 环境自检：/env/check（对齐 Python 版“命令行 --help”自检逻辑）。
- [x] WebUI 静态托管：挂载前端产物，根路径返回 index.html。
- [x] WebSocket：日志流与状态流。

### 平台覆盖
- [x] 将 zhihu/ks 从“detail 最小闭环”扩展到 search/creator（先补参数与落盘，再补分页/并发与风控；weibo、bilibili、tieba 已补齐）。

### 存储/数据
- [x] 输出目录可配置（DATA_DIR；同时影响 store 落盘与 /data API）。
- [ ] Excel 与更多 DB 后端（按实际需求取舍）。

## 开发任务清单（可执行）
- [x] T-001 抽象 Platform 接口 + registry，并改造 cmd 入口。
- [x] T-002 HTTP 重试/退避 + 代理失效切换（xhs/douyin）。
- [x] T-003 落地 SQLite 存储（notes/comments/creators）与全局去重。
- [x] T-004 增加 API Server：/healthz、/run、/stop、/status。
- [x] T-005 补齐其他平台：先实现 B 站 detail 最小闭环。
- [x] T-006 补齐其他平台：再实现 微博 detail 最小闭环。
- [x] T-007 增加统一 logger（结构化）并替换 fmt.Printf。
- [x] T-008 增加 Data API：/data/files、/data/files/{path} 预览、/data/download、/data/stats。
- [x] T-009 WebSocket：/ws/logs、/ws/status（内置任务执行器直接推送，无需子进程）。
- [x] T-010 WebUI：静态资源托管 + 配置页 + 任务页（复用 T-008/T-009）。
- [x] T-011 扩展更多平台能力：bili/tieba/zhihu/ks 的 search/creator。
- [x] T-011a 微博：search/creator（m.weibo.cn /api/container/getIndex + /statuses/show + creator profile）。
- [x] T-011b bilibili：search/creator（搜索接口 + up 投稿列表 + view 详情保存）。
- [x] T-011c tieba：search/creator（/f/search/res 抓列表 + /home/main 抓用户帖子列表 + 归档详情页）。
- [x] T-012 存储扩展：Excel（可选）与输出目录配置。
- [x] T-013 测试体系：可回放的 HTTP fixture（签名/接口解析稳定性）。
- [x] T-014 存储扩展：MySQL/Postgres（SQL schema + upsert + comments 去重写入）。
