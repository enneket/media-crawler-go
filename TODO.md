# MediaCrawler Go 转写进度与 TODO

本文用于对齐 Python 版 `/home/zjx/code/mine/MediaCrawler` 的能力，并跟踪 Go 版 `/home/zjx/code/mine/media-crawler-go` 的转写进度。

## 已完成（Go，对齐 Python）

### 平台覆盖
- [x] 平台注册表 + 内置平台：xhs / douyin / bilibili / weibo / tieba / zhihu / kuaishou。
- [x] 三种模式：search / detail / creator（各平台均有实现入口）。

### CLI / 配置
- [x] CLI 入口：读取 config.yaml（viper）后按 PLATFORM 启动爬虫；支持 `-api` 启动 WebUI/API。
- [x] YAML + 环境变量加载：`MEDIA_CRAWLER_` 前缀覆盖；默认值对齐 Python 核心配置。
- [x] HTTP 基础能力：超时/重试/退避（HTTP_TIMEOUT_SEC / HTTP_RETRY_*）。
- [x] 代理池：kuaidaili / wandouhttp，支持失效切换（403/429 等场景 InvalidateCurrent）。
- [x] Cache：memory / redis / none（用于跨流程去重、词云缓存等）。

### 存储 / 数据
- [x] 文件落地：json(逐行追加 JSON) / CSV（含 BOM）/ XLSX（含 header 校验）。
- [x] DB 后端：SQLite / MySQL / Postgres（自动初始化 schema，notes/creators/comments upsert/去重写入）。
- [x] 输出目录可配置：DATA_DIR（同时影响落盘与 /data API）。
- [x] Data API：文件列表/预览/下载/统计（/data/files /data/files/* /data/download/* /data/stats）。
- [x] 增值能力：评论词云（/data/wordcloud）。

### API / WebUI
- [x] API：/healthz /run /stop /status /config/platforms /config/options /env/check。
- [x] WebSocket：/ws/logs /ws/status。
- [x] WebUI：内置静态资源（embed），根路径返回 index.html。

### 评论 / 媒体
- [x] 评论抓取：目前覆盖 xhs / douyin / bilibili / weibo / tieba / zhihu / kuaishou（支持分页；douyin/xhs/bilibili/weibo/tieba 支持二级评论开关；zhihu/kuaishou 为 HTML 初始数据 best-effort）。
- [x] 媒体下载：目前覆盖 xhs / douyin / weibo / bilibili（ENABLE_GET_MEDIAS=true 时下载到 note/media；weibo/bilibili 为 best-effort）。

### 测试
- [x] 平台注册/别名测试与多平台 replay 测试。
- [x] 代理池、Data API、WebSocket、SQLite upsert 等单测覆盖。

## Python 版能力基线（摘要）
- 平台：xhs / dy / ks / bili / wb / tieba / zhihu（7 个）
- 模式：search / detail / creator
- 存储：csv / json(数组写回) / excel / sqlite / mysql(db) / mongodb / postgres
- 代理：kuaidaili / wandouhttp（代理池 + 自动刷新）
- 登录：qrcode / phone / cookie（以 xhs/dy 的 Playwright 登录流最完整）
- 评论：7 平台均有“抓评论”实现（含部分平台二级评论开关）
- 媒体：至少覆盖 xhs / dy / wb / bili（按配置落盘）
- 词云：json 模式 + 开关打开时，爬取完成后自动生成

## 与 Python 版差异（未完成/需确认）

### 评论覆盖
- [x] 补齐评论抓取的平台覆盖：zhihu / kuaishou（已补齐；当前为 HTML 初始数据 best-effort，后续如需“全量翻页”再补稳定评论 API）。

### 媒体下载覆盖
- [x] 补齐媒体下载的平台覆盖：weibo / bilibili（已补齐；当前为 best-effort，bilibili 视频下载依赖 /x/player/playurl 可用性）。

### 存储/落盘格式
- [x] 兼容 Python 的 `SAVE_DATA_OPTION=excel`（已支持 excel 作为 xlsx 别名，并在 /config/options 暴露 excel）。
- [ ] 对齐（或明确文档差异）JSON 输出语义与目录结构：Python 的 json 为“数组写回 + data/{platform}/json/…”；Go 当前为“逐行追加 JSON + data/{platform}/notes/<note_id>/… 及全局文件”。

### 词云触发方式
- [ ] 评估是否提供“任务结束自动生成词云”的能力（Python 在 json 模式会自动生成；Go 当前通过 /data/wordcloud 手动触发）。

### 登录说明
- [ ] 细化“各平台支持的登录形态”说明：Go 的 phone/qrcode 自动化流程目前主要覆盖 xhs/douyin，其它平台通常仅依赖 COOKIES（HTTP client）。

## 开发任务清单（可执行）
- [x] T-101 增加 MongoDB 存储后端（store + /config/options 对齐）。
- [x] T-102 对齐 comments 的 CSV/XLSX：引入统一 Comment 结构并改造各平台落盘（当前仅 xhs/douyin 有评论抓取）。
- [x] T-103 补齐登录流程：phone/qrcode 已尝试自动切换登录方式（best-effort）。
- [x] T-104 落地 LOGIN_PHONE：phone 登录时尝试自动填充手机号（xhs/douyin）。
- [x] T-105 代理供应商按需扩展：新增 static provider 支持自定义代理列表/文件。
- [x] T-106 增加 HTTP 形态的日志查询：提供最近 N 条日志缓存接口。
- [x] T-201 补齐评论抓取的平台覆盖：zhihu/kuaishou（bilibili/weibo/tieba 已完成）。
- [x] T-202 补齐 weibo/bilibili 的媒体下载（并统一文件命名与重试策略）。
- [x] T-203 兼容 save_data_option：接受 excel 作为 xlsx 的别名，并同步更新文档与 /config/options。
- [ ] T-204 增加 Python 兼容输出模式（JSON 数组 + data/{platform}/{file_type}/命名规则）。
- [ ] T-205 可选：任务结束自动生成词云（对齐 Python 行为，可通过开关控制）。
