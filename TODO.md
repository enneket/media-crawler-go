# MediaCrawler Go 转写进度与 TODO

本文用于对齐 Python 版 `/home/zjx/code/mine/MediaCrawler` 的能力，并跟踪 Go 版 `/home/zjx/code/mine/media-crawler-go` 的转写进度。

## 已完成（Go，对齐 Python）

### 平台覆盖
- [x] 平台注册表 + 内置平台：xhs / douyin / bilibili / weibo / tieba / zhihu / kuaishou。
- [x] 三种模式：search / detail / creator（各平台均有实现入口）。

### CLI / 配置
- [x] CLI 入口：读取 config.yaml（viper）后按 PLATFORM 启动爬虫；支持 `-api` 启动 WebUI/API；支持 init-db 与常用参数覆盖。
- [x] YAML + 环境变量加载：`MEDIA_CRAWLER_` 前缀覆盖；默认值对齐 Python 核心配置。
- [x] HTTP 基础能力：超时/重试/退避（HTTP_TIMEOUT_SEC / HTTP_RETRY_*）。
- [x] 代理池：kuaidaili / wandouhttp / jishu_http / static(list/file)，支持失效切换（403/429 等场景 InvalidateCurrent）。
- [x] Cache：memory / redis / none（用于跨流程去重、词云缓存等）。
- [x] CDP 模式：可复用本机 Chrome/Edge 登录态（ENABLE_CDP_MODE / CDP_DEBUG_PORT / USER_DATA_DIR 等）。

### 存储 / 数据
- [x] 文件落地：json(逐行追加 JSON) / CSV（含 BOM）/ XLSX（含 header 校验）。
- [x] DB 后端：SQLite / MySQL / Postgres / MongoDB（自动初始化 schema，notes/creators/comments upsert/去重写入）。
- [x] 输出目录可配置：DATA_DIR（同时影响落盘与 /data API）。
- [x] Data API：文件列表/预览/下载/统计（/data/files /data/files/* /data/download/* /data/stats）。
- [x] 增值能力：评论词云（/data/wordcloud）。

### API / WebUI
- [x] API：/healthz /run /stop /status /config/platforms /config/options /env/check。
- [x] WebSocket：/ws/logs /ws/status。
- [x] WebUI：内置静态资源（embed），根路径返回 index.html。

### 评论 / 媒体
- [x] 评论抓取：目前覆盖 xhs / douyin / bilibili / weibo / tieba / zhihu / kuaishou（支持分页；douyin/xhs/bilibili/weibo/tieba/zhihu/kuaishou 支持二级评论开关；zhihu/kuaishou 优先走评论 API 翻页，失败回退 HTML 初始数据 best-effort）。
- [x] 媒体下载：目前覆盖 xhs / douyin / weibo / bilibili（ENABLE_GET_MEDIAS=true 时下载到 note/media；weibo/bilibili 为 best-effort）。

### 测试
- [x] 平台注册/别名测试与多平台 replay 测试。
- [x] 代理池、Data API、WebSocket、SQLite upsert 等单测覆盖。

## Python 版能力基线（摘要）
- 平台：xhs / dy / ks / bili / wb / tieba / zhihu（7 个）
- 模式：search / detail / creator
- 存储：csv / json(数组写回) / excel / sqlite / mysql(db) / mongodb / postgres
- 代理：kuaidaili / wandouhttp / jishu_http（代理池 + 自动刷新）
- 登录：qrcode / phone / cookie（以 xhs/dy 的 Playwright 登录流最完整）
- 评论：7 平台均有“抓评论”实现（含部分平台二级评论开关）
- 媒体：至少覆盖 xhs / dy / wb / bili（按配置落盘）
- 词云：json 模式 + 开关打开时，爬取完成后自动生成
- WebUI / API：FastAPI + 静态 WebUI 资源
- CDP：支持 CDP 模式复用本机浏览器登录态

## 与 Python 版差异（功能一致口径下可忽略）

功能一致口径：以 Python 版开源项目“对外可用能力”为准（7 平台 + search/detail/creator + 评论/子评开关 + 媒体下载开关 + 词云开关 + 代理池 + 多种落盘/DB 后端 + WebUI 数据预览/下载/统计）。不要求兼容 Python 的接口路径、字段名、以及 DB 的列式多表设计。

### 评论覆盖
- [x] 补齐评论抓取的平台覆盖：zhihu / kuaishou（已补齐；支持全量翻页 best-effort：优先 API 翻页，失败回退 HTML 初始数据解析）。

### API 兼容层（可选）
- [x] （可选）增加 Python 版 API 兼容路由：`/crawler/start`、`/crawler/stop`、`/crawler/status`（保留现有 `/run`、`/stop`、`/status`）。
- [x] （可选）兼容 Python 版请求字段命名：`login_type/save_option/specified_ids/creator_ids/enable_comments/...` 映射到 Go 的 RunRequest/Config。
- [x] （可选）增加 Python 版 `/api` 前缀兼容：`/api/health`、`/api/env/check`、`/api/config/*`、`/api/data/*`、`/api/logs`、`/api/ws/*`（保持现有无前缀路由不变）。
- [x] （可选）对齐 Python `/api/env/check` 语义：执行依赖/环境探测（Playwright/浏览器/权限）并返回可读错误，而不仅是版本信息。

### 媒体下载覆盖
- [x] 补齐媒体下载的平台覆盖：weibo / bilibili（已补齐；当前为 best-effort，bilibili 视频下载依赖 /x/player/playurl 可用性）。

### 存储/落盘格式
- [x] 兼容 Python 的 `SAVE_DATA_OPTION=excel`：支持 excel 映射为 `xlsx_book`（单文件多 Sheet），并在 /config/options 暴露 excel。
- [x] 对齐（或明确文档差异）JSON 输出语义与目录结构：新增 `PYTHON_COMPAT_OUTPUT=true` 时输出“数组写回 + data/{platform}/json/...”，默认仍保留 Go 的 per-note/jsonl 结构。

### SaveDataOption 兼容（可选）
- [x] （可选）CLI/API 兼容 Python `save_option=db/sqlite/postgres/mongodb`：自动映射到 `STORE_BACKEND`（并保持 `SAVE_DATA_OPTION` 仅表示文件格式）。

### 词云触发方式
- [x] 任务结束自动生成词云：新增 `ENABLE_GET_WORDCLOUD=true` 时任务结束自动生成（同时保留 /data/wordcloud 手动触发）。

### 登录说明
- [x] 细化“各平台支持的登录形态”说明：README 已补充各平台支持范围（xhs/douyin 支持 qrcode/phone/cookie，其它平台一般为 cookie）。

### Phone 登录自动化（验证码回填）
- [x] （可选）对齐 Python 的短信转发登录链路：提供 SMS Webhook（接收短信内容提取验证码）+ Redis 缓存，并在 xhs/douyin `LOGIN_TYPE=phone` 时自动回填验证码。

### 代理能力对齐
- [x] 代理供应商对齐：补齐 Python 版的 `jishu_http` provider（支持 jisuhttp/jishuhttp/jishu_http）。
- [x] 统一代理接入：bilibili/weibo/tieba/zhihu/kuaishou 的 HTTP client 全链路走代理池（并补齐 douyin 短链解析的代理接入）。

### 词云（DB 后端）
- [x] 自动词云读取 DB 覆盖：支持 sqlite/mysql/postgres/mongodb（best-effort，读取 comments.data_json 生成词云）。

### CLI 形态
- [x] CLI 参数对齐：支持 init-db 子命令，支持常用参数覆盖（cookies/inputs/keywords 等直接覆盖配置）。
- [x] CLI 覆盖项补齐：支持 `--get_comment/--get_sub_comment/--headless/--save_data_option/--start/--max_concurrency_num` 等常用覆盖（含 `--specified_id/--creator_id` 别名）。

### 反检测能力（Stealth）
- [x] Stealth 脚本注入对齐（best-effort）：Go 版在 xhs/douyin 的 Playwright/CDP 上注入统一 init script（含 webdriver/languages/plugins/permissions/webgl 等常见特征处理）。

### CDP 端口选择
- [x] CDP DebugPort 自动回退：按配置端口作为起点探测可用端口并回退，避免端口被占用导致启动失败。

### 词云能力细节
- [x] 词云质量对齐（best-effort）：支持 STOP_WORDS_FILE/CUSTOM_WORDS/FONT_PATH，保存 PNG 与词频 JSON；分词为“汉字段 + 停用词切分 + 自定义词匹配”的简化实现。

### B 站高级配置（需确认是否要完全对齐）
- [x] （可选）对齐 Python bilibili 高级项：`BILI_QN`（清晰度）、`START_DAY/END_DAY`（时间范围过滤）、`MAX_NOTES_PER_DAY`（按日上限）、动态/联系人等扩展抓取项（如仍需要）。

### 数据库结构化程度（可选差异）
- [x] （可选）对齐 Python 的“按平台多表 + 列级字段”的关系型 schema（或明确不对齐）：Go 版当前为统一表 + data_json，便于归档但不利于复杂 SQL 分析（已评估：保持现状）。

### Excel 导出（可选）
- [-] （可选）对齐 Python Excel 额外 Sheet：B 站支持导出 Contacts/Dynamics（Blocked: 需先补齐采集能力）。

## 体验差异（待增强，不影响主功能）

这些能力不影响 Python 开源版的“功能入口”，但在稳定性/易用性/输出质量上仍有差距，建议后续按需补齐。

### 评论稳定性
- [x] 知乎评论全量翻页（best-effort）：优先走知乎 v4 评论 API 翻页（paging.next），失败回退 HTML 初始数据解析；支持子评论分页（best-effort）。
- [x] 快手评论全量翻页（best-effort）：优先走 GraphQL 评论接口翻页（pcursor），失败回退 HTML 初始数据解析；支持子评论 subCommentList 翻页（best-effort）。

### 媒体下载稳定性
- [x] B 站下载稳定性增强（best-effort）：下载限并发、指数退避+抖动，避免瞬时并发触发 429/风控。
- [x] 微博下载稳定性增强（best-effort）：下载限并发、指数退避+抖动，并识别 text/html 或 json 响应避免落地垃圾文件。

### 反检测强度
- [x] 支持注入完整 stealth.min.js（best-effort）：支持通过配置指定脚本路径/目录（默认 best-effort 内置脚本），并自动探测 `libs/stealth.min.js`。

### Excel 导出体验
- [x] Excel 单文件多 Sheet（best-effort）：新增 `SAVE_DATA_OPTION=excel` 映射为 `xlsx_book`，输出单个 workbook（Contents/Comments/Creators），含表头样式/冻结首行/自动列宽。

### 存储语义一致性
- [x] 统一/澄清 STORE_BACKEND 与 SAVE_DATA_OPTION（best-effort）：补充文档与 CLI 说明，减少与 Python `SAVE_DATA_OPTION` 的认知差异。

## 开发任务清单（可执行）
- [x] T-101 增加 MongoDB 存储后端（store + /config/options 对齐）。
- [x] T-102 对齐 comments 的 CSV/XLSX：引入统一 Comment 结构并改造各平台落盘（已覆盖 7 平台，含二级评论开关）。
- [x] T-103 补齐登录流程：phone/qrcode 已尝试自动切换登录方式（best-effort）。
- [x] T-104 落地 LOGIN_PHONE：phone 登录时尝试自动填充手机号（xhs/douyin）。
- [x] T-105 代理供应商按需扩展：新增 static provider 支持自定义代理列表/文件。
- [x] T-106 增加 HTTP 形态的日志查询：提供最近 N 条日志缓存接口。
- [x] T-201 补齐评论抓取的平台覆盖：zhihu/kuaishou（bilibili/weibo/tieba 已完成）。
- [x] T-202 补齐 weibo/bilibili 的媒体下载（并统一文件命名与重试策略）。
- [x] T-203 兼容 save_data_option：接受 excel 作为 xlsx 的别名，并同步更新文档与 /config/options。
- [x] T-204 增加 Python 兼容输出模式（JSON 数组 + data/{platform}/{file_type}/命名规则）。
- [x] T-205 任务结束自动生成词云（对齐 Python 行为，可通过开关控制）。
- [x] T-301 代理供应商对齐：新增 jishu_http provider。
- [x] T-302 统一代理接入：所有平台 HTTP client 统一走代理池。
- [x] T-303 词云 DB 覆盖：支持从 mysql/postgres/mongodb 读取评论数据生成词云。
- [x] T-304 丰富 CLI：补齐 init_db 与常用参数覆盖/子命令体系。
- [x] T-401 反检测对齐：注入统一 stealth init script（best-effort）。
- [x] T-402 CDP 端口对齐：自动探测可用 DebugPort 并回退。
- [x] T-403 词云对齐：支持停用词/自定义词/更合理中文分词（best-effort），并补齐 PNG/词频输出。
- [x] T-404 CLI 覆盖对齐：补齐评论/子评论/Headless/CDP/词云等运行开关的 CLI 覆盖。
- [x] T-501 知乎评论全量翻页与稳定抓取（best-effort）。
- [x] T-502 快手评论全量翻页与稳定抓取（best-effort）。
- [x] T-503 媒体下载稳定性增强：B 站/微博（best-effort）。
- [x] T-504 Stealth 强度增强（best-effort）：支持配置完整 stealth.min.js 注入与路径解析增强。
- [x] T-505 Excel 导出体验对齐：单文件多 Sheet + 样式优化（best-effort）。
- [x] T-506 存储语义对齐（best-effort）：补充 README 与 /config/options 说明，明确 STORE_BACKEND（是否写 DB）与 SAVE_DATA_OPTION（文件格式）职责边界，并增加 xlsx_book/excel 的解释。
- [x] T-601 Python API 兼容路由：增加 `/crawler/start|stop|status` 并兼容字段映射。
- [x] T-602 Phone 登录自动化：增加 SMS Webhook + Redis 验证码缓存，xhs/douyin 自动回填验证码。
- [x] T-603 SaveDataOption 兼容：接受 python `save_option=db/sqlite/postgres/mongodb` 并映射 `STORE_BACKEND`。
- [x] T-604 B 站高级配置对齐：补齐 `BILI_QN`、`START_DAY/END_DAY` 与扩展抓取项（如仍需要）。
- [x] T-605 Python API 前缀兼容：增加 `/api/*` 路由别名与 WS 别名，保持现有接口不变。
- [x] （可选）T-606 结构化 DB schema 评估：确定是否要对齐 Python 的多表列式设计，并给出方案（已评估：保持 data_json 统一存储，暂不对齐多表设计）。
- [x] T-607 EnvCheck 对齐：实现依赖探测版 `/api/env/check`（Playwright/浏览器/CDP 端口/权限等）。
- [x] （可选）T-608 Excel 扩展：B 站 Contacts/Dynamics 导出与落盘（Blocked: 需先补齐 Dynamics/Contacts 采集能力）。
