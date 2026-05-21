package socio

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ComputeFileMetrics aggregates commit_files and pr_files into file_metrics and ownership rows.
func ComputeFileMetrics(ctx context.Context, pool *pgxpool.Pool, repositoryID int64) ([]FileMetrics, []OwnerRow, error) {
	since := time.Now().AddDate(0, 0, -90)

	type fileAgg struct {
		fileID      int64
		commits     int
		additions   int
		deletions   int
		lastActive  time.Time
		authorCounts map[uuid.UUID]int
	}

	agg := make(map[int64]*fileAgg)

	rows, err := pool.Query(ctx, `
		SELECT cf.file_id, c.author_contributor_id, cf.additions, cf.deletions, c.committed_at
		FROM commit_files cf
		JOIN commits c ON c.id = cf.commit_id
		WHERE c.repository_id=$1 AND c.committed_at >= $2
	`, repositoryID, since)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var fileID int64
		var authorID *uuid.UUID
		var add, del int
		var at time.Time
		if err := rows.Scan(&fileID, &authorID, &add, &del, &at); err != nil {
			rows.Close()
			return nil, nil, err
		}
		a := agg[fileID]
		if a == nil {
			a = &fileAgg{fileID: fileID, authorCounts: make(map[uuid.UUID]int)}
			agg[fileID] = a
		}
		a.commits++
		a.additions += add
		a.deletions += del
		if at.After(a.lastActive) {
			a.lastActive = at
		}
		if authorID != nil {
			a.authorCounts[*authorID]++
		}
	}
	rows.Close()

	prRows, err := pool.Query(ctx, `
		SELECT pf.file_id, pr.author_contributor_id, pf.additions, pf.deletions, pr.created_at
		FROM pr_files pf
		JOIN pull_requests pr ON pr.id = pf.pull_request_id
		WHERE pr.repository_id=$1 AND pr.created_at >= $2
	`, repositoryID, since)
	if err != nil {
		return nil, nil, err
	}
	for prRows.Next() {
		var fileID int64
		var authorID *uuid.UUID
		var add, del int
		var at time.Time
		if err := prRows.Scan(&fileID, &authorID, &add, &del, &at); err != nil {
			prRows.Close()
			return nil, nil, err
		}
		a := agg[fileID]
		if a == nil {
			a = &fileAgg{fileID: fileID, authorCounts: make(map[uuid.UUID]int)}
			agg[fileID] = a
		}
		a.commits++
		a.additions += add
		a.deletions += del
		if at.After(a.lastActive) {
			a.lastActive = at
		}
		if authorID != nil {
			a.authorCounts[*authorID]++
		}
	}
	prRows.Close()

	if len(agg) == 0 {
		return nil, nil, nil
	}

	var churnValues []float64
	for _, a := range agg {
		churnValues = append(churnValues, float64(a.additions+a.deletions))
	}
	sort.Float64s(churnValues)
	p90 := percentile(churnValues, 0.9)
	if p90 <= 0 {
		p90 = 1
	}

	metrics := make([]FileMetrics, 0, len(agg))
	var ownerRows []OwnerRow

	for _, a := range agg {
		churn := float64(a.additions + a.deletions)
		authors := len(a.authorCounts)
		bus := authors
		if bus < 1 {
			bus = 1
		}

		var dominant uuid.UUID
		maxCount := 0
		totalAuthorCommits := 0
		for cid, cnt := range a.authorCounts {
			totalAuthorCommits += cnt
			if cnt > maxCount {
				maxCount = cnt
				dominant = cid
			}
		}
		share := 0.0
		if totalAuthorCommits > 0 {
			share = float64(maxCount) / float64(totalAuthorCommits)
		}

		hotspotScore := math.Min(1, churn/p90)
		isHotspot := hotspotScore >= 0.75
		hasBusRisk := bus <= 1 && a.commits >= 3

		risk := RiskLow
		switch {
		case isHotspot && hasBusRisk:
			risk = RiskCritical
		case isHotspot || hasBusRisk:
			risk = RiskHigh
		case hotspotScore >= 0.45:
			risk = RiskMedium
		}

		var domPtr *uuid.UUID
		if dominant != uuid.Nil {
			domPtr = &dominant
		}
		last := a.lastActive
		metrics = append(metrics, FileMetrics{
			FileID:             a.fileID,
			ChurnScore:         churn,
			CommitCount90d:     a.commits,
			UniqueAuthors90d:   authors,
			BusFactor:          bus,
			HotspotScore:       hotspotScore,
			RiskLevel:          risk,
			IsHotspot:          isHotspot,
			HasBusFactorRisk:   hasBusRisk,
			DominantOwnerID:    domPtr,
			DominantOwnerShare: share,
			LastActivityAt:     &last,
		})

		for cid, cnt := range a.authorCounts {
			sh := 0.0
			if totalAuthorCommits > 0 {
				sh = float64(cnt) / float64(totalAuthorCommits)
			}
			ownerRows = append(ownerRows, OwnerRow{
				FileID: a.fileID, ContributorID: cid, CommitCount: cnt, Share: sh,
			})
		}
	}
	return metrics, ownerRows, nil
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(p*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
