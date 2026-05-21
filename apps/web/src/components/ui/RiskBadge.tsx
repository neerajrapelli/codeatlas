export function RiskBadge({ level }: { level: string }) {
  const key = level.toLowerCase();
  const cls =
    key === 'critical' || key === 'high'
      ? 'risk-badge--high'
      : key === 'medium'
        ? 'risk-badge--medium'
        : 'risk-badge--low';
  return <span className={`risk-badge ${cls}`}>{level}</span>;
}
