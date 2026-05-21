export function GraphSkeleton() {
  return (
    <div
      className="graph-canvas"
      style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(4, 180px)',
        gap: 24,
        padding: 48,
        alignContent: 'start',
        overflow: 'auto',
      }}
    >
      {Array.from({ length: 28 }).map((_, i) => (
        <div key={i} className="node-skeleton" />
      ))}
    </div>
  );
}
