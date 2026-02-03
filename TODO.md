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
- [x] 文件落地：JSON Lines / CSV（含 BOM）/ XLSX（含 header 校验）。
- [x] DB 后端：SQLite / MySQL / Postgres（自动初始化 schema，notes/creators/comments upsert/去重写入）。
- [x] 输出目录可配置：DATA_DIR（同时影响落盘与 /data API）。
- [x] Data API：文件列表/预览/下载/统计（/data/files /data/files/* /data/download/* /data/stats）。
- [x] 增值能力：评论词云（/data/wordcloud）。

### API / WebUI
- [x] API：/healthz /run /stop /status /config/platforms /config/options /env/check。
- [x] WebSocket：/ws/logs /ws/status。
- [x] WebUI：内置静态资源（embed），根路径返回 index.html。

### 测试
- [x] 平台注册/别名测试与多平台 replay 测试。
- [x] 代理池、Data API、WebSocket、SQLite upsert 等单测覆盖。

## 与 Python 版差异（未完成/需确认）

### 存储/数据
- [x] MongoDB 存储后端（支持 notes/creators/comments；并在 /config/options 暴露 mongodb）。
- [x] comments 导出已基本对齐：xhs/douyin 提供统一格式的全局 comments.(jsonl|csv|xlsx) + per-note 保持原格式。

### 登录
- [x] LOGIN_TYPE=phone/qrcode 的“自动化流程”已增强：会尝试自动打开登录弹窗并切到对应方式（xhs/douyin，best-effort），仍需手动完成验证/扫码/短信。
- [x] LOGIN_PHONE 已生效：LOGIN_TYPE=phone 时会尝试自动填充手机号到登录输入框（xhs/douyin，best-effort）。
- [x] SAVE_LOGIN_STATE 已生效：为 false 时使用临时 userDataDir，任务结束自动清理（xhs/douyin）。

### 代理
- [x] 代理供应商差异已缓解：新增 static provider 支持自定义代理列表/文件（仍可按需补齐更多供应商）。

### API/WebUI
- [x] 增加“拉取历史日志”接口：GET /logs（并兼容 GET /crawler/logs）。

### 文档/样例一致性
- [x] README 平台状态描述已更新（与代码内置平台对齐，并补充 API mode）。
- [x] config.example.yaml 的 STORE_BACKEND 注释与实际支持项对齐（file/sqlite/mysql/postgres/mongodb）。

## 开发任务清单（可执行）
- [x] T-101 增加 MongoDB 存储后端（store + /config/options 对齐）。
- [x] T-102 对齐 comments 的 CSV/XLSX：引入统一 Comment 结构并改造各平台落盘（当前仅 xhs/douyin 有评论抓取）。
- [x] T-103 补齐登录流程：phone/qrcode 已尝试自动切换登录方式（best-effort）。
- [x] T-104 落地 LOGIN_PHONE：phone 登录时尝试自动填充手机号（xhs/douyin）。
- [x] T-105 代理供应商按需扩展：新增 static provider 支持自定义代理列表/文件。
- [x] T-106 增加 HTTP 形态的日志查询：提供最近 N 条日志缓存接口。
