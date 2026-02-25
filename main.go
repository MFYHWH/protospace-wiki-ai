package main

import (
	"fmt"
	"os"
	"strings"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "scrape":
		runScrape()
	case "serve":
		runServe()
	case "version":
		fmt.Println(version)
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: protospace-wiki <command>\n\nCommands:\n")
	fmt.Fprintf(os.Stderr, "  scrape   Download all wiki pages into wiki.db\n")
	fmt.Fprintf(os.Stderr, "  serve    Run MCP server (stdio transport)\n")
	fmt.Fprintf(os.Stderr, "  version  Print version\n")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func formatCategories(cats []string) string {
	quoted := make([]string, len(cats))
	for i, c := range cats {
		quoted[i] = `"` + strings.ReplaceAll(c, `"`, `\"`) + `"`
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}
