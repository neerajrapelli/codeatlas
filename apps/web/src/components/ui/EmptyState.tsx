import type { ReactNode } from 'react';

export function EmptyState({
  icon = 'codicon-info',
  title,
  description,
  action,
}: {
  icon?: string;
  title: string;
  description?: string;
  action?: ReactNode;
}) {
  return (
    <div className="empty-state-block">
      <i className={`codicon ${icon} empty-state-block__icon`} aria-hidden />
      <h4 className="empty-state-block__title">{title}</h4>
      {description ? <p className="empty-state-block__desc">{description}</p> : null}
      {action ? <div className="empty-state-block__action">{action}</div> : null}
    </div>
  );
}
