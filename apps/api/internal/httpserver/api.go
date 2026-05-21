package httpserver

import (
	"codeatlas/apps/api/internal/ai"
	"codeatlas/apps/api/internal/blastradius"
	"codeatlas/apps/api/internal/config"
	"codeatlas/apps/api/internal/driftdetector"
	"codeatlas/apps/api/internal/ingestprogress"
	"codeatlas/apps/api/internal/jobqueue"
	"codeatlas/apps/api/internal/livingdocs"
	"codeatlas/apps/api/internal/mcp"
	"codeatlas/apps/api/internal/onboarding"
	"codeatlas/apps/api/internal/repoingest"
	"codeatlas/apps/api/internal/socio"
	"codeatlas/apps/api/internal/teams"
	"codeatlas/apps/api/internal/vcsauth"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultBlastDepth   = 3
	maxClusterFiles     = 500
	maxClusterEdges     = 2000
	maxBlastResultFiles = 200
)

// API holds HTTP handler dependencies (domain controllers use this struct).
type API struct {
	cfg          config.Config
	pool         *pgxpool.Pool
	aiService    *ai.Service
	ingest       *repoingest.Service
	ingestQueue  jobqueue.JobQueue
	progressBus  ingestprogress.EventBus
	socioQuery   *socio.QueryService
	blastSvc     *blastradius.Service
	driftEngine  *driftdetector.Engine
	driftStore   *driftdetector.Store
	mcpServer    *mcp.Server
	teamsSvc     *teams.Service
	onboarding   *onboarding.Service
	livingDocs   *livingdocs.Service
	vcs          *vcsauth.Service
}
