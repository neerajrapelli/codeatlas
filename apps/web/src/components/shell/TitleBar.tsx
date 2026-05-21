import { useMemo } from 'react';

import { CommandPaletteTrigger } from '../ui/CommandPalette';
import { basename } from '../../lib/fileType';
import { dedupeRepositories } from '../../lib/repositories';
import { useStore } from '../../store';

export function TitleBar({ onOpenPalette }: { onOpenPalette: () => void }) {
  const repositories = useStore((s) => s.repositories);
  const activeRepoId = useStore((s) => s.activeRepoId);
  const setActiveRepo = useStore((s) => s.setActiveRepo);
  const selectedNodePath = useStore((s) => s.selectedNodePath);
  const graphPrefix = useStore((s) => s.graphPrefix);

  const repoList = useMemo(() => dedupeRepositories(repositories), [repositories]);
  const active = repoList.find((r) => r.id === activeRepoId);

  const crumbLabel = selectedNodePath
    ? basename(selectedNodePath)
    : graphPrefix
      ? graphPrefix
      : 'Repository root';

  const crumbTitle = selectedNodePath
    ? selectedNodePath
    : graphPrefix
      ? `${active?.name ?? ''} / ${graphPrefix}`
      : active?.name ?? 'Repository root';

  return (
    <header className="title-bar">
      <div className="title-bar__brand">
        <i className="codicon codicon-symbol-class title-bar__brand-icon" aria-hidden />
        <span className="title-bar__brand-text">CodeAtlas</span>
      </div>
      <select
        className="title-bar__repo-select"
        value={activeRepoId ?? ''}
        onChange={(e) => {
          const v = e.target.value;
          setActiveRepo(v ? Number(v) : null);
        }}
        aria-label="Repository"
        title={active?.name}
      >
        <option value="">Select repository…</option>
        {repoList.map((r) => (
          <option key={r.id} value={r.id}>
            {r.name}
          </option>
        ))}
      </select>
      <div className="title-bar__search-slot">
        <CommandPaletteTrigger onOpen={onOpenPalette} />
      </div>
      <span className="title-bar__breadcrumb" title={crumbTitle}>
        {active ? <span className="title-bar__breadcrumb-repo">{active.name}</span> : null}
        {active ? <span className="title-bar__breadcrumb-sep">/</span> : null}
        <span className="title-bar__breadcrumb-path">{crumbLabel}</span>
      </span>
    </header>
  );
}
