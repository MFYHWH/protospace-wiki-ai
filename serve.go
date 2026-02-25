package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func runServe() {
	dbPath := envOr("WIKI_DB", "./wiki.db")
	db, err := openStore(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening %q: %v\nRun \"protospace-wiki scrape\" first.\n", dbPath, err)
		os.Exit(1)
	}
	defer db.close()

	count, err := db.pageCount()
	if err != nil || count == 0 {
		fmt.Fprintf(os.Stderr, "Error: No pages in %q. Run \"protospace-wiki scrape\" first.\n", dbPath)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Loaded %d pages from %s\n", count, dbPath)

	s := server.NewMCPServer("protospace-wiki", "1.0.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	s.AddTool(
		mcp.NewTool("search_wiki",
			mcp.WithDescription("Search across all Protospace wiki pages. Returns matching page titles and text snippets. Use this to find relevant pages before reading them."),
			mcp.WithString("query", mcp.Required(), mcp.Description(`Search terms (e.g. "laser cutter safety")`)),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			query, err := req.RequireString("query")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			results, err := db.search(query, 10)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
			}
			if len(results) == 0 {
				return mcp.NewToolResultText(fmt.Sprintf("No results found for %q.", query)), nil
			}
			var lines []string
			for i, r := range results {
				lines = append(lines, fmt.Sprintf("%d. **%s**\n   %s", i+1, r.title, r.snippet))
			}
			return mcp.NewToolResultText(strings.Join(lines, "\n\n")), nil
		},
	)

	s.AddTool(
		mcp.NewTool("read_page",
			mcp.WithDescription("Read the full content of a Protospace wiki page. Use search_wiki first to find the right page title."),
			mcp.WithString("title", mcp.Required(), mcp.Description(`Page title or filename (e.g. "Laser Cutter" or "Laser_Cutter.md")`)),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			title, err := req.RequireString("title")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			p, err := db.readPage(title)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Page %q not found. Try search_wiki to find the correct title.", title)), nil
			}
			content := fmt.Sprintf("---\ntitle: \"%s\"\nsource: %s\nrevision: %s\ncategories: %s\n---\n\n%s",
				strings.ReplaceAll(p.title, `"`, `\"`), p.source, p.revision, p.categories, p.body,
			)
			return mcp.NewToolResultText(content), nil
		},
	)

	fmt.Fprintf(os.Stderr, "MCP server running on stdio\n")
	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
