package driftdetector

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Engine validates architecture rules against the dependency graph.
type Engine struct {
	pool *pgxpool.Pool
}

func NewEngine(pool *pgxpool.Pool) *Engine {
	return &Engine{pool: pool}
}

func (e *Engine) ValidateAll(ctx context.Context, repositoryID int64) ([]Violation, error) {
	rules, err := e.listEnabledRules(ctx, repositoryID)
	if err != nil {
		return nil, err
	}
	graph, err := loadGraph(ctx, e.pool, repositoryID)
	if err != nil {
		return nil, err
	}
	violations := evaluateRules(rules, graph)
	if err := e.persistViolations(ctx, repositoryID, violations); err != nil {
		return nil, err
	}
	return violations, nil
}

func (e *Engine) ValidateFile(ctx context.Context, repositoryID int64, filePath string) ([]Violation, error) {
	rules, err := e.listEnabledRules(ctx, repositoryID)
	if err != nil {
		return nil, err
	}
	graph, err := loadGraph(ctx, e.pool, repositoryID)
	if err != nil {
		return nil, err
	}
	filePath = normPath(filePath)
	all := evaluateRules(rules, graph)
	out := make([]Violation, 0)
	for _, v := range all {
		if v.SourceFile == filePath || v.TargetFile == filePath {
			out = append(out, v)
		}
	}
	return out, nil
}

func (e *Engine) CheckChangedFiles(ctx context.Context, repositoryID int64, changed []string) ([]Violation, error) {
	rules, err := e.listEnabledRules(ctx, repositoryID)
	if err != nil {
		return nil, err
	}
	graph, err := loadGraph(ctx, e.pool, repositoryID)
	if err != nil {
		return nil, err
	}
	all := evaluateRules(rules, graph)
	changedSet := make(map[string]struct{}, len(changed))
	for _, p := range changed {
		changedSet[normPath(p)] = struct{}{}
	}
	out := make([]Violation, 0)
	for _, v := range all {
		if _, ok := changedSet[v.SourceFile]; ok {
			out = append(out, v)
			continue
		}
		if _, ok := changedSet[v.TargetFile]; ok {
			out = append(out, v)
		}
	}
	return out, nil
}

type graphData struct {
	pathByID map[int64]string
	deps     []struct{ fromPath, toPath string }
}

func loadGraph(ctx context.Context, pool *pgxpool.Pool, repositoryID int64) (*graphData, error) {
	rows, err := pool.Query(ctx, `SELECT id, relative_path FROM files WHERE repository_id = $1`, repositoryID)
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}
	defer rows.Close()
	pathByID := make(map[int64]string)
	for rows.Next() {
		var id int64
		var path string
		if err := rows.Scan(&id, &path); err != nil {
			return nil, err
		}
		pathByID[id] = normPath(path)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	drows, err := pool.Query(ctx, `
		SELECT from_file_id, to_file_id FROM file_dependencies WHERE repository_id = $1
	`, repositoryID)
	if err != nil {
		return nil, fmt.Errorf("list deps: %w", err)
	}
	defer drows.Close()
	var deps []struct{ fromPath, toPath string }
	for drows.Next() {
		var from, to int64
		if err := drows.Scan(&from, &to); err != nil {
			return nil, err
		}
		fp, ok1 := pathByID[from]
		tp, ok2 := pathByID[to]
		if ok1 && ok2 {
			deps = append(deps, struct{ fromPath, toPath string }{fp, tp})
		}
	}
	if err := drows.Err(); err != nil {
		return nil, err
	}
	return &graphData{pathByID: pathByID, deps: deps}, nil
}

func evaluateRules(rules []Rule, g *graphData) []Violation {
	var out []Violation
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		switch rule.RuleType {
		case "no_import", "layer_order":
			for _, d := range g.deps {
				if matchGlob(rule.SourcePattern, d.fromPath) && matchGlob(rule.TargetPattern, d.toPath) {
					out = append(out, Violation{
						RuleID:     rule.ID,
						RuleName:   rule.Name,
						SourceFile: d.fromPath,
						TargetFile: d.toPath,
						Severity:   rule.Severity,
						Message:    fmt.Sprintf("%s imports %s — violates %q", d.fromPath, d.toPath, rule.Name),
					})
				}
			}
		case "must_import":
			sources := matchedPaths(g, rule.SourcePattern)
			for _, src := range sources {
				has := false
				for _, d := range g.deps {
					if d.fromPath != src {
						continue
					}
					if matchGlob(rule.TargetPattern, d.toPath) {
						has = true
						break
					}
				}
				if !has {
					out = append(out, Violation{
						RuleID:     rule.ID,
						RuleName:   rule.Name,
						SourceFile: src,
						TargetFile: rule.TargetPattern,
						Severity:   rule.Severity,
						Message:    fmt.Sprintf("%s must import something matching %s — violates %q", src, rule.TargetPattern, rule.Name),
					})
				}
			}
		case "no_circular":
			out = append(out, findCycles(g, rule)...)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SourceFile != out[j].SourceFile {
			return out[i].SourceFile < out[j].SourceFile
		}
		return out[i].TargetFile < out[j].TargetFile
	})
	return out
}

func matchedPaths(g *graphData, pattern string) []string {
	var paths []string
	seen := make(map[string]struct{})
	for _, p := range g.pathByID {
		if matchGlob(pattern, p) {
			if _, ok := seen[p]; !ok {
				seen[p] = struct{}{}
				paths = append(paths, p)
			}
		}
	}
	sort.Strings(paths)
	return paths
}

func findCycles(g *graphData, rule Rule) []Violation {
	scope := matchedPaths(g, rule.SourcePattern)
	if len(scope) == 0 {
		scope = matchedPaths(g, "**")
	}
	inScope := make(map[string]struct{}, len(scope))
	for _, p := range scope {
		inScope[p] = struct{}{}
	}
	adj := make(map[string][]string)
	for _, d := range g.deps {
		if _, ok := inScope[d.fromPath]; !ok {
			continue
		}
		if _, ok := inScope[d.toPath]; !ok {
			continue
		}
		adj[d.fromPath] = append(adj[d.fromPath], d.toPath)
	}
	state := make(map[string]int) // 0=unvisited 1=visiting 2=done
	var violations []Violation
	var dfs func(string)
	dfs = func(node string) {
		state[node] = 1
		for _, next := range adj[node] {
			switch state[next] {
			case 1:
				violations = append(violations, Violation{
					RuleID:     rule.ID,
					RuleName:   rule.Name,
					SourceFile: node,
					TargetFile: next,
					Severity:   rule.Severity,
					Message:    fmt.Sprintf("circular dependency: %s → %s — violates %q", node, next, rule.Name),
				})
			case 0:
				dfs(next)
			}
		}
		state[node] = 2
	}
	for _, p := range scope {
		if state[p] == 0 {
			dfs(p)
		}
	}
	return violations
}

func (e *Engine) listEnabledRules(ctx context.Context, repositoryID int64) ([]Rule, error) {
	rows, err := e.pool.Query(ctx, `
		SELECT id, repository_id, name, COALESCE(description,''), rule_type,
		       source_pattern, target_pattern, severity, enabled, created_at
		FROM architecture_rules
		WHERE repository_id = $1 AND enabled = true
		ORDER BY created_at
	`, repositoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rules []Rule
	for rows.Next() {
		var r Rule
		if err := rows.Scan(
			&r.ID, &r.RepositoryID, &r.Name, &r.Description, &r.RuleType,
			&r.SourcePattern, &r.TargetPattern, &r.Severity, &r.Enabled, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

func (e *Engine) persistViolations(ctx context.Context, repositoryID int64, violations []Violation) error {
	tx, err := e.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE rule_violations SET is_active = false, resolved_at = now()
		WHERE repository_id = $1 AND is_active = true
	`, repositoryID); err != nil {
		return err
	}
	for _, v := range violations {
		if _, err := tx.Exec(ctx, `
			INSERT INTO rule_violations (rule_id, repository_id, source_file, target_file, is_active)
			VALUES ($1, $2, $3, $4, true)
		`, v.RuleID, repositoryID, v.SourceFile, v.TargetFile); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// Store provides CRUD for architecture rules.
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) CreateRule(ctx context.Context, repositoryID int64, req CreateRuleRequest) (Rule, error) {
	severity := req.Severity
	if severity == "" {
		severity = "warning"
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	var r Rule
	err := s.pool.QueryRow(ctx, `
		INSERT INTO architecture_rules (repository_id, name, description, rule_type, source_pattern, target_pattern, severity, enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, repository_id, name, COALESCE(description,''), rule_type, source_pattern, target_pattern, severity, enabled, created_at
	`, repositoryID, req.Name, req.Description, req.RuleType, req.SourcePattern, req.TargetPattern, severity, enabled).Scan(
		&r.ID, &r.RepositoryID, &r.Name, &r.Description, &r.RuleType,
		&r.SourcePattern, &r.TargetPattern, &r.Severity, &r.Enabled, &r.CreatedAt,
	)
	return r, err
}

func (s *Store) ListRules(ctx context.Context, repositoryID int64) ([]Rule, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, repository_id, name, COALESCE(description,''), rule_type,
		       source_pattern, target_pattern, severity, enabled, created_at
		FROM architecture_rules WHERE repository_id = $1 ORDER BY created_at
	`, repositoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rules []Rule
	for rows.Next() {
		var r Rule
		if err := rows.Scan(
			&r.ID, &r.RepositoryID, &r.Name, &r.Description, &r.RuleType,
			&r.SourcePattern, &r.TargetPattern, &r.Severity, &r.Enabled, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

func (s *Store) DeleteRule(ctx context.Context, repositoryID int64, ruleID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM architecture_rules WHERE id = $1 AND repository_id = $2
	`, ruleID, repositoryID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("rule not found")
	}
	return nil
}

func (s *Store) ListActiveViolations(ctx context.Context, repositoryID int64) ([]Violation, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT v.id, v.rule_id, r.name, v.source_file, v.target_file, r.severity, v.detected_at
		FROM rule_violations v
		JOIN architecture_rules r ON r.id = v.rule_id
		WHERE v.repository_id = $1 AND v.is_active = true
		ORDER BY r.severity DESC, v.detected_at DESC
	`, repositoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Violation
	for rows.Next() {
		var v Violation
		if err := rows.Scan(&v.ID, &v.RuleID, &v.RuleName, &v.SourceFile, &v.TargetFile, &v.Severity, &v.DetectedAt); err != nil {
			return nil, err
		}
		v.Message = fmt.Sprintf("%s imports %s — violates %q", v.SourceFile, v.TargetFile, v.RuleName)
		out = append(out, v)
	}
	return out, rows.Err()
}
