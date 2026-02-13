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

type XRPScanOnChainProvider struct {
	client  *http.Client
	baseURL string
	tracer  trace.Tracer
}

func NewXRPScanOnChainProvider(tracer trace.Tracer, baseURL string) *XRPScanOnChainProvider {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = "https://api.xrpscan.com"
	}
	return &XRPScanOnChainProvider{
		client:  &http.Client{Timeout: 20 * time.Second},
		baseURL: strings.TrimRight(baseURL, "/"),
		tracer:  tracer,
	}
}

func (p *XRPScanOnChainProvider) FetchSnapshot(ctx context.Context, interval string, bucketTime time.Time) (*OnChainSnapshot, error) {
	_, span := p.tracer.Start(ctx, "onchain.xrp-xrpscan.fetch")
	defer span.End()

	queueSize, expectedLedgerSize, medianFee, err := p.fetchFeeMetrics(ctx)
	if err != nil {
		return nil, err
	}
	loadFactor, err := p.fetchLoadFactor(ctx)
	if err != nil {
		return nil, err
	}

	queueNorm := clamp((queueSize/expectedLedgerSize)-0.35, -1, 1)
	feeNorm := clamp((medianFee-128000.0)/300000.0, -1, 1)
	loadNorm := clamp((loadFactor-1.0)/5.0, -1, 1)

	score := clamp((0.40*loadNorm)-(0.40*queueNorm)-(0.20*feeNorm), -1, 1)
	confidence := confidenceFromScore(score)

	return &OnChainSnapshot{
		ProviderKey: "xrp_xrpscan",
		Symbol:      "XRP",
		Interval:    interval,
		BucketTime:  bucketTime.UTC(),
		Score:       score,
		Confidence:  confidence,
		Metrics: map[string]float64{
			"current_queue_size":   queueSize,
			"expected_ledger_size": expectedLedgerSize,
			"median_fee":           medianFee,
			"load_factor":          loadFactor,
		},
	}, nil
}

func (p *XRPScanOnChainProvider) fetchFeeMetrics(ctx context.Context) (float64, float64, float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/api/v1/network/fee", nil)
	if err != nil {
		return 0, 1, 0, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return 0, 1, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, 1, 0, fmt.Errorf("xrpscan fee error %d: %s", resp.StatusCode, string(body))
	}

	var payload struct {
		CurrentQueueSize   any `json:"current_queue_size"`
		ExpectedLedgerSize any `json:"expected_ledger_size"`
		Drops              struct {
			MedianFee any `json:"median_fee"`
		} `json:"drops"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, 1, 0, fmt.Errorf("decode xrpscan fee payload: %w", err)
	}

	queue := asFloat(payload.CurrentQueueSize)
	expected := asFloat(payload.ExpectedLedgerSize)
	if expected <= 0 {
		expected = 1
	}
	medianFee := asFloat(payload.Drops.MedianFee)
	return queue, expected, medianFee, nil
}

func (p *XRPScanOnChainProvider) fetchLoadFactor(ctx context.Context) (float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/api/v1/network/server_info", nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("xrpscan server_info error %d: %s", resp.StatusCode, string(body))
	}

	var payload struct {
		Info struct {
			LoadFactor any `json:"load_factor"`
		} `json:"info"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, fmt.Errorf("decode xrpscan server_info payload: %w", err)
	}
	load := asFloat(payload.Info.LoadFactor)
	if load <= 0 {
		load = 1
	}
	return load, nil
}
