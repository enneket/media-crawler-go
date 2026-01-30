# MediaCrawler Go

This is a Go rewrite of the [MediaCrawler](https://github.com/NanmiCoder/MediaCrawler) project.
It currently supports crawling **Xiaohongshu (XHS)** with signature generation using Playwright.

## Prerequisites

- Go 1.21+
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
- `LOGIN_TYPE: qrcode/phone` relies on completing login manually in the opened browser window; the crawler waits up to `LOGIN_WAIT_TIMEOUT_SEC`.

## Output

- Notes: `data/<platform>/notes/<note_id>/note.(json|csv)`
- Comments: `data/<platform>/notes/<note_id>/comments.(jsonl|csv)` (deduped via `comments.idx`)
- Media: `data/<platform>/notes/<note_id>/media/*`

## Douyin Detail

- Set `PLATFORM: "douyin"` (or `"dy"`), `CRAWLER_TYPE: "detail"`
- Provide `DY_SPECIFIED_NOTE_URL_LIST` with `/video/<aweme_id>` URL or numeric aweme_id
- `ENABLE_GET_COMMENTS` will fetch `/aweme/v1/web/comment/list/` (and optional `/reply/` if `ENABLE_GET_SUB_COMMENTS`)
- `ENABLE_GET_MEDIAS` will download `play_addr.url_list[0]` and up to 3 cover urls to `media/`

## Douyin Search / Creator

- `CRAWLER_TYPE: "search"` will use `KEYWORDS` to search and then reuse the same detail pipeline.
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
- [x] Douyin Crawling (detail)
- [x] Signature Generation (X-S, X-T, X-S-Common) using Playwright
- [x] Persistent Browser Context (Login state saving)
- [x] Comment Crawling (pagination, optional sub-comments)
- [x] Media Download (basic)
- [x] CDP Mode (connect over remote debugging)
- [x] Proxy Pool (kuaidaili / wandouhttp)
- [ ] Other Platforms (Bilibili, Weibo, etc.)

See [TODO.md](./TODO.md) for the porting checklist.

## Disclaimer

This project is for learning and research purposes only. Please comply with the target platform's terms of use.
