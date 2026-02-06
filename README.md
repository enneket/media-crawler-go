# MediaCrawler Go

This is a Go rewrite of the [MediaCrawler](https://github.com/NanmiCoder/MediaCrawler) project.
It supports crawling multiple platforms with signature generation using Playwright:

- xhs (Xiaohongshu / 小红书)
- douyin (抖音)
- bilibili
- weibo
- tieba
- zhihu
- kuaishou

## Prerequisites

- Go (see `go.mod`)
- Chrome/Chromium browser

## Installation

1. Clone the repository
2. Install dependencies:
   ```bash
   go mod download
   ```
3. Install Playwright browsers (optional if you have Chrome):
   ```bash
   go run github.com/playwright-community/playwright-go/cmd/playwright install
   ```

## Configuration

Create a `config.yaml` file in the root directory (see `config.example.yaml`).

Notes:
- If `HEADLESS: true`, you must use `LOGIN_TYPE: cookie` and provide `COOKIES`.
- `LOGIN_TYPE: qrcode/phone` relies on completing login manually in the opened browser window; the crawler waits up to `LOGIN_WAIT_TIMEOUT_SEC`. If `LOGIN_TYPE: phone` and `LOGIN_PHONE` is set, it will try to prefill the phone input (best-effort).
- Login support by platform (best-effort):
  - xhs/douyin: qrcode / phone / cookie
  - bilibili/weibo/tieba/zhihu/kuaishou: cookie (HTTP client)
- Proxy: set `ENABLE_IP_PROXY: true`. `IP_PROXY_PROVIDER_NAME` supports `kuaidaili`, `wandouhttp`, `jisuhttp` (or `jishuhttp`/`jishu_http`), and `static` (use `IP_PROXY_LIST` or `IP_PROXY_FILE`).
- `STORE_BACKEND` controls DB writes (`file` disables DB; `sqlite/mysql/postgres/mongodb` will upsert notes/creators and insert comments into DB in addition to file output).
- `SAVE_DATA_OPTION` controls file output: `json` / `csv` / `xlsx` / `xlsx_book` (`excel` is accepted as an alias and will be normalized to `xlsx_book` for Python compatibility).
- `PYTHON_COMPAT_OUTPUT: true` will additionally write Python-style JSON arrays to `data/<platform>/json/<crawler_type>_<item_type>_<date>.json`.
- `ENABLE_GET_WORDCLOUD: true` will auto-generate `wordcloud_comments_*.svg` after the task finishes (best-effort).
- Bilibili Specific:
  - `BILI_QN`: Video quality (e.g. 80 for 1080P).
  - `BILI_DATE_RANGE_START` / `BILI_DATE_RANGE_END`: Filter videos by publish date (YYYY-MM-DD).
  - `BILI_MAX_NOTES_PER_DAY`: Daily limit for videos.
  - `BILI_ENABLE_GET_DYNAMICS: true`: In `creator` mode, also crawl creator's dynamics (feed) and save to `Dynamics` sheet (xlsx_book) or jsonl.

## Output

- Notes: `data/<platform>/notes/<note_id>/note.(json|csv|xlsx)` or a single workbook `data/<platform>/<platform>_<crawler_type>_<timestamp>.xlsx` (if `SAVE_DATA_OPTION=xlsx_book` or `excel`)
- Comments: `data/<platform>/notes/<note_id>/comments.(jsonl|csv|xlsx)` (deduped via `comments.idx`)
- Global Comments: `data/<platform>/comments.(jsonl|csv|xlsx)` (unified schema, deduped via `comments.global.idx`)
- Workbook mode: `SAVE_DATA_OPTION=xlsx_book` (or `excel`) writes `Contents/Comments/Creators` sheets into one workbook (best-effort); Bilibili creator mode adds `Dynamics` sheet.
- Media: `data/<platform>/notes/<note_id>/media/*`

## API Mode (Web UI)

Start the API server:

```bash
go run cmd/media-crawler/main.go -api -addr :8080
```

Then open `http://127.0.0.1:8080/` in the browser.

## Douyin Detail (Example)

- Set `PLATFORM: "douyin"` (or `"dy"`), `CRAWLER_TYPE: "detail"`
- Provide `DY_SPECIFIED_NOTE_URL_LIST` with `/video/<aweme_id>` URL or numeric aweme_id
- `ENABLE_GET_COMMENTS` will fetch `/aweme/v1/web/comment/list/` (and optional `/reply/` if `ENABLE_GET_SUB_COMMENTS`)
- `ENABLE_GET_MEDIAS` will download `play_addr.url_list[0]` and up to 3 cover urls to `media/`

## Douyin Search / Creator

- `CRAWLER_TYPE: "search"` will use `KEYWORDS` to search (signed with `a_bogus`) and then reuse the same detail pipeline.
- `CRAWLER_TYPE: "creator"` will use `DY_CREATOR_ID_LIST` to fetch creator profile and posts, then reuse the same detail pipeline.

## Usage

Build and run:

```bash
go build -o media-crawler cmd/media-crawler/main.go
./media-crawler
```

Or run directly:

```bash
go run cmd/media-crawler/main.go
```

CLI examples:

```bash
# Run with overrides (no config edit)
./media-crawler -platform xhs -mode search -keywords "编程副业,编程兼职"

# Detail mode with explicit inputs (meaning depends on platform+mode)
./media-crawler -platform bilibili -mode detail -inputs "https://www.bilibili.com/video/BV1xxx,https://www.bilibili.com/video/BV2yyy"

# Init DB schema/indexes for SQL backends
./media-crawler init-db -store_backend sqlite -sqlite_path data/media_crawler.db
```

## Features

- [x] Xiaohongshu Crawling (search/detail/creator)
- [x] Douyin Crawling (search/detail/creator)
- [x] Bilibili Crawling (search/detail/creator + dynamics)
- [x] Weibo Crawling (search/detail/creator)
- [x] Tieba Crawling (search/detail/creator)
- [x] Zhihu Crawling (search/detail/creator)
- [x] Kuaishou Crawling (search/detail/creator)
- [x] Signature Generation (X-S, X-T, X-S-Common) using Playwright
- [x] Persistent Browser Context (Login state saving)
- [x] Comment Crawling (pagination, optional sub-comments, currently for xhs/douyin/bilibili/weibo/tieba/zhihu/kuaishou)
- [x] Media Download (basic, currently for xhs/douyin/weibo/bilibili)
- [x] CDP Mode (connect over remote debugging)
- [x] Proxy Pool (kuaidaili / wandouhttp / jisuhttp / static list)
- [x] Store Backends (file/sqlite/mysql/postgres/mongodb)

See [TODO.md](./TODO.md) for the porting checklist.

## Disclaimer

This project is for learning and research purposes only. Please comply with the target platform's terms of use.
