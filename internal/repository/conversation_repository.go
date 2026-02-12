package repository

import (
	"context"
	"time"

	"bug-free-umbrella/internal/domain"

	"go.opentelemetry.io/otel/trace"
)

type ConversationRepository struct {
	pool   PgxPool
	tracer trace.Tracer
}

func NewConversationRepository(pool PgxPool, tracer trace.Tracer) *ConversationRepository {
	return &ConversationRepository{pool: pool, tracer: tracer}
}

func (r *ConversationRepository) AppendMessage(ctx context.Context, chatID int64, role, content string) error {
	_, span := r.tracer.Start(ctx, "conversation-repo.append-message")
	defer span.End()

	_, err := r.pool.Exec(ctx,
		`INSERT INTO conversation_messages (chat_id, role, content) VALUES ($1, $2, $3)`,
		chatID, role, content,
	)
	return err
}

func (r *ConversationRepository) RecentMessages(ctx context.Context, chatID int64, limit int) ([]domain.ConversationMessage, error) {
	_, span := r.tracer.Start(ctx, "conversation-repo.recent-messages")
	defer span.End()

	rows, err := r.pool.Query(ctx,
		`SELECT role, content, created_at
		 FROM conversation_messages
		 WHERE chat_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2`,
		chatID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []domain.ConversationMessage
	for rows.Next() {
		var m domain.ConversationMessage
		var ts time.Time
		if err := rows.Scan(&m.Role, &m.Content, &ts); err != nil {
			return nil, err
		}
		m.CreatedAt = ts.UTC()
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Reverse: DB returns newest-first, we need oldest-first for prompt building
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}
