package archintel

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) UpsertDecision(ctx context.Context, repoID int64, in DecisionRecord, sourceKind, sourceRef string) (string, error) {
	tradeoffs, _ := json.Marshal(in.Tradeoffs)
	affectedModules, _ := json.Marshal(in.AffectedModules)
	affectedFiles, _ := json.Marshal(in.AffectedFiles)
	participants, _ := json.Marshal(in.Participants)
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO architecture_decisions(
		  repository_id, tenant_id, title, summary, status, confidence, tradeoffs,
		  affected_modules, affected_files, participants, source_kind, source_ref, updated_at
		)
		VALUES(
		  $1,
		  COALESCE((SELECT tenant_id FROM repositories WHERE id=$1),'default'),
		  $2,$3,$4,$5,$6::jsonb,$7::jsonb,$8::jsonb,$9::jsonb,$10,$11,NOW()
		)
		ON CONFLICT DO NOTHING
		RETURNING id::text
	`,
		repoID,
		in.Title,
		in.Summary,
		string(in.Status),
		in.Confidence,
		string(tradeoffs),
		string(affectedModules),
		string(affectedFiles),
		string(participants),
		sourceKind,
		sourceRef,
	).Scan(&id)
	if err != nil {
		// fallback to latest similar record for idempotency without extra unique constraints
		err = s.pool.QueryRow(ctx, `
			SELECT id::text
			FROM architecture_decisions
			WHERE repository_id=$1 AND source_kind=$2 AND source_ref=$3 AND summary=$4
			ORDER BY updated_at DESC
			LIMIT 1
		`, repoID, sourceKind, sourceRef, in.Summary).Scan(&id)
		if err != nil {
			return "", fmt.Errorf("upsert decision: %w", err)
		}
	}
	return id, nil
}

func (s *Store) InsertDecisionEvent(
	ctx context.Context,
	repoID int64,
	decisionID, eventType, summary, actor string,
	eventAt time.Time,
	meta map[string]any,
) error {
	raw, _ := json.Marshal(meta)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO architecture_decision_events(
		  decision_id, repository_id, tenant_id, event_type, summary, actor_login, event_at, metadata
		)
		VALUES(
		  $1::uuid, $2, COALESCE((SELECT tenant_id FROM repositories WHERE id=$2),'default'),
		  $3, $4, $5, $6, $7::jsonb
		)
	`, decisionID, repoID, eventType, summary, actor, eventAt, string(raw))
	return err
}
