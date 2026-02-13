package provider

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"
)

type RSSProvider struct {
	client *http.Client
	tracer trace.Tracer
}

func NewRSSProvider(tracer trace.Tracer) *RSSProvider {
	return &RSSProvider{
		client: &http.Client{Timeout: 20 * time.Second},
		tracer: tracer,
	}
}

func (p *RSSProvider) FetchFeed(ctx context.Context, feedURL string, maxItems int) ([]ContentItem, error) {
	_, span := p.tracer.Start(ctx, "rss.fetch-feed")
	defer span.End()

	feedURL = strings.TrimSpace(feedURL)
	if feedURL == "" {
		return nil, fmt.Errorf("feed url is required")
	}
	if maxItems <= 0 {
		maxItems = 40
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("rss fetch error %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rss struct {
		Channel struct {
			Title string `xml:"title"`
			Items []struct {
				Title       string `xml:"title"`
				Link        string `xml:"link"`
				Description string `xml:"description"`
				GUID        string `xml:"guid"`
				PubDate     string `xml:"pubDate"`
				Creator     string `xml:"creator"`
				Author      string `xml:"author"`
			} `xml:"item"`
		} `xml:"channel"`
	}
	if err := xml.Unmarshal(body, &rss); err != nil {
		return nil, fmt.Errorf("decode rss payload: %w", err)
	}

	items := make([]ContentItem, 0, min(maxItems, len(rss.Channel.Items)))
	for i, row := range rss.Channel.Items {
		if i >= maxItems {
			break
		}
		title := sanitizeText(row.Title, 300)
		if title == "" {
			continue
		}
		publishedAt := parseRSSDate(row.PubDate)
		if publishedAt.IsZero() {
			publishedAt = time.Now().UTC()
		}
		author := sanitizeText(row.Creator, 120)
		if author == "" {
			author = sanitizeText(row.Author, 120)
		}
		sourceID := sanitizeText(row.GUID, 250)
		if sourceID == "" {
			sourceID = sanitizeText(row.Link, 250)
		}
		if sourceID == "" {
			h := sha1.Sum([]byte(title + "|" + publishedAt.Format(time.RFC3339Nano)))
			sourceID = hex.EncodeToString(h[:])
		}

		items = append(items, ContentItem{
			Source:       "news",
			SourceItemID: sourceID,
			Title:        title,
			URL:          sanitizeText(row.Link, 500),
			Excerpt:      sanitizeText(htmlStrip(row.Description), 420),
			Author:       author,
			PublishedAt:  publishedAt.UTC(),
			Metadata: map[string]any{
				"feed_url": feedURL,
				"channel":  sanitizeText(rss.Channel.Title, 120),
			},
		})
	}

	return items, nil
}

func parseRSSDate(v string) time.Time {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}
	}
	layouts := []string{time.RFC1123Z, time.RFC1123, time.RFC822Z, time.RFC822, time.RFC3339}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, v); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

func htmlStrip(in string) string {
	if strings.TrimSpace(in) == "" {
		return ""
	}
	var b strings.Builder
	inside := false
	for _, r := range in {
		switch r {
		case '<':
			inside = true
			continue
		case '>':
			inside = false
			continue
		}
		if !inside {
			b.WriteRune(r)
		}
	}
	return b.String()
}
