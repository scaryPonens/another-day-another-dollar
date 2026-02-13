package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"
)

const fearGreedBaseURL = "https://api.alternative.me"

type FearGreedProvider struct {
	client  *http.Client
	baseURL string
	tracer  trace.Tracer
}

func NewFearGreedProvider(tracer trace.Tracer) *FearGreedProvider {
	return &FearGreedProvider{
		client:  &http.Client{Timeout: 15 * time.Second},
		baseURL: fearGreedBaseURL,
		tracer:  tracer,
	}
}

func (p *FearGreedProvider) FetchLatest(ctx context.Context) (*FearGreedPoint, error) {
	_, span := p.tracer.Start(ctx, "feargreed.fetch-latest")
	defer span.End()

	url := strings.TrimRight(p.baseURL, "/") + "/fng/?limit=1"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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
		return nil, fmt.Errorf("fear & greed API error %d: %s", resp.StatusCode, string(body))
	}

	var payload struct {
		Data []struct {
			Value            string `json:"value"`
			Classification   string `json:"value_classification"`
			Timestamp        string `json:"timestamp"`
			TimeUntilUpdateS string `json:"time_until_update"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode fear & greed response: %w", err)
	}
	if len(payload.Data) == 0 {
		return nil, fmt.Errorf("fear & greed response has no rows")
	}

	row := payload.Data[0]
	value, err := strconv.Atoi(strings.TrimSpace(row.Value))
	if err != nil {
		return nil, fmt.Errorf("parse fear & greed value: %w", err)
	}
	ts, err := strconv.ParseInt(strings.TrimSpace(row.Timestamp), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse fear & greed timestamp: %w", err)
	}
	if ts > 1_000_000_000_000 {
		ts = ts / 1000
	}
	updateS := 0
	if row.TimeUntilUpdateS != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(row.TimeUntilUpdateS)); err == nil && n >= 0 {
			updateS = n
		}
	}

	return &FearGreedPoint{
		Value:            value,
		Classification:   row.Classification,
		Timestamp:        time.Unix(ts, 0).UTC(),
		TimeUntilUpdateS: updateS,
	}, nil
}
