# MediaCrawler Go 转写进度与 TODO

本文用于对齐 Python 版 `/home/zjx/code/mine/MediaCrawler` 的能力，并跟踪 Go 版 `/home/zjx/code/mine/media-crawler-go` 的转写进度。

## 已完成（Go）

### 通用
- [x] CLI 入口：按平台启动爬虫（xhs/douyin）。
- [x] YAML/环境变量配置加载（viper）。
- [x] JSON Lines / CSV 追加写存储（notes/comments）。
- [x] 简单媒体下载（HTTP GET 并发）。

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

## 进行中（Go）
- [x] README 与现状对齐（Comment/Media/CDP 勾选项、配置说明）。

## 待转写（高优先）

### 小红书（xhs）
- [x] 登录方式对齐：qrcode/phone/cookie（qrcode/phone 走浏览器手动登录等待，cookie 走 COOKIES 校验）。
- [x] Creator 信息抓取（HTML 解析 `__INITIAL_STATE__`）与存储。
- [x] 存储进一步对齐：按 note_id 分目录落媒体；评论/内容去重（jsonl/csv + idx）。
- [ ] （可选）DB/SQLite/Mongo：持久化与全局去重。

### 抖音（douyin）
- [x] detail 模式（最小闭环）：解析 aweme_id，调用 detail API，按 note_id 落盘。
- [x] a_bogus 生成（detail）：移植 `libs/douyin.js` 并用 GoJS 计算签名。
- [ ] 实现三模式（search/detail/creator）：补齐 search/creator。
- [x] 评论抓取与去重落盘（detail）：comment list + reply（可选）。
- [x] 媒体下载（video/cover）。

### 其他平台（bili / wb / tieba / zhihu / ks）
- [ ] 平台骨架与 client/login/core/store 分层。
- [ ] 签名/参数策略迁移（如 zhihu.js 等）。

## 待转写（中优先）
- [ ] WebUI / API（对齐 Python FastAPI 版）：启动/管理 crawler、状态输出。
- [ ] 词云等增值能力（对齐 Python main.py 的可选功能）。
- [ ] 更完善的错误处理与重试/限速策略（HTTP、下载、签名失败）。

## 本仓库的下一批建议任务（推荐顺序）
- [ ] 完善 xhs：代理池（先把抗风控能力补齐）。
- [ ] 补齐 douyin：search/creator + 评论/媒体下载。
- [ ] 抽象平台接口：统一 crawler/client/store 的编排与并发控制。
