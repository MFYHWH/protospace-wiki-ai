package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"
)

func runScrape() {
	apiURL := envOr("WIKI_API", "https://wiki.protospace.ca/api.php")
	wikiURL := envOr("WIKI_URL", "https://wiki.protospace.ca")
	cookie := envOr("WIKI_COOKIE", "human_check=verified")
	dbPath := envOr("WIKI_DB", "./wiki.db")

	os.Remove(dbPath)

	client := newWikiClient(apiURL, cookie)
	db, err := createStore(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer db.close()

	ctx := context.Background()
	fmt.Fprintf(os.Stderr, "Fetching page list...\n")
	pages, err := client.getAllPages(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Found %d pages\n", len(pages))

	tx, err := db.beginTx()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var ok, skip, fail int
	for _, p := range pages {
		parsed, err := client.parsePage(ctx, p.Title)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  FAIL: %s: %v\n", p.Title, err)
			fail++
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if parsed == nil {
			skip++
			time.Sleep(100 * time.Millisecond)
			continue
		}

		title := cleanDisplayTitle(parsed.displayTitle)
		body, err := htmlToMarkdown(parsed.html, wikiURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  FAIL: %s: %v\n", p.Title, err)
			fail++
			time.Sleep(100 * time.Millisecond)
			continue
		}

		source := wikiURL + "/" + encodeWikiTitle(p.Title)
		err = db.insertTx(tx, title, body, source, strconv.Itoa(parsed.revID), formatCategories(parsed.categories))
		if err != nil {
			fmt.Fprintf(os.Stderr, "  FAIL: %s: %v\n", p.Title, err)
			fail++
			time.Sleep(100 * time.Millisecond)
			continue
		}

		ok++
		if ok%50 == 0 {
			fmt.Fprintf(os.Stderr, "  %d pages converted...\n", ok)
		}
		time.Sleep(100 * time.Millisecond)
	}

	if err := tx.Commit(); err != nil {
		fmt.Fprintf(os.Stderr, "Error committing: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "\nDone: %d converted, %d skipped, %d failed\n", ok, skip, fail)
}
