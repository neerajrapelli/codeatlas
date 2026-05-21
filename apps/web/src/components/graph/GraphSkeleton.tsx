export function GraphSkeleton({ message = 'Loading architecture map…' }: { message?: string }) {
  return (
    <div className="graph-skeleton" aria-busy="true" aria-live="polite">
      <div className="graph-skeleton__grid">
        {Array.from({ length: 28 }).map((_, i) => (
          <div key={i} className="node-skeleton" />
        ))}
      </div>
      <div className="graph-skeleton__overlay">
        <span className="graph-skeleton__spinner" aria-hidden />
        <span className="graph-skeleton__label">{message}</span>
      </div>
    </div>
  );
}

/** Semi-transparent overlay while ELK layout runs on top of an existing canvas. */
export function GraphLayoutOverlay({ message = 'Computing layout…' }: { message?: string }) {
  return (
    <div className="graph-layout-overlay" aria-busy="true" aria-live="polite">
      <span className="graph-skeleton__spinner" aria-hidden />
      <span className="graph-skeleton__label">{message}</span>
    </div>
  );
}
