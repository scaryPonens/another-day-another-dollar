package provider

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace"
)

func TestFearGreedFetchLatest(t *testing.T) {
	p := NewFearGreedProvider(trace.NewNoopTracerProvider().Tracer("test"))
	p.baseURL = "https://example.com"
	p.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/fng/" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		body := `{"data":[{"value":"63","value_classification":"Greed","timestamp":"1771009800","time_until_update":"1111"}]}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(body)),
			Header:     make(http.Header),
		}, nil
	})}

	point, err := p.FetchLatest(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if point.Value != 63 || point.Classification != "Greed" || point.TimeUntilUpdateS != 1111 {
		t.Fatalf("unexpected point: %+v", point)
	}
	if !point.Timestamp.Equal(time.Unix(1771009800, 0).UTC()) {
		t.Fatalf("unexpected timestamp: %v", point.Timestamp)
	}
}
