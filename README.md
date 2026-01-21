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

Create a `config.yaml` file in the root directory (see `config.yaml` example).

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

- [x] Xiaohongshu Search Crawling
- [x] Signature Generation (X-S, X-T, X-S-Common) using Playwright
- [x] Persistent Browser Context (Login state saving)
- [ ] Comment Crawling
- [ ] Media Download
- [ ] Other Platforms (Douyin, Bilibili, etc.)

## Disclaimer

This project is for learning and research purposes only. Please comply with the target platform's terms of use.
