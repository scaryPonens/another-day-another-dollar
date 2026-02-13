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

type ADAKoiosOnChainProvider struct {
	client  *http.Client
	baseURL string
	tracer  trace.Tracer
}

func NewADAKoiosOnChainProvider(tracer trace.Tracer, baseURL string) *ADAKoiosOnChainProvider {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = "https://api.koios.rest"
	}
	return &ADAKoiosOnChainProvider{
		client:  &http.Client{Timeout: 20 * time.Second},
		baseURL: strings.TrimRight(baseURL, "/"),
		tracer:  tracer,
	}
}

func (p *ADAKoiosOnChainProvider) FetchSnapshot(ctx context.Context, interval string, bucketTime time.Time) (*OnChainSnapshot, error) {
	_, span := p.tracer.Start(ctx, "onchain.ada-koios.fetch")
	defer span.End()

	latestEpoch, fees, err := p.fetchTotals(ctx)
	if err != nil {
		return nil, err
	}
	txCount, pacePerHour, err := p.fetchEpochMetrics(ctx, latestEpoch)
	if err != nil {
		return nil, err
	}

	txNorm := clamp((txCount-120000.0)/180000.0, -1, 1)
	feeNorm := clamp((fees-45_000_000_000.0)/120_000_000_000.0, -1, 1)
	paceNorm := clamp((pacePerHour-300.0)/800.0, -1, 1)

	score := clamp((0.5*txNorm)+(0.25*feeNorm)+(0.25*paceNorm), -1, 1)
	confidence := confidenceFromScore(score)

	return &OnChainSnapshot{
		ProviderKey: "ada_koios",
		Symbol:      "ADA",
		Interval:    interval,
		BucketTime:  bucketTime.UTC(),
		Score:       score,
		Confidence:  confidence,
		Metrics: map[string]float64{
			"epoch":            float64(latestEpoch),
			"tx_count":         txCount,
			"fees":             fees,
			"tx_pace_per_hour": pacePerHour,
		},
	}, nil
}

func (p *ADAKoiosOnChainProvider) fetchTotals(ctx context.Context) (int, float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/api/v1/totals", nil)
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, 0, fmt.Errorf("koios totals error %d: %s", resp.StatusCode, string(body))
	}

	var rows []struct {
		EpochNo int    `json:"epoch_no"`
		Fees    string `json:"fees"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return 0, 0, fmt.Errorf("decode koios totals payload: %w", err)
	}
	if len(rows) == 0 {
		return 0, 0, fmt.Errorf("koios totals payload has no rows")
	}
	return rows[0].EpochNo, parseFloatString(rows[0].Fees), nil
}

func (p *ADAKoiosOnChainProvider) fetchEpochMetrics(ctx context.Context, epoch int) (float64, float64, error) {
	query := url.Values{}
	query.Set("_epoch_no", fmt.Sprintf("%d", epoch))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/api/v1/epoch_info?"+query.Encode(), nil)
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, 0, fmt.Errorf("koios epoch_info error %d: %s", resp.StatusCode, string(body))
	}

	var rows []struct {
		TxCount   float64 `json:"tx_count"`
		StartTime int64   `json:"start_time"`
		EndTime   int64   `json:"end_time"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return 0, 0, fmt.Errorf("decode koios epoch_info payload: %w", err)
	}
	if len(rows) == 0 {
		return 0, 0, fmt.Errorf("koios epoch_info payload has no rows")
	}

	txCount := rows[0].TxCount
	durationHours := 1.0
	if rows[0].EndTime > rows[0].StartTime {
		durationHours = float64(rows[0].EndTime-rows[0].StartTime) / 3600.0
		if durationHours <= 0 {
			durationHours = 1
		}
	}
	pacePerHour := txCount / durationHours
	return txCount, pacePerHour, nil
}
