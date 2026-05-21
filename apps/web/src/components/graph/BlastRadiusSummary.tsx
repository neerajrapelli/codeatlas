import { useStore } from '../../store';

function riskLabel(score: number): string {
  if (score >= 0.6) return 'HIGH';
  if (score >= 0.35) return 'MEDIUM';
  return 'LOW';
}

export function BlastRadiusSummary() {
  const blast = useStore((s) => s.blastRadius);
  const clearBlastRadius = useStore((s) => s.clearBlastRadius);

  if (!blast) return null;

  const br = blast.blast_radius;
  const path = blast.target.file_path;

  return (
    <div className="blast-radius-panel">
      <h4>
        BLAST RADIUS: <span className="mono">{path}</span>
      </h4>
      <p className="blast-radius-panel__stats">
        {br.total_files_affected} files affected
        <br />
        {br.direct_dependents} direct · {br.transitive_dependents} transitive
        <br />
        Risk score: <strong>{riskLabel(br.risk_score)}</strong> ({br.risk_score.toFixed(2)})
      </p>
      {blast.warnings.length > 0 ? (
        <ul className="blast-radius-panel__warnings">
          {blast.warnings.map((w) => (
            <li key={w}>⚠ {w}</li>
          ))}
        </ul>
      ) : null}
      <div className="blast-radius-panel__actions">
        <button type="button" className="btn-secondary" onClick={() => clearBlastRadius()}>
          Clear
        </button>
      </div>
    </div>
  );
}
