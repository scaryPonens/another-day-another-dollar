package provider

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestRedditFetchHot(t *testing.T) {
	p := NewRedditProvider(trace.NewNoopTracerProvider().Tracer("test"))
	p.baseURL = "https://example.com"
	p.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/r/Bitcoin/hot.json" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		if req.Header.Get("User-Agent") == "" {
			t.Fatalf("expected user-agent header")
		}
		body := `{"data":{"children":[{"data":{"id":"abc123","subreddit":"Bitcoin","title":"BTC breaks out","selftext":"Market is moving up","author":"alice","created_utc":1771009800,"permalink":"/r/Bitcoin/comments/abc123/post","url":"https://example.com/fallback","score":10,"num_comments":3}}]}}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(body)),
			Header:     make(http.Header),
		}, nil
	})}

	items, err := p.FetchHot(context.Background(), "Bitcoin", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	item := items[0]
	if item.Source != "reddit" || item.SourceItemID != "abc123" {
		t.Fatalf("unexpected item ids: %+v", item)
	}
	if item.URL != "https://example.com/r/Bitcoin/comments/abc123/post" {
		t.Fatalf("unexpected permalink url: %s", item.URL)
	}
	if item.Metadata["subreddit"] != "Bitcoin" {
		t.Fatalf("expected subreddit metadata, got %+v", item.Metadata)
	}
}
