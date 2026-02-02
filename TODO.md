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
- [ ] 平台编排抽象：internal/crawler 目前仅有 interface，cmd 仍按 switch 分发。
- [ ] 更完善的错误处理：统一重试/限速/风控提示与可观测性（日志/metrics）。

## 待转写（P0，高优先）
- [ ] 平台接口与注册表：统一 Search/Detail/Creator 的输入输出与并发控制。
- [x] 存储后端：SQLite（纯 Go）落地，并支持全局去重（STORE_BACKEND=sqlite，SQLITE_PATH）。
- [ ] Web API（对齐 Python FastAPI 能力）：启动/停止任务、状态查询、运行参数校验。
- [ ] 其他平台骨架：bili / wb / tieba / zhihu / ks（先跑通 detail + store，再补 search/creator）。

## 待转写（P1，中优先）
- [ ] WebUI：可视化配置、任务管理、日志与数据预览。
- [ ] 词云等增值能力：评论词云（读取 comments 数据生成图片）。
- [ ] 测试体系：按平台/模式的可回放测试（mock HTTP + 签名模块单测）。

## 开发任务清单（可执行）
- [x] T-001 抽象 Platform 接口 + registry，并改造 cmd 入口。
- [x] T-002 HTTP 重试/退避 + 代理失效切换（xhs/douyin）。
- [x] T-003 落地 SQLite 存储（notes/comments/creators）与全局去重。
- [x] T-004 增加 API Server：/healthz、/run、/stop、/status。
- [x] T-005 补齐其他平台：先实现 B 站 detail 最小闭环。
- [x] T-006 补齐其他平台：再实现 微博 detail 最小闭环。
- [x] T-007 增加统一 logger（结构化）并替换 fmt.Printf。
