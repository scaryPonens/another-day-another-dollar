package marketintel

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"bug-free-umbrella/internal/domain"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type SentimentScore struct {
	ItemID     int64
	Score      float64
	Confidence float64
	Label      string
	Model      string
	Reason     string
}

type BatchLLMScorer interface {
	ScoreBatch(ctx context.Context, items []domain.MarketIntelItem) ([]SentimentScore, error)
}

type Scorer struct {
	llm       BatchLLMScorer
	batchSize int
}

func NewScorer(llm BatchLLMScorer, batchSize int) *Scorer {
	if batchSize <= 0 {
		batchSize = 24
	}
	return &Scorer{llm: llm, batchSize: batchSize}
}

func (s *Scorer) Score(ctx context.Context, items []domain.MarketIntelItem) ([]SentimentScore, error) {
	if len(items) == 0 {
		return nil, nil
	}

	resultByID := make(map[int64]SentimentScore, len(items))
	for _, item := range items {
		score, confidence, label, reason := HeuristicSentiment(item.Title, item.Excerpt)
		resultByID[item.ID] = SentimentScore{
			ItemID:     item.ID,
			Score:      score,
			Confidence: confidence,
			Label:      label,
			Reason:     reason,
			Model:      "heuristic:v1",
		}
	}

	if s.llm != nil {
		for start := 0; start < len(items); start += s.batchSize {
			end := start + s.batchSize
			if end > len(items) {
				end = len(items)
			}
			batch := items[start:end]
			scored, err := s.llm.ScoreBatch(ctx, batch)
			if err != nil {
				continue
			}
			for _, row := range scored {
				current, ok := resultByID[row.ItemID]
				if !ok {
					continue
				}
				current.Score = clamp(row.Score, -1, 1)
				current.Confidence = clamp(row.Confidence, 0, 1)
				current.Label = normalizeLabel(row.Label)
				current.Reason = strings.TrimSpace(row.Reason)
				if current.Reason == "" {
					current.Reason = "llm"
				}
				if row.Model != "" {
					current.Model = row.Model
				}
				resultByID[row.ItemID] = current
			}
		}
	}

	out := make([]SentimentScore, 0, len(items))
	for _, item := range items {
		if scored, ok := resultByID[item.ID]; ok {
			out = append(out, scored)
		}
	}
	return out, nil
}

func HeuristicSentiment(title, excerpt string) (float64, float64, string, string) {
	text := strings.ToLower(strings.TrimSpace(title + " " + excerpt))
	if text == "" {
		return 0, 0.25, "neutral", "empty-text"
	}

	bullish := []string{"bull", "breakout", "surge", "rally", "adoption", "outflow", "growth", "buy", "uptrend", "recover"}
	bearish := []string{"bear", "dump", "sell", "crash", "hack", "lawsuit", "ban", "inflow", "decline", "downtrend", "liquidation"}

	bullCount := countMatches(text, bullish)
	bearCount := countMatches(text, bearish)

	raw := float64(bullCount-bearCount) / float64(bullCount+bearCount+1)
	score := clamp(raw, -1, 1)
	confidence := clamp(0.35+(0.1*float64(absInt(bullCount-bearCount))), 0.25, 0.70)

	label := "neutral"
	if score > 0.2 {
		label = "bullish"
	} else if score < -0.2 {
		label = "bearish"
	}
	reason := fmt.Sprintf("heuristic keywords bull=%d bear=%d", bullCount, bearCount)
	return score, confidence, label, reason
}

func countMatches(text string, tokens []string) int {
	count := 0
	for _, token := range tokens {
		if strings.Contains(text, token) {
			count++
		}
	}
	return count
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func normalizeLabel(label string) string {
	label = strings.ToLower(strings.TrimSpace(label))
	switch label {
	case "bull", "bullish", "positive":
		return "bullish"
	case "bear", "bearish", "negative":
		return "bearish"
	default:
		return "neutral"
	}
}

type openAIChatClient interface {
	CreateChatCompletion(ctx context.Context, params openai.ChatCompletionNewParams) (*openai.ChatCompletion, error)
}

type OpenAIScorer struct {
	client openAIChatClient
	model  string
}

func NewOpenAIScorer(apiKey string, model string) *OpenAIScorer {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil
	}
	if strings.TrimSpace(model) == "" {
		model = "gpt-4o-mini"
	}
	client := openai.NewClient(option.WithAPIKey(apiKey))
	return &OpenAIScorer{
		client: &openAIClient{client: client},
		model:  model,
	}
}

func (s *OpenAIScorer) ScoreBatch(ctx context.Context, items []domain.MarketIntelItem) ([]SentimentScore, error) {
	if s == nil || s.client == nil || len(items) == 0 {
		return nil, nil
	}

	var sb strings.Builder
	for _, item := range items {
		sb.WriteString(fmt.Sprintf("id=%d\n", item.ID))
		sb.WriteString(fmt.Sprintf("title=%s\n", strings.TrimSpace(item.Title)))
		sb.WriteString(fmt.Sprintf("excerpt=%s\n\n", strings.TrimSpace(item.Excerpt)))
	}

	systemPrompt := "You score crypto sentiment. Return ONLY JSON array. Each object requires: id (int), score (-1..1), confidence (0..1), label (bullish|neutral|bearish), reason (short text). No markdown."
	userPrompt := "Items:\n" + sb.String()

	completion, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionNewParams{
		Model: s.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userPrompt),
		},
	})
	if err != nil {
		return nil, err
	}
	if len(completion.Choices) == 0 {
		return nil, fmt.Errorf("empty scorer completion")
	}

	raw := strings.TrimSpace(completion.Choices[0].Message.Content)
	raw = trimCodeFence(raw)

	var parsed []struct {
		ID         int64   `json:"id"`
		Score      float64 `json:"score"`
		Confidence float64 `json:"confidence"`
		Label      string  `json:"label"`
		Reason     string  `json:"reason"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("parse scorer json: %w", err)
	}

	byID := make(map[int64]struct{}, len(items))
	for _, item := range items {
		byID[item.ID] = struct{}{}
	}

	out := make([]SentimentScore, 0, len(parsed))
	for _, row := range parsed {
		if _, ok := byID[row.ID]; !ok {
			continue
		}
		out = append(out, SentimentScore{
			ItemID:     row.ID,
			Score:      clamp(row.Score, -1, 1),
			Confidence: clamp(row.Confidence, 0, 1),
			Label:      normalizeLabel(row.Label),
			Reason:     strings.TrimSpace(row.Reason),
			Model:      "llm:" + s.model,
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ItemID < out[j].ItemID })
	return out, nil
}

func trimCodeFence(v string) string {
	v = strings.TrimSpace(v)
	if strings.HasPrefix(v, "```") {
		v = strings.TrimPrefix(v, "```")
		v = strings.TrimSpace(v)
		if strings.HasPrefix(strings.ToLower(v), "json") {
			v = strings.TrimSpace(v[4:])
		}
		v = strings.TrimSuffix(v, "```")
		v = strings.TrimSpace(v)
	}
	return v
}

type openAIClient struct {
	client openai.Client
}

func (c *openAIClient) CreateChatCompletion(ctx context.Context, params openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	return c.client.Chat.Completions.New(ctx, params)
}
