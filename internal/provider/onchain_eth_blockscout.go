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

type ETHBlockscoutOnChainProvider struct {
	client  *http.Client
	baseURL string
	tracer  trace.Tracer
}

func NewETHBlockscoutOnChainProvider(tracer trace.Tracer, baseURL string) *ETHBlockscoutOnChainProvider {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = "https://eth.blockscout.com"
	}
	return &ETHBlockscoutOnChainProvider{
		client:  &http.Client{Timeout: 20 * time.Second},
		baseURL: strings.TrimRight(baseURL, "/"),
		tracer:  tracer,
	}
}

func (p *ETHBlockscoutOnChainProvider) FetchSnapshot(ctx context.Context, interval string, bucketTime time.Time) (*OnChainSnapshot, error) {
	_, span := p.tracer.Start(ctx, "onchain.eth-blockscout.fetch")
	defer span.End()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/api/v2/stats", nil)
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
		return nil, fmt.Errorf("eth blockscout error %d: %s", resp.StatusCode, string(body))
	}

	var payload struct {
		TransactionsToday            any `json:"transactions_today"`
		NetworkUtilizationPercentage any `json:"network_utilization_percentage"`
		GasPrices                    struct {
			Average any `json:"average"`
		} `json:"gas_prices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode eth blockscout payload: %w", err)
	}

	txToday := asFloat(payload.TransactionsToday)
	utilization := asFloat(payload.NetworkUtilizationPercentage)
	gasAvg := asFloat(payload.GasPrices.Average)

	txNorm := clamp((txToday-1_500_000.0)/1_500_000.0, -1, 1)
	utilNorm := clamp((utilization-45.0)/55.0, -1, 1)
	gasPenalty := clamp((gasAvg-25.0)/120.0, -1, 1)

	score := clamp((0.45*txNorm)+(0.35*utilNorm)-(0.20*gasPenalty), -1, 1)
	confidence := confidenceFromScore(score)

	return &OnChainSnapshot{
		ProviderKey: "eth_blockscout",
		Symbol:      "ETH",
		Interval:    interval,
		BucketTime:  bucketTime.UTC(),
		Score:       score,
		Confidence:  confidence,
		Metrics: map[string]float64{
			"transactions_today":             txToday,
			"network_utilization_percentage": utilization,
			"gas_price_average":              gasAvg,
		},
	}, nil
}
