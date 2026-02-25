# protospace-wiki-ai

The wiki is behind a cookie check that blocks AI crawlers. This repo opens it up so AI tools can actually read it — both for discovery (people finding Protospace through AI) and for members using AI to search the wiki.

One-time server deploy. Members don't need to do anything.

## What's in here

- `nginx/` — configs to allowlist AI crawlers past the cookie check, plus `robots.txt` and `llms.txt`
- `serve.go` — optional MCP server for direct Claude Desktop integration
- `scrape.go` — downloads the wiki into a local SQLite database for the MCP server

Single Go binary, no dependencies. `make build` and you're done.

## Deploy: nginx crawler allowlist

Let AI crawlers through. Copy the static files and integrate the configs:

```bash
sudo cp nginx/robots.txt /var/www/robots.txt
sudo cp nginx/llms.txt /var/www/llms.txt
```

1. Add `nginx/ai-crawlers.conf` map directive to your `http {}` block
2. Add `nginx/wiki-markdown.conf` location blocks to your `server {}` block
3. Add `if ($is_ai_crawler) { break; }` to the cookie check

```bash
sudo nginx -t && sudo systemctl reload nginx
```

Verify:
```bash
curl -s -A "ClaudeBot/1.0" https://wiki.protospace.ca/Main_Page | wc -c  # >> 314
curl -s https://wiki.protospace.ca/robots.txt | head -1                   # robots.txt served
```

To undo: remove the map block and `$is_ai_crawler` bypass, reload nginx.

## Optional: Local MCP server

If a member wants Claude Desktop to have search/read tools for the wiki directly:

```bash
make build
./protospace-wiki scrape   # ~2 min, creates wiki.db
```

Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json` on Mac):

```json
{
  "mcpServers": {
    "protospace-wiki": {
      "command": "/FULL/PATH/TO/protospace-wiki",
      "args": ["serve"],
      "env": { "WIKI_DB": "/FULL/PATH/TO/wiki.db" }
    }
  }
}
```

Two tools: `search_wiki` (full-text search, BM25 ranking) and `read_page` (full page content). Re-run `scrape` to refresh.

## Environment variables

### scrape

| Variable | Default | Description |
|----------|---------|-------------|
| `WIKI_API` | `https://wiki.protospace.ca/api.php` | MediaWiki API |
| `WIKI_URL` | `https://wiki.protospace.ca` | Base URL |
| `WIKI_COOKIE` | `human_check=verified` | Auth cookie |
| `WIKI_DB` | `./wiki.db` | Output DB path |

### serve

| Variable | Default | Description |
|----------|---------|-------------|
| `WIKI_DB` | `./wiki.db` | DB path |

## AI crawlers allowlisted

Configured in `nginx/ai-crawlers.conf`. Verified against official docs, Feb 2026.

| Provider | Bots |
|----------|------|
| Anthropic | ClaudeBot, Claude-SearchBot, Claude-User |
| OpenAI | GPTBot, OAI-SearchBot, ChatGPT-User |
| Perplexity | PerplexityBot, Perplexity-User |
| Google | Google-Extended, Google-CloudVertexBot, Gemini-Deep-Research |
| Apple | Applebot-Extended |
| Meta | meta-externalagent, meta-externalfetcher |

## Building from source

Requires [Go 1.24+](https://go.dev/dl/). No CGo.

```bash
make build          # Current platform → ./protospace-wiki
make build-all      # All platforms → dist/
make clean          # Remove build artifacts
```
