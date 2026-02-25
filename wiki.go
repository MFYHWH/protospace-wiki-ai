package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// --- MediaWiki API client ---

type wikiClient struct {
	apiURL string
	cookie string
	http   *http.Client
}

type pageInfo struct {
	PageID int    `json:"pageid"`
	NS     int    `json:"ns"`
	Title  string `json:"title"`
}

type parsedPage struct {
	title        string
	displayTitle string
	html         string
	revID        int
	categories   []string
}

func newWikiClient(apiURL, cookie string) *wikiClient {
	return &wikiClient{
		apiURL: apiURL,
		cookie: cookie,
		http:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *wikiClient) getAllPages(ctx context.Context) ([]pageInfo, error) {
	var pages []pageInfo
	cont := ""
	for {
		params := url.Values{
			"action":      {"query"},
			"list":        {"allpages"},
			"apnamespace": {"0"},
			"aplimit":     {"max"},
			"format":      {"json"},
		}
		if cont != "" {
			params.Set("apcontinue", cont)
		}
		body, err := c.get(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("allpages: %w", err)
		}
		var resp struct {
			Query struct {
				AllPages []pageInfo `json:"allpages"`
			} `json:"query"`
			Continue struct {
				APContinue string `json:"apcontinue"`
			} `json:"continue"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("allpages decode: %w", err)
		}
		pages = append(pages, resp.Query.AllPages...)
		if resp.Continue.APContinue == "" {
			break
		}
		cont = resp.Continue.APContinue
	}
	return pages, nil
}

func (c *wikiClient) parsePage(ctx context.Context, title string) (*parsedPage, error) {
	params := url.Values{
		"action": {"parse"},
		"page":   {title},
		"format": {"json"},
		"prop":   {"text|displaytitle|revid|categories"},
	}
	body, err := c.get(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("parse %q: %w", title, err)
	}
	var resp struct {
		Error *struct {
			Info string `json:"info"`
		} `json:"error"`
		Parse struct {
			Title        string `json:"title"`
			DisplayTitle string `json:"displaytitle"`
			Text         struct {
				Content string `json:"*"`
			} `json:"text"`
			RevID      int `json:"revid"`
			Categories []struct {
				Name string `json:"*"`
			} `json:"categories"`
		} `json:"parse"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse decode %q: %w", title, err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("parse %q: %s", title, resp.Error.Info)
	}
	if strings.Contains(resp.Parse.Text.Content, `class="redirectMsg"`) {
		return nil, nil
	}
	cats := make([]string, len(resp.Parse.Categories))
	for i, cat := range resp.Parse.Categories {
		cats[i] = cat.Name
	}
	return &parsedPage{
		title:        resp.Parse.Title,
		displayTitle: resp.Parse.DisplayTitle,
		html:         resp.Parse.Text.Content,
		revID:        resp.Parse.RevID,
		categories:   cats,
	}, nil
}

func (c *wikiClient) get(ctx context.Context, params url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.apiURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	if c.cookie != "" {
		req.Header.Set("Cookie", c.cookie)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func encodeWikiTitle(title string) string {
	return strings.ReplaceAll(url.PathEscape(title), "%2F", "/")
}

// --- HTML → Markdown ---

var mdConverter = converter.NewConverter(
	converter.WithPlugins(
		base.NewBasePlugin(),
		commonmark.NewCommonmarkPlugin(
			commonmark.WithHeadingStyle(commonmark.HeadingStyleATX),
		),
		table.NewTablePlugin(),
	),
)

func htmlToMarkdown(rawHTML, wikiURL string) (string, error) {
	cleaned := stripMWNoise(rawHTML)
	var opts []converter.ConvertOptionFunc
	if wikiURL != "" {
		opts = append(opts, converter.WithDomain(wikiURL))
	}
	return mdConverter.ConvertString(cleaned, opts...)
}

func cleanDisplayTitle(displayTitle string) string {
	doc, err := html.Parse(strings.NewReader(displayTitle))
	if err != nil {
		return displayTitle
	}
	var buf strings.Builder
	extractText(doc, &buf)
	return strings.TrimSpace(buf.String())
}

func extractText(n *html.Node, buf *strings.Builder) {
	if n.Type == html.TextNode {
		buf.WriteString(n.Data)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractText(c, buf)
	}
}

func stripMWNoise(rawHTML string) string {
	doc, err := html.Parse(strings.NewReader(rawHTML))
	if err != nil {
		return rawHTML
	}
	removeNodes(doc)
	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return rawHTML
	}
	return buf.String()
}

func removeNodes(n *html.Node) {
	var next *html.Node
	for c := n.FirstChild; c != nil; c = next {
		next = c.NextSibling
		if shouldRemove(c) {
			n.RemoveChild(c)
		} else {
			removeNodes(c)
		}
	}
}

func shouldRemove(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	for _, a := range n.Attr {
		if a.Key == "class" {
			for _, cls := range strings.Fields(a.Val) {
				switch cls {
				case "mw-editsection", "navbox", "reflist", "references":
					return true
				}
			}
		}
		if a.Key == "id" && a.Val == "toc" {
			return true
		}
	}
	return n.DataAtom == atom.Style || n.DataAtom == atom.Script
}
