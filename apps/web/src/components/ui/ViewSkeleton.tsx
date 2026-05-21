export function ViewSkeleton({ rows = 5 }: { rows?: number }) {
  return (
    <div className="view-skeleton" aria-hidden>
      {Array.from({ length: rows }, (_, i) => (
        <div key={i} className="view-skeleton__row">
          <div className="view-skeleton__line view-skeleton__line--title" />
          <div className="view-skeleton__line view-skeleton__line--sub" />
        </div>
      ))}
    </div>
  );
}
