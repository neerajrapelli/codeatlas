import { useEffect, useState } from 'react';

import { getApiBase } from '../apiBase';

export interface FileOverlay {
  fileId: string;
  isHotspot: boolean;
  hasBusFactorRisk: boolean;
  riskLevel?: string;
  architectureSignalCount?: number;
  dominantOwnerLogin?: string;
}

export interface IngestionStatus {
  repositoryId: number;
  codeIndex: { status: string; stage: string; progressPercent: number; filesIndexed: number };
  socioTechnical: {
    phase: string;
    status: string;
    completionPercent: number;
    staleness: string;
    errorDetails?: string;
    steps?: Array<{ step: string; status: string; itemsProcessed: number }>;
  };
  graphCompleteness: {
    codeGraphReady: boolean;
    socioHistoryReady: boolean;
    partialDataWarning: boolean;
  };
}

interface OwnershipRow {
  fileId: number;
  path: string;
  contributorCount: number;
  busFactor: number;
  riskLevel: string;
  dominantOwnerShare: number;
  dominantOwner?: { login: string };
}

interface HotspotRow {
  fileId: number;
  path: string;
  hotspotScore: number;
  churnScore: number;
  riskLevel: string;
  busFactor: number;
  commitCount90d: number;
}

interface SocioPanelsProps {
  repositoryId: number;
  selectedFileId: string | null;
  onIngestionStatus?: (status: IngestionStatus | null) => void;
}

export function SocioPanels({ repositoryId, selectedFileId, onIngestionStatus }: SocioPanelsProps) {
  const [ownership, setOwnership] = useState<OwnershipRow[]>([]);
  const [hotspots, setHotspots] = useState<HotspotRow[]>([]);
  const [ingestion, setIngestion] = useState<IngestionStatus | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);

  useEffect(() => {
    const base = getApiBase();
    const ac = new AbortController();
    const run = async () => {
      try {
        const statusRes = await fetch(`${base}/repositories/${String(repositoryId)}/ingestion/status`, {
          signal: ac.signal,
        });
        if (statusRes.ok) {
          const st = (await statusRes.json()) as IngestionStatus;
          setIngestion(st);
          onIngestionStatus?.(st);
        }
        const ownUrl =
          selectedFileId != null
            ? `${base}/repositories/${String(repositoryId)}/ownership?fileId=${encodeURIComponent(selectedFileId)}`
            : `${base}/repositories/${String(repositoryId)}/ownership`;
        const [ownRes, hotRes] = await Promise.all([
          fetch(ownUrl, { signal: ac.signal }),
          fetch(`${base}/repositories/${String(repositoryId)}/hotspots?limit=8`, { signal: ac.signal }),
        ]);
        if (ownRes.ok) {
          const j = (await ownRes.json()) as { ownership?: OwnershipRow[] };
          setOwnership(Array.isArray(j.ownership) ? j.ownership : []);
        }
        if (hotRes.ok) {
          const j = (await hotRes.json()) as { hotspots?: HotspotRow[] };
          setHotspots(Array.isArray(j.hotspots) ? j.hotspots : []);
        }
        setLoadError(null);
      } catch (e) {
        if ((e as Error).name === 'AbortError') return;
        setLoadError(e instanceof Error ? e.message : 'Failed to load socio data');
      }
    };
    void run();
    const timer = setInterval(() => void run(), 5000);
    return () => {
      ac.abort();
      clearInterval(timer);
    };
  }, [repositoryId, selectedFileId, onIngestionStatus]);

  const fileOwnership = selectedFileId
    ? ownership.find((o) => String(o.fileId) === selectedFileId)
    : ownership[0];

  const socioRunning =
    ingestion?.socioTechnical.status === 'running' || ingestion?.socioTechnical.status === 'pending';
  const socioSkipped = ingestion?.socioTechnical.status === 'skipped';

  return (
    <div className="socio-panels">
      {ingestion?.graphCompleteness.partialDataWarning ? (
        <div className="socio-banner" role="status">
          <strong>Partial intelligence</strong>
          <span>
            Code map {ingestion.codeIndex.status} · socio sync {ingestion.socioTechnical.status} (
            {Math.round(ingestion.socioTechnical.completionPercent)}%)
            {socioRunning ? ' — enriching in background' : ''}
          </span>
          {ingestion.socioTechnical.errorDetails ? (
            <span className="socio-banner__err">{ingestion.socioTechnical.errorDetails}</span>
          ) : null}
          {socioSkipped ? (
            <span className="socio-banner__hint">Set GITHUB_TOKEN for GitHub history sync.</span>
          ) : null}
        </div>
      ) : null}

      <section className="socio-panel">
        <h3>Ownership</h3>
        {fileOwnership ? (
          <dl className="socio-stats">
            <div>
              <dt>Owner</dt>
              <dd>{fileOwnership.dominantOwner?.login ?? '—'}</dd>
            </div>
            <div>
              <dt>Contributors</dt>
              <dd>{fileOwnership.contributorCount}</dd>
            </div>
            <div>
              <dt>Bus factor</dt>
              <dd>{fileOwnership.busFactor}</dd>
            </div>
            <div>
              <dt>Risk</dt>
              <dd className={`risk-pill risk-pill--${fileOwnership.riskLevel || 'low'}`}>
                {fileOwnership.riskLevel || 'low'}
              </dd>
            </div>
          </dl>
        ) : (
          <p className="socio-empty">
            {selectedFileId ? 'No ownership metrics for this file yet.' : 'Select a file or wait for GitHub sync.'}
          </p>
        )}
      </section>

      <section className="socio-panel">
        <h3>Hotspots</h3>
        {hotspots.length === 0 ? (
          <p className="socio-empty">Hotspot ranking appears after commit history sync.</p>
        ) : (
          <ul className="hotspot-list">
            {hotspots.map((h) => (
              <li key={h.fileId} className={`hotspot-row risk-pill--${h.riskLevel}`}>
                <span className="hotspot-row__path">{h.path.split('/').pop() ?? h.path}</span>
                <span className="hotspot-row__score">{(h.hotspotScore * 100).toFixed(0)}%</span>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section className="socio-panel socio-panel--muted">
        <h3>Engineering memory</h3>
        <p className="socio-empty">
          Architecture signals from PRs and issues (Phase 2) will surface here after engineering-memory sync.
        </p>
      </section>

      {loadError ? <p className="warning">{loadError}</p> : null}
    </div>
  );
}
