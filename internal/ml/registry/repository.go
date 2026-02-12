package registry

import (
	"context"
	"errors"
	"time"

	"bug-free-umbrella/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.opentelemetry.io/otel/trace"
)

type row interface {
	Scan(dest ...any) error
}

type tx interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type pool interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Repository struct {
	pool   pool
	tracer trace.Tracer
}

func NewRepository(pool pool, tracer trace.Tracer) *Repository {
	return &Repository{pool: pool, tracer: tracer}
}

func (r *Repository) NextVersion(ctx context.Context, modelKey string) (int, error) {
	_, span := r.tracer.Start(ctx, "ml-model-registry.next-version")
	defer span.End()

	var version int
	err := r.pool.QueryRow(ctx, `SELECT COALESCE(MAX(version), 0) + 1 FROM ml_model_versions WHERE model_key = $1`, modelKey).Scan(&version)
	return version, err
}

func (r *Repository) InsertModelVersion(ctx context.Context, model domain.MLModelVersion) (*domain.MLModelVersion, error) {
	_, span := r.tracer.Start(ctx, "ml-model-registry.insert")
	defer span.End()

	if model.ModelKey == "" || model.Version <= 0 {
		return nil, errors.New("invalid model version payload")
	}
	var out domain.MLModelVersion
	err := r.pool.QueryRow(ctx, `
INSERT INTO ml_model_versions (
    model_key, version, feature_spec_version,
    trained_from, trained_to, trained_at,
    hyperparams_json, metrics_json,
    artifact_format, artifact_blob,
    is_active, activated_at
) VALUES (
    $1, $2, $3,
    $4, $5, COALESCE($6, NOW()),
    $7, $8,
    $9, $10,
    $11, $12
)
RETURNING id, model_key, version, feature_spec_version,
          trained_from, trained_to, trained_at,
          hyperparams_json, metrics_json,
          artifact_format, artifact_blob,
          is_active, activated_at, created_at`,
		model.ModelKey,
		model.Version,
		model.FeatureSpecVersion,
		model.TrainedFrom.UTC(),
		model.TrainedTo.UTC(),
		nullIfZeroTime(model.TrainedAt),
		fallbackJSON(model.HyperparamsJSON),
		fallbackJSON(model.MetricsJSON),
		model.ArtifactFormat,
		model.ArtifactBlob,
		model.IsActive,
		nullTime(model.ActivatedAt),
	).Scan(
		&out.ID,
		&out.ModelKey,
		&out.Version,
		&out.FeatureSpecVersion,
		&out.TrainedFrom,
		&out.TrainedTo,
		&out.TrainedAt,
		&out.HyperparamsJSON,
		&out.MetricsJSON,
		&out.ArtifactFormat,
		&out.ArtifactBlob,
		&out.IsActive,
		&out.ActivatedAt,
		&out.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	normalizeModelTimes(&out)
	return &out, nil
}

func (r *Repository) GetActiveModel(ctx context.Context, modelKey string) (*domain.MLModelVersion, error) {
	_, span := r.tracer.Start(ctx, "ml-model-registry.get-active")
	defer span.End()

	return r.getOne(ctx, `
SELECT id, model_key, version, feature_spec_version,
       trained_from, trained_to, trained_at,
       hyperparams_json, metrics_json,
       artifact_format, artifact_blob,
       is_active, activated_at, created_at
FROM ml_model_versions
WHERE model_key = $1 AND is_active = TRUE
ORDER BY version DESC
LIMIT 1`, modelKey)
}

func (r *Repository) GetLatestModel(ctx context.Context, modelKey string) (*domain.MLModelVersion, error) {
	_, span := r.tracer.Start(ctx, "ml-model-registry.get-latest")
	defer span.End()

	return r.getOne(ctx, `
SELECT id, model_key, version, feature_spec_version,
       trained_from, trained_to, trained_at,
       hyperparams_json, metrics_json,
       artifact_format, artifact_blob,
       is_active, activated_at, created_at
FROM ml_model_versions
WHERE model_key = $1
ORDER BY version DESC
LIMIT 1`, modelKey)
}

func (r *Repository) ActivateModel(ctx context.Context, modelKey string, version int) error {
	_, span := r.tracer.Start(ctx, "ml-model-registry.activate")
	defer span.End()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `UPDATE ml_model_versions SET is_active = FALSE, activated_at = NULL WHERE model_key = $1`, modelKey); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE ml_model_versions SET is_active = TRUE, activated_at = NOW() WHERE model_key = $1 AND version = $2`, modelKey, version)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return tx.Commit(ctx)
}

func (r *Repository) getOne(ctx context.Context, query string, arg any) (*domain.MLModelVersion, error) {
	var out domain.MLModelVersion
	err := r.pool.QueryRow(ctx, query, arg).Scan(
		&out.ID,
		&out.ModelKey,
		&out.Version,
		&out.FeatureSpecVersion,
		&out.TrainedFrom,
		&out.TrainedTo,
		&out.TrainedAt,
		&out.HyperparamsJSON,
		&out.MetricsJSON,
		&out.ArtifactFormat,
		&out.ArtifactBlob,
		&out.IsActive,
		&out.ActivatedAt,
		&out.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	normalizeModelTimes(&out)
	return &out, nil
}

func normalizeModelTimes(model *domain.MLModelVersion) {
	model.TrainedFrom = model.TrainedFrom.UTC()
	model.TrainedTo = model.TrainedTo.UTC()
	model.TrainedAt = model.TrainedAt.UTC()
	model.CreatedAt = model.CreatedAt.UTC()
	if model.ActivatedAt != nil {
		t := model.ActivatedAt.UTC()
		model.ActivatedAt = &t
	}
}

func fallbackJSON(v string) string {
	if v == "" {
		return "{}"
	}
	return v
}

func nullIfZeroTime(v time.Time) any {
	if v.IsZero() {
		return nil
	}
	return v.UTC()
}

func nullTime(v *time.Time) any {
	if v == nil || v.IsZero() {
		return nil
	}
	t := v.UTC()
	return t
}
