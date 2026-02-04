# MediaCrawler Go 目录结构（Agent 速览）

这是一个用 Go 重写的 MediaCrawler（多平台爬虫/采集器）项目。整体布局遵循常见的 Go 工程习惯：`cmd/` 放可执行入口，核心业务放在 `internal/`，配置与说明文档放在仓库根目录。

## 顶层结构

```
.
├── cmd/
│   └── media-crawler/
│       └── main.go
├── internal/
│   ├── api/
│   │   ├── webui/          # 内嵌静态页面（index.html/app.js/styles.css）
│   │   └── *.go            # API Server / WebSocket / 数据接口等
│   ├── browser/            # Playwright/CDP 相关：浏览器启动、探测、用户数据目录
│   ├── cache/              # 缓存抽象与实现（memory/redis），从配置构建
│   ├── config/             # 配置结构、默认值、加载（viper）、规范化
│   ├── crawler/            # 通用爬取流程：并发、重试、风险提示、错误分类等
│   ├── downloader/         # 媒体下载（目前主要面向 xhs/douyin）
│   ├── logger/             # 日志与广播（用于 API/WS 实时推送）
│   ├── platform/           # 各平台实现与注册表（xhs/douyin/bilibili/...）
│   ├── proxy/              # 代理池与 provider（kuaidaili/wandouhttp/static）
│   └── store/              # 存储层：file/sqlite/mysql/postgres/mongodb + 导出 xlsx
├── config.example.yaml      # 配置示例（复制为 config.yaml 使用）
├── go.mod / go.sum          # Go module 依赖
├── README.md                # 使用说明
└── TODO.md                  # 迁移/待办清单
```

## 入口与运行模式

- CLI 入口：[main.go](file:///home/zjx/code/mine/media-crawler-go/cmd/media-crawler/main.go)
  - 默认模式：按 `config.yaml` 启动爬虫，选择 `PLATFORM` 并执行对应平台 Runner
  - API 模式：`-api` 启动 Web UI + HTTP API（默认 `:8080`）

## internal/ 目录详解

### internal/api：API Server + Web UI

- HTTP 路由与服务装配：[server.go](file:///home/zjx/code/mine/media-crawler-go/internal/api/server.go)
- Web UI 静态资源内嵌（Go embed）：[webui.go](file:///home/zjx/code/mine/media-crawler-go/internal/api/webui.go)
- 静态资源目录：[internal/api/webui/](file:///home/zjx/code/mine/media-crawler-go/internal/api/webui)

### internal/platform：平台实现与注册

- 注册表与平台构造：[registry.go](file:///home/zjx/code/mine/media-crawler-go/internal/platform/registry.go)
- 各平台通常包含：
  - `client.go`：HTTP/签名/请求封装
  - `crawler.go`：平台 Runner（实现通用 `crawler.Runner` 接口）
  - `parse.go`：响应解析与数据模型转换
  - `register.go`：在 init 中把平台注册到 registry（入口通过匿名导入触发）

### internal/config：配置加载

- 配置结构与加载入口：[config.go](file:///home/zjx/code/mine/media-crawler-go/internal/config/config.go)
- 配置示例：[config.example.yaml](file:///home/zjx/code/mine/media-crawler-go/config.example.yaml)

### internal/crawler：通用爬虫能力

这里放跨平台可复用的“底座能力”，例如并发控制、重试退避、错误分类/风险提示等（供 `internal/platform/*` 复用）。

### internal/store：数据落地与导出

提供多种存储后端与导出能力，输出目录默认为 `DATA_DIR`（示例配置里是 `data/`）。具体文件布局可参考 [README.md](file:///home/zjx/code/mine/media-crawler-go/README.md) 的 Output 一节。

### internal/proxy：代理池

从配置构建代理 provider，支持动态池化与切换（目前内置 `kuaidaili`、`wandouhttp`、`static`）。

## 常用导航

- 项目说明与运行方式：[README.md](file:///home/zjx/code/mine/media-crawler-go/README.md)
- 迁移/待办清单：[TODO.md](file:///home/zjx/code/mine/media-crawler-go/TODO.md)
