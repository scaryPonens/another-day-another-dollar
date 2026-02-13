package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"
)

const (
	redditBaseURL     = "https://www.reddit.com"
	defaultRedditUA   = "bug-free-umbrella/1.0 (+https://github.com/scaryPonens/bug-free-umbrella)"
	defaultRedditSize = 40
)

type RedditProvider struct {
	client    *http.Client
	baseURL   string
	userAgent string
	tracer    trace.Tracer
}

func NewRedditProvider(tracer trace.Tracer) *RedditProvider {
	return &RedditProvider{
		client:    &http.Client{Timeout: 20 * time.Second},
		baseURL:   redditBaseURL,
		userAgent: defaultRedditUA,
		tracer:    tracer,
	}
}

func (p *RedditProvider) FetchHot(ctx context.Context, subreddit string, limit int) ([]ContentItem, error) {
	_, span := p.tracer.Start(ctx, "reddit.fetch-hot")
	defer span.End()

	subreddit = strings.TrimSpace(subreddit)
	if subreddit == "" {
		return nil, fmt.Errorf("subreddit is required")
	}
	if limit <= 0 {
		limit = defaultRedditSize
	}
	if limit > 100 {
		limit = 100
	}

	base := strings.TrimRight(p.baseURL, "/")
	u := fmt.Sprintf("%s/r/%s/hot.json?limit=%d", base, url.PathEscape(subreddit), limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if p.userAgent != "" {
		req.Header.Set("User-Agent", p.userAgent)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("reddit API error %d: %s", resp.StatusCode, string(body))
	}

	var payload struct {
		Data struct {
			Children []struct {
				Data struct {
					ID          string  `json:"id"`
					Subreddit   string  `json:"subreddit"`
					Title       string  `json:"title"`
					SelfText    string  `json:"selftext"`
					Author      string  `json:"author"`
					CreatedUTC  float64 `json:"created_utc"`
					Permalink   string  `json:"permalink"`
					URL         string  `json:"url"`
					Score       float64 `json:"score"`
					NumComments float64 `json:"num_comments"`
				} `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode reddit response: %w", err)
	}

	items := make([]ContentItem, 0, len(payload.Data.Children))
	for _, row := range payload.Data.Children {
		data := row.Data
		if strings.TrimSpace(data.ID) == "" || strings.TrimSpace(data.Title) == "" {
			continue
		}
		publishedAt := time.Unix(int64(data.CreatedUTC), 0).UTC()
		permalink := strings.TrimSpace(data.Permalink)
		itemURL := strings.TrimSpace(data.URL)
		if permalink != "" {
			itemURL = base + permalink
		}
		metadata := map[string]any{
			"subreddit":    strings.TrimSpace(data.Subreddit),
			"score":        data.Score,
			"num_comments": data.NumComments,
		}
		items = append(items, ContentItem{
			Source:       "reddit",
			SourceItemID: data.ID,
			Title:        sanitizeText(data.Title, 300),
			URL:          itemURL,
			Excerpt:      sanitizeText(data.SelfText, 420),
			Author:       sanitizeText(data.Author, 120),
			PublishedAt:  publishedAt,
			Metadata:     metadata,
		})
	}

	return items, nil
}

func sanitizeText(in string, maxLen int) string {
	in = strings.TrimSpace(in)
	if in == "" {
		return ""
	}
	in = strings.ReplaceAll(in, "\n", " ")
	in = strings.ReplaceAll(in, "\r", " ")
	in = strings.Join(strings.Fields(in), " ")
	if maxLen > 0 && len(in) > maxLen {
		in = in[:maxLen]
	}
	return in
}
