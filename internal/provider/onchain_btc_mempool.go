package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"
)

type BTCMempoolOnChainProvider struct {
	client  *http.Client
	baseURL string
	tracer  trace.Tracer
}

func NewBTCMempoolOnChainProvider(tracer trace.Tracer, baseURL string) *BTCMempoolOnChainProvider {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = "https://mempool.space"
	}
	return &BTCMempoolOnChainProvider{
		client:  &http.Client{Timeout: 20 * time.Second},
		baseURL: strings.TrimRight(baseURL, "/"),
		tracer:  tracer,
	}
}

func (p *BTCMempoolOnChainProvider) FetchSnapshot(ctx context.Context, interval string, bucketTime time.Time) (*OnChainSnapshot, error) {
	_, span := p.tracer.Start(ctx, "onchain.btc-mempool.fetch")
	defer span.End()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/api/v1/statistics/24h", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("btc mempool error %d: %s", resp.StatusCode, string(body))
	}

	var rows []struct {
		Count           float64 `json:"count"`
		VBytesPerSecond float64 `json:"vbytes_per_second"`
		MinFee          float64 `json:"min_fee"`
		TotalFee        float64 `json:"total_fee"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, fmt.Errorf("decode btc mempool payload: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("btc mempool payload has no rows")
	}

	r := rows[0]
	countNorm := clamp((r.Count-120000.0)/180000.0, -1, 1)
	throughputNorm := clamp((r.VBytesPerSecond-1200.0)/2400.0, -1, 1)
	feeLoadNorm := clamp((r.MinFee-5.0)/40.0, -1, 1)
	totalFeeNorm := clamp((r.TotalFee-2_000_000.0)/8_000_000.0, -1, 1)

	score := clamp((0.35*countNorm)+(0.35*throughputNorm)+(0.15*totalFeeNorm)-(0.15*feeLoadNorm), -1, 1)
	confidence := confidenceFromScore(score)

	return &OnChainSnapshot{
		ProviderKey: "btc_mempool",
		Symbol:      "BTC",
		Interval:    interval,
		BucketTime:  bucketTime.UTC(),
		Score:       score,
		Confidence:  confidence,
		Metrics: map[string]float64{
			"count":             r.Count,
			"vbytes_per_second": r.VBytesPerSecond,
			"min_fee":           r.MinFee,
			"total_fee":         r.TotalFee,
		},
	}, nil
}
