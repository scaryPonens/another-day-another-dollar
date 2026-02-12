package advisor

import (
	"context"
	"errors"
	"testing"
	"time"

	"bug-free-umbrella/internal/domain"

	"github.com/openai/openai-go"
	"go.opentelemetry.io/otel/trace"
)

func TestAskHappyPath(t *testing.T) {
	llm := &stubLLMClient{
		response: &openai.ChatCompletion{
			Choices: []openai.ChatCompletionChoice{
				{Message: openai.ChatCompletionMessage{Content: "BTC looks bullish"}},
			},
		},
	}
	store := &stubConvStore{}
	prices := &stubPrices{
		price: &domain.PriceSnapshot{Symbol: "BTC", PriceUSD: 50000},
	}
	signals := &stubSignals{}

	svc := NewAdvisorService(
		trace.NewNoopTracerProvider().Tracer("test"),
		llm, prices, signals, store, "gpt-4o-mini", 20,
	)

	reply, err := svc.Ask(context.Background(), 123, "What about BTC?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply != "BTC looks bullish" {
		t.Fatalf("expected 'BTC looks bullish', got %q", reply)
	}
	// Verify messages were stored (user + assistant)
	if len(store.messages) != 2 {
		t.Fatalf("expected 2 stored messages, got %d", len(store.messages))
	}
	if store.messages[0].role != "user" {
		t.Fatalf("expected first stored message role=user, got %s", store.messages[0].role)
	}
	if store.messages[1].role != "assistant" {
		t.Fatalf("expected second stored message role=assistant, got %s", store.messages[1].role)
	}
}

func TestAskLLMError(t *testing.T) {
	llm := &stubLLMClient{err: errors.New("api down")}
	store := &stubConvStore{}
	prices := &stubPrices{allPrices: []*domain.PriceSnapshot{}}
	signals := &stubSignals{}

	svc := NewAdvisorService(
		trace.NewNoopTracerProvider().Tracer("test"),
		llm, prices, signals, store, "gpt-4o-mini", 20,
	)

	_, err := svc.Ask(context.Background(), 123, "What looks good?")
	if err == nil {
		t.Fatal("expected error from LLM failure")
	}
	// User message should still have been stored
	if len(store.messages) != 1 || store.messages[0].role != "user" {
		t.Fatalf("expected user message to be stored despite LLM error, got %d messages", len(store.messages))
	}
}

func TestAskConversationStoreFailureNonFatal(t *testing.T) {
	llm := &stubLLMClient{
		response: &openai.ChatCompletion{
			Choices: []openai.ChatCompletionChoice{
				{Message: openai.ChatCompletionMessage{Content: "response"}},
			},
		},
	}
	store := &stubConvStore{appendErr: errors.New("db down")}
	prices := &stubPrices{allPrices: []*domain.PriceSnapshot{}}
	signals := &stubSignals{}

	svc := NewAdvisorService(
		trace.NewNoopTracerProvider().Tracer("test"),
		llm, prices, signals, store, "gpt-4o-mini", 20,
	)

	reply, err := svc.Ask(context.Background(), 123, "test")
	if err != nil {
		t.Fatalf("store failure should be non-fatal, got: %v", err)
	}
	if reply != "response" {
		t.Fatalf("expected 'response', got %q", reply)
	}
}

func TestAskContextGatheringFailure(t *testing.T) {
	llm := &stubLLMClient{
		response: &openai.ChatCompletion{
			Choices: []openai.ChatCompletionChoice{
				{Message: openai.ChatCompletionMessage{Content: "no data available"}},
			},
		},
	}
	store := &stubConvStore{}
	prices := &stubPrices{err: errors.New("price service down")}
	signals := &stubSignals{}

	svc := NewAdvisorService(
		trace.NewNoopTracerProvider().Tracer("test"),
		llm, prices, signals, store, "gpt-4o-mini", 20,
	)

	reply, err := svc.Ask(context.Background(), 123, "What looks good?")
	if err != nil {
		t.Fatalf("context failure should be non-fatal, got: %v", err)
	}
	if reply != "no data available" {
		t.Fatalf("expected 'no data available', got %q", reply)
	}
}

func TestAskNoHistory(t *testing.T) {
	llm := &stubLLMClient{
		response: &openai.ChatCompletion{
			Choices: []openai.ChatCompletionChoice{
				{Message: openai.ChatCompletionMessage{Content: "fresh start"}},
			},
		},
	}
	store := &stubConvStore{}
	prices := &stubPrices{allPrices: []*domain.PriceSnapshot{}}
	signals := &stubSignals{}

	svc := NewAdvisorService(
		trace.NewNoopTracerProvider().Tracer("test"),
		llm, prices, signals, store, "gpt-4o-mini", 20,
	)

	reply, err := svc.Ask(context.Background(), 999, "Hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply != "fresh start" {
		t.Fatalf("expected 'fresh start', got %q", reply)
	}
}

func TestAskDefaultMaxHistory(t *testing.T) {
	svc := NewAdvisorService(
		trace.NewNoopTracerProvider().Tracer("test"),
		&stubLLMClient{}, &stubPrices{}, &stubSignals{}, &stubConvStore{},
		"gpt-4o-mini", 0,
	)
	if svc.maxHistory != 20 {
		t.Fatalf("expected default maxHistory=20, got %d", svc.maxHistory)
	}
}

// --- stubs ---

type stubLLMClient struct {
	response *openai.ChatCompletion
	err      error
}

func (s *stubLLMClient) CreateChatCompletion(ctx context.Context, params openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	return s.response, s.err
}

type storedMsg struct {
	chatID  int64
	role    string
	content string
}

type stubConvStore struct {
	messages  []storedMsg
	history   []domain.ConversationMessage
	appendErr error
	recentErr error
}

func (s *stubConvStore) AppendMessage(ctx context.Context, chatID int64, role, content string) error {
	if s.appendErr != nil {
		return s.appendErr
	}
	s.messages = append(s.messages, storedMsg{chatID: chatID, role: role, content: content})
	return nil
}

func (s *stubConvStore) RecentMessages(ctx context.Context, chatID int64, limit int) ([]domain.ConversationMessage, error) {
	if s.recentErr != nil {
		return nil, s.recentErr
	}
	// Return stored messages as history (simulates reading back what was appended)
	var msgs []domain.ConversationMessage
	for _, m := range s.messages {
		if m.chatID == chatID {
			msgs = append(msgs, domain.ConversationMessage{
				Role:      m.role,
				Content:   m.content,
				CreatedAt: time.Now(),
			})
		}
	}
	if len(msgs) > limit {
		msgs = msgs[len(msgs)-limit:]
	}
	return msgs, nil
}

type stubPrices struct {
	price     *domain.PriceSnapshot
	allPrices []*domain.PriceSnapshot
	err       error
}

func (s *stubPrices) GetCurrentPrice(ctx context.Context, symbol string) (*domain.PriceSnapshot, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.price != nil {
		return s.price, nil
	}
	return &domain.PriceSnapshot{Symbol: symbol}, nil
}

func (s *stubPrices) GetCurrentPrices(ctx context.Context) ([]*domain.PriceSnapshot, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.allPrices, nil
}

type stubSignals struct {
	signals []domain.Signal
	err     error
}

func (s *stubSignals) ListSignals(ctx context.Context, filter domain.SignalFilter) ([]domain.Signal, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.signals, nil
}
