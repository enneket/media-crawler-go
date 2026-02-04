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
- Proxy: set `ENABLE_IP_PROXY: true`. `IP_PROXY_PROVIDER_NAME` supports `kuaidaili`, `wandouhttp`, and `static` (use `IP_PROXY_LIST` or `IP_PROXY_FILE`).
- `SAVE_DATA_OPTION`: `json` / `csv` / `xlsx` (`excel` is accepted as an alias for Python compatibility).

## Output

- Notes: `data/<platform>/notes/<note_id>/note.(json|csv|xlsx)`
- Comments: `data/<platform>/notes/<note_id>/comments.(jsonl|csv|xlsx)` (deduped via `comments.idx`, xlsx currently for xhs/douyin)
- Global Comments: `data/<platform>/comments.(jsonl|csv|xlsx)` (unified schema, deduped via `comments.global.idx`, currently for xhs/douyin)
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

## Features

- [x] Xiaohongshu Crawling (search/detail/creator)
- [x] Douyin Crawling (search/detail/creator)
- [x] Bilibili Crawling (search/detail/creator)
- [x] Weibo Crawling (search/detail/creator)
- [x] Tieba Crawling (search/detail/creator)
- [x] Zhihu Crawling (search/detail/creator)
- [x] Kuaishou Crawling (search/detail/creator)
- [x] Signature Generation (X-S, X-T, X-S-Common) using Playwright
- [x] Persistent Browser Context (Login state saving)
- [x] Comment Crawling (pagination, optional sub-comments, currently for xhs/douyin/bilibili/weibo)
- [x] Media Download (basic, currently for xhs/douyin)
- [x] CDP Mode (connect over remote debugging)
- [x] Proxy Pool (kuaidaili / wandouhttp / static list)
- [x] Store Backends (file/sqlite/mysql/postgres/mongodb)

See [TODO.md](./TODO.md) for the porting checklist.

## Disclaimer

This project is for learning and research purposes only. Please comply with the target platform's terms of use.
