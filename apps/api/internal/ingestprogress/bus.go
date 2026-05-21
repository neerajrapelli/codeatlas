package ingestprogress

import "context"

// EventBus publishes ingestion progress for SSE and cross-instance fan-out.
type EventBus interface {
	Publish(repositoryID int64, ev StreamEvent)
	Latest(repositoryID int64) (StreamEvent, bool)
	Subscribe(ctx context.Context, repositoryID int64) (<-chan StreamEvent, func(), error)
}
