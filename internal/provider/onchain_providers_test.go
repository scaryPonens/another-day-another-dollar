package provider

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace"
)

func TestBTCMempoolOnChainProvider(t *testing.T) {
	p := NewBTCMempoolOnChainProvider(trace.NewNoopTracerProvider().Tracer("test"), "https://example.com")
	p.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v1/statistics/24h" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		body := `[{"count":150000,"vbytes_per_second":2000,"min_fee":4,"total_fee":4000000}]`
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(body)), Header: make(http.Header)}, nil
	})}

	snap, err := p.FetchSnapshot(context.Background(), "1h", time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.Symbol != "BTC" || snap.ProviderKey != "btc_mempool" {
		t.Fatalf("unexpected snapshot id: %+v", snap)
	}
	if snap.Score < -1 || snap.Score > 1 || snap.Confidence < 0 || snap.Confidence > 1 {
		t.Fatalf("score/conf bounds violated: %+v", snap)
	}
}

func TestETHBlockscoutOnChainProvider(t *testing.T) {
	p := NewETHBlockscoutOnChainProvider(trace.NewNoopTracerProvider().Tracer("test"), "https://example.com")
	p.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/v2/stats" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		body := `{"transactions_today":"2000000","network_utilization_percentage":55.0,"gas_prices":{"average":12.5}}`
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(body)), Header: make(http.Header)}, nil
	})}

	snap, err := p.FetchSnapshot(context.Background(), "1h", time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.Symbol != "ETH" || snap.ProviderKey != "eth_blockscout" {
		t.Fatalf("unexpected snapshot id: %+v", snap)
	}
}

func TestADAKoiosOnChainProvider(t *testing.T) {
	p := NewADAKoiosOnChainProvider(trace.NewNoopTracerProvider().Tracer("test"), "https://example.com")
	p.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		var body string
		switch {
		case req.URL.Path == "/api/v1/totals":
			body = `[{"epoch_no":612,"fees":"64349910572"}]`
		case req.URL.Path == "/api/v1/epoch_info":
			body = `[{"tx_count":135737,"start_time":1770587091,"end_time":1771019091}]`
		default:
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(body)), Header: make(http.Header)}, nil
	})}

	snap, err := p.FetchSnapshot(context.Background(), "4h", time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.Symbol != "ADA" || snap.ProviderKey != "ada_koios" {
		t.Fatalf("unexpected snapshot id: %+v", snap)
	}
}

func TestXRPScanOnChainProvider(t *testing.T) {
	p := NewXRPScanOnChainProvider(trace.NewNoopTracerProvider().Tracer("test"), "https://example.com")
	p.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		var body string
		switch {
		case strings.HasSuffix(req.URL.Path, "/api/v1/network/fee"):
			body = `{"current_queue_size":"5","expected_ledger_size":"400","drops":{"median_fee":"128000"}}`
		case strings.HasSuffix(req.URL.Path, "/api/v1/network/server_info"):
			body = `{"info":{"load_factor":1.2}}`
		default:
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(body)), Header: make(http.Header)}, nil
	})}

	snap, err := p.FetchSnapshot(context.Background(), "1h", time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.Symbol != "XRP" || snap.ProviderKey != "xrp_xrpscan" {
		t.Fatalf("unexpected snapshot id: %+v", snap)
	}
}
