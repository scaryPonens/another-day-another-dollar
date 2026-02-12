package advisor

import (
	"context"
	"fmt"
	"log"

	"bug-free-umbrella/internal/domain"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// LLMClient abstracts the OpenAI chat completions API for testability.
type LLMClient interface {
	CreateChatCompletion(ctx context.Context, params openai.ChatCompletionNewParams) (*openai.ChatCompletion, error)
}

// PriceQuerier provides current price data for the advisor's context.
type PriceQuerier interface {
	GetCurrentPrices(ctx context.Context) ([]*domain.PriceSnapshot, error)
	GetCurrentPrice(ctx context.Context, symbol string) (*domain.PriceSnapshot, error)
}

// SignalQuerier provides signal data for the advisor's context.
type SignalQuerier interface {
	ListSignals(ctx context.Context, filter domain.SignalFilter) ([]domain.Signal, error)
}

// ConversationStore persists and retrieves conversation messages.
type ConversationStore interface {
	AppendMessage(ctx context.Context, chatID int64, role, content string) error
	RecentMessages(ctx context.Context, chatID int64, limit int) ([]domain.ConversationMessage, error)
}

type AdvisorService struct {
	tracer     trace.Tracer
	llm        LLMClient
	prices     PriceQuerier
	signals    SignalQuerier
	convStore  ConversationStore
	model      string
	maxHistory int
}

func NewAdvisorService(
	tracer trace.Tracer,
	llm LLMClient,
	prices PriceQuerier,
	signals SignalQuerier,
	convStore ConversationStore,
	model string,
	maxHistory int,
) *AdvisorService {
	if maxHistory <= 0 {
		maxHistory = 20
	}
	return &AdvisorService{
		tracer:     tracer,
		llm:        llm,
		prices:     prices,
		signals:    signals,
		convStore:  convStore,
		model:      model,
		maxHistory: maxHistory,
	}
}

func (s *AdvisorService) Ask(ctx context.Context, chatID int64, userMessage string) (string, error) {
	ctx, span := s.tracer.Start(ctx, "advisor.ask")
	defer span.End()
	span.SetAttributes(attribute.Int64("chat_id", chatID))

	// 1. Persist the user message
	if err := s.convStore.AppendMessage(ctx, chatID, "user", userMessage); err != nil {
		log.Printf("failed to store user message: %v", err)
	}

	// 2. Extract mentioned symbols for targeted context
	mentionedSymbols := ExtractSymbols(userMessage)

	// 3. Gather market context
	marketContext, err := s.gatherContext(ctx, mentionedSymbols)
	if err != nil {
		log.Printf("failed to gather market context: %v", err)
		marketContext = "Market data temporarily unavailable."
	}

	// 4. Build system prompt with live data
	systemPrompt := BuildSystemPrompt(marketContext)

	// 5. Load conversation history
	history, err := s.convStore.RecentMessages(ctx, chatID, s.maxHistory)
	if err != nil {
		log.Printf("failed to load conversation history: %v", err)
		history = nil
	}

	// 6. Construct messages array
	messages := s.buildMessages(systemPrompt, history)

	// 7. Call LLM
	reply, err := s.callLLM(ctx, messages)
	if err != nil {
		span.RecordError(err)
		return "", fmt.Errorf("advisor unavailable: %w", err)
	}

	// 8. Persist the assistant reply
	if err := s.convStore.AppendMessage(ctx, chatID, "assistant", reply); err != nil {
		log.Printf("failed to store assistant reply: %v", err)
	}

	return reply, nil
}

func (s *AdvisorService) gatherContext(ctx context.Context, symbols []string) (string, error) {
	ctx, span := s.tracer.Start(ctx, "advisor.gather-context")
	defer span.End()

	var prices []*domain.PriceSnapshot
	var signals []domain.Signal

	if len(symbols) > 0 {
		for _, sym := range symbols {
			p, err := s.prices.GetCurrentPrice(ctx, sym)
			if err == nil {
				prices = append(prices, p)
			}
			sigs, err := s.signals.ListSignals(ctx, domain.SignalFilter{Symbol: sym, Limit: 5})
			if err == nil {
				signals = append(signals, sigs...)
			}
		}
	} else {
		var err error
		prices, err = s.prices.GetCurrentPrices(ctx)
		if err != nil {
			return "", err
		}
		signals, _ = s.signals.ListSignals(ctx, domain.SignalFilter{Limit: 10})
	}

	return FormatMarketContext(prices, signals), nil
}

func (s *AdvisorService) buildMessages(
	systemPrompt string,
	history []domain.ConversationMessage,
) []openai.ChatCompletionMessageParamUnion {
	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(history)+1)

	// System prompt always first
	messages = append(messages, openai.SystemMessage(systemPrompt))

	// Conversation history (already limited by RecentMessages query)
	for _, msg := range history {
		switch msg.Role {
		case "user":
			messages = append(messages, openai.UserMessage(msg.Content))
		case "assistant":
			messages = append(messages, openai.AssistantMessage(msg.Content))
		}
	}

	return messages
}

func (s *AdvisorService) callLLM(
	ctx context.Context,
	messages []openai.ChatCompletionMessageParamUnion,
) (string, error) {
	ctx, span := s.tracer.Start(ctx, "advisor.llm-call")
	defer span.End()
	span.SetAttributes(
		attribute.String("llm.model", s.model),
		attribute.Int("llm.message_count", len(messages)),
	)

	completion, err := s.llm.CreateChatCompletion(ctx, openai.ChatCompletionNewParams{
		Model:    s.model,
		Messages: messages,
	})
	if err != nil {
		return "", err
	}
	if len(completion.Choices) == 0 {
		return "", fmt.Errorf("no choices in LLM response")
	}

	reply := completion.Choices[0].Message.Content
	span.SetAttributes(attribute.Int("llm.reply_length", len(reply)))
	return reply, nil
}

// openaiClient wraps the official SDK's chat completions service.
type openaiClient struct {
	client openai.Client
}

func NewOpenAIClient(apiKey string) LLMClient {
	client := openai.NewClient(option.WithAPIKey(apiKey))
	return &openaiClient{client: client}
}

func (c *openaiClient) CreateChatCompletion(
	ctx context.Context,
	params openai.ChatCompletionNewParams,
) (*openai.ChatCompletion, error) {
	return c.client.Chat.Completions.New(ctx, params)
}
