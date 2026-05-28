-- +goose Down
DROP INDEX IF EXISTS idx_discussion_docs_search_gin;
DROP INDEX IF EXISTS idx_discussion_docs_repo_tenant_time;
DROP INDEX IF EXISTS idx_discussion_docs_unique;
DROP TABLE IF EXISTS discussion_documents;

DROP INDEX IF EXISTS idx_arch_decision_events_decision;
DROP INDEX IF EXISTS idx_arch_decision_events_repo_tenant_event;
DROP TABLE IF EXISTS architecture_decision_events;

DROP INDEX IF EXISTS idx_arch_decision_links_module;
DROP INDEX IF EXISTS idx_arch_decision_links_repo_tenant_kind;
DROP INDEX IF EXISTS idx_arch_decision_links_decision;
DROP TABLE IF EXISTS architecture_decision_links;

DROP INDEX IF EXISTS idx_arch_decisions_status;
DROP INDEX IF EXISTS idx_arch_decisions_repo_tenant;
DROP TABLE IF EXISTS architecture_decisions;
