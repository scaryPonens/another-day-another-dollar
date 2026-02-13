package provider

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestRSSFetchFeed(t *testing.T) {
	p := NewRSSProvider(trace.NewNoopTracerProvider().Tracer("test"))
	p.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		xml := `<?xml version="1.0"?><rss version="2.0"><channel><title>Example Feed</title><item><title>ETH adoption rises</title><link>https://news.example/eth</link><description><![CDATA[<p>Ethereum growth continues</p>]]></description><guid>guid-1</guid><pubDate>Fri, 13 Feb 2026 10:00:00 +0000</pubDate><author>Reporter</author></item></channel></rss>`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(xml)),
			Header:     make(http.Header),
		}, nil
	})}

	items, err := p.FetchFeed(context.Background(), "https://news.example/rss", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	item := items[0]
	if item.Source != "news" || item.SourceItemID != "guid-1" {
		t.Fatalf("unexpected item: %+v", item)
	}
	if item.Excerpt != "Ethereum growth continues" {
		t.Fatalf("expected html stripped excerpt, got %q", item.Excerpt)
	}
}
