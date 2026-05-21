package blastradius

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultDepth = 3
	maxDepth     = 10
)

// Service computes blast-radius (inbound dependency fan-in) for a repository file.
type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) Analyze(ctx context.Context, repositoryID int64, filePath, symbol string, depth int) (*Result, error) {
	filePath = normPath(filePath)
	if filePath == "" {
		return nil, fmt.Errorf("file_path is required")
	}
	if depth <= 0 {
		depth = defaultDepth
	}
	if depth > maxDepth {
		depth = maxDepth
	}

	files, deps, err := loadRepoGraph(ctx, s.pool, repositoryID)
	if err != nil {
		return nil, err
	}
	pathToID := make(map[string]int64, len(files))
	idToPath := make(map[int64]string, len(files))
	for _, f := range files {
		p := normPath(f.path)
		pathToID[p] = f.id
		idToPath[f.id] = p
	}
	targetID, ok := pathToID[filePath]
	if !ok {
		return nil, fmt.Errorf("file not found: %s", filePath)
	}

	inbound := inboundAdjacency(deps)
	depthOf := bfsDependents(targetID, inbound, depth)

	ids := make([]int64, 0, len(depthOf))
	for id := range depthOf {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		if depthOf[ids[i]] != depthOf[ids[j]] {
			return depthOf[ids[i]] < depthOf[ids[j]]
		}
		return idToPath[ids[i]] < idToPath[ids[j]]
	})

	enrich, err := s.loadEnrichment(ctx, repositoryID, append(ids, targetID))
	if err != nil {
		return nil, err
	}

	targetEn := enrich.byID[targetID]
	targetOwner := formatOwner(targetEn.ownerLogin)
	busScore := targetEn.dominantShare
	if busScore <= 0 && targetEn.busFactor > 0 {
		busScore = 1.0 / float64(targetEn.busFactor)
	}

	direct := 0
	for _, d := range depthOf {
		if d == 1 {
			direct++
		}
	}

	affected := make([]AffectedFile, 0, len(ids))
	teamsSet := make(map[string]struct{})
	var noTests, multiTeam int
	maxBusRisk := 0.0

	for _, id := range ids {
		p := idToPath[id]
		dep := depthOf[id]
		e := enrich.byID[id]
		owner := formatOwner(e.ownerLogin)
		rel := "transitive"
		if dep == 1 {
			rel = "direct_import"
		}
		if !e.hasTests {
			noTests++
		}
		team := pseudoTeam(p)
		if team != "" {
			teamsSet[team] = struct{}{}
		}
		busRisk := busFactorRisk(e.busFactor, e.hasBusFactorRisk)
		if busRisk > maxBusRisk {
			maxBusRisk = busRisk
		}
		affected = append(affected, AffectedFile{
			FilePath:     p,
			Depth:        dep,
			Relationship: rel,
			Owner:        owner,
			HasTests:     e.hasTests,
			RiskLevel:    e.riskLevel,
		})
	}

	transitive := len(depthOf) - direct
	total := len(depthOf)
	teams := sortedKeys(teamsSet)
	if len(teams) > 1 {
		multiTeam = len(teams)
	}

	riskScore := computeRiskScore(total, noTests, multiTeam, maxBusRisk)
	warnings := buildWarnings(noTests, multiTeam, enrich.signalsByPath)

	return &Result{
		Target: TargetInfo{
			FilePath:       filePath,
			Symbol:         strings.TrimSpace(symbol),
			Owner:          targetOwner,
			BusFactorScore: clamp01(busScore),
		},
		BlastRadius: BlastSummary{
			DirectDependents:     direct,
			TransitiveDependents: transitive,
			TotalFilesAffected:   total,
			RiskScore:            riskScore,
			TeamsAffected:        teams,
		},
		Files:    affected,
		Warnings: warnings,
	}, nil
}

type fileEnrichment struct {
	ownerLogin         string
	dominantShare      float64
	busFactor          int
	hasBusFactorRisk   bool
	riskLevel          string
	hasTests           bool
}

type enrichmentBundle struct {
	byID            map[int64]fileEnrichment
	signalsByPath   map[string][]string
}

func (s *Service) loadEnrichment(ctx context.Context, repositoryID int64, fileIDs []int64) (*enrichmentBundle, error) {
	out := &enrichmentBundle{
		byID:          make(map[int64]fileEnrichment, len(fileIDs)),
		signalsByPath: make(map[string][]string),
	}
	if len(fileIDs) == 0 {
		return out, nil
	}

	rows, err := s.pool.Query(ctx, `
		SELECT f.id, f.relative_path,
		       COALESCE(c.login, ''),
		       COALESCE(fm.dominant_owner_share, 0),
		       COALESCE(fm.bus_factor, 0),
		       COALESCE(fm.has_bus_factor_risk, false),
		       COALESCE(fm.risk_level, 'low')
		FROM files f
		LEFT JOIN file_metrics fm ON fm.file_id = f.id AND fm.repository_id = f.repository_id
		LEFT JOIN contributors c ON c.id = fm.dominant_owner_id
		WHERE f.repository_id = $1 AND f.id = ANY($2)
	`, repositoryID, fileIDs)
	if err != nil {
		return nil, fmt.Errorf("load file metrics: %w", err)
	}
	defer rows.Close()

	pathByID := make(map[int64]string)
	for rows.Next() {
		var id int64
		var path, login, risk string
		var share float64
		var bus int
		var busRisk bool
		if err := rows.Scan(&id, &path, &login, &share, &bus, &busRisk, &risk); err != nil {
			return nil, fmt.Errorf("scan enrichment: %w", err)
		}
		path = normPath(path)
		pathByID[id] = path
		out.byID[id] = fileEnrichment{
			ownerLogin:       login,
			dominantShare:    share,
			busFactor:        bus,
			hasBusFactorRisk: busRisk,
			riskLevel:        risk,
			hasTests:         isTestPath(path),
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Mark files that have a corresponding test file in the repo (same stem).
	testRows, err := s.pool.Query(ctx, `
		SELECT relative_path FROM files WHERE repository_id = $1
	`, repositoryID)
	if err == nil {
		defer testRows.Close()
		testStems := make(map[string]struct{})
		for testRows.Next() {
			var p string
			if err := testRows.Scan(&p); err != nil {
				break
			}
			if isTestPath(p) {
				testStems[testStemKey(p)] = struct{}{}
			}
		}
		for id, p := range pathByID {
			if out.byID[id].hasTests {
				continue
			}
			if _, ok := testStems[testStemKey(p)]; ok {
				e := out.byID[id]
				e.hasTests = true
				out.byID[id] = e
			}
		}
	}

	sigRows, err := s.pool.Query(ctx, `
		SELECT f.relative_path, s.signal_type
		FROM architecture_signals s
		JOIN files f ON f.id = s.file_id
		WHERE s.repository_id = $1 AND s.confidence >= 0.7
		  AND s.signal_type IN ('coupling_warning', 'known_fragility')
	`, repositoryID)
	if err == nil {
		defer sigRows.Close()
		for sigRows.Next() {
			var path, sigType string
			if err := sigRows.Scan(&path, &sigType); err != nil {
				break
			}
			path = normPath(path)
			out.signalsByPath[path] = append(out.signalsByPath[path], sigType)
		}
	}

	return out, nil
}

func loadRepoGraph(ctx context.Context, pool *pgxpool.Pool, repositoryID int64) ([]struct {
	id   int64
	path string
}, []struct{ from, to int64 }, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, relative_path FROM files WHERE repository_id = $1
	`, repositoryID)
	if err != nil {
		return nil, nil, fmt.Errorf("list files: %w", err)
	}
	defer rows.Close()
	var files []struct {
		id   int64
		path string
	}
	for rows.Next() {
		var fr struct {
			id   int64
			path string
		}
		if err := rows.Scan(&fr.id, &fr.path); err != nil {
			return nil, nil, fmt.Errorf("scan file: %w", err)
		}
		files = append(files, fr)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	drows, err := pool.Query(ctx, `
		SELECT from_file_id, to_file_id FROM file_dependencies WHERE repository_id = $1
	`, repositoryID)
	if err != nil {
		return nil, nil, fmt.Errorf("list deps: %w", err)
	}
	defer drows.Close()
	var deps []struct{ from, to int64 }
	for drows.Next() {
		var from, to int64
		if err := drows.Scan(&from, &to); err != nil {
			return nil, nil, fmt.Errorf("scan dep: %w", err)
		}
		deps = append(deps, struct{ from, to int64 }{from, to})
	}
	if err := drows.Err(); err != nil {
		return nil, nil, err
	}
	return files, deps, nil
}

func computeRiskScore(total, noTests, multiTeamCount int, maxBusRisk float64) float64 {
	if total == 0 {
		return 0
	}
	tf := float64(total)
	score := (float64(noTests)/tf)*0.4 + (float64(multiTeamCount)/tf)*0.3 + maxBusRisk*0.3
	return clamp01(score)
}

func busFactorRisk(busFactor int, flagged bool) float64 {
	if flagged || busFactor <= 1 {
		return 1
	}
	if busFactor == 2 {
		return 0.5
	}
	return 0.2
}

func buildWarnings(noTests, multiTeam int, signals map[string][]string) []string {
	var w []string
	if noTests > 0 {
		w = append(w, fmt.Sprintf("%d affected file(s) have no test coverage", noTests))
	}
	if multiTeam > 1 {
		w = append(w, fmt.Sprintf("%d team(s) affected by this change", multiTeam))
	}
	for path, sigs := range signals {
		for _, st := range sigs {
			if st == "coupling_warning" {
				w = append(w, fmt.Sprintf("coupling_warning signal on %s", path))
				break
			}
		}
	}
	sort.Strings(w)
	return w
}

func formatOwner(login string) string {
	login = strings.TrimSpace(login)
	if login == "" {
		return ""
	}
	if strings.HasPrefix(login, "@") {
		return login
	}
	return "@" + login
}

func pseudoTeam(path string) string {
	path = normPath(path)
	parts := strings.Split(path, "/")
	start := 0
	if len(parts) > 0 && (parts[0] == "src" || parts[0] == "apps" || parts[0] == "packages" || parts[0] == "lib") {
		start = 1
	}
	if start < len(parts) {
		return parts[start]
	}
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func isTestPath(path string) bool {
	p := strings.ToLower(normPath(path))
	if strings.Contains(p, "__tests__") || strings.Contains(p, "/test/") || strings.Contains(p, "/tests/") {
		return true
	}
	base := filepath.Base(p)
	return strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") ||
		strings.HasSuffix(base, "_test.go") || strings.HasSuffix(base, "_test.ts") ||
		strings.HasSuffix(base, "_test.tsx")
}

func testStemKey(path string) string {
	p := normPath(path)
	base := filepath.Base(p)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	for _, suf := range []string{".test", ".spec", "_test"} {
		stem = strings.TrimSuffix(stem, suf)
	}
	dir := filepath.Dir(p)
	return dir + "/" + stem
}

func normPath(p string) string {
	p = filepath.ToSlash(strings.TrimSpace(p))
	p = strings.TrimPrefix(p, "./")
	return p
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		if k != "" {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}
