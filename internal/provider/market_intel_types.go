package provider

import "time"

type FearGreedPoint struct {
	Value            int
	Classification   string
	Timestamp        time.Time
	TimeUntilUpdateS int
}

type ContentItem struct {
	Source       string
	SourceItemID string
	Title        string
	URL          string
	Excerpt      string
	Author       string
	PublishedAt  time.Time
	Metadata     map[string]any
}

type OnChainSnapshot struct {
	ProviderKey string
	Symbol      string
	Interval    string
	BucketTime  time.Time
	Score       float64
	Confidence  float64
	Metrics     map[string]float64
}
