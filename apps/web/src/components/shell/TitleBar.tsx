import { CommandPaletteTrigger } from '../ui/CommandPalette';
import { useStore } from '../../store';
import { basename } from '../../lib/fileType';

export function TitleBar({ onOpenPalette }: { onOpenPalette: () => void }) {
  const repositories = useStore((s) => s.repositories);
  const activeRepoId = useStore((s) => s.activeRepoId);
  const setActiveRepo = useStore((s) => s.setActiveRepo);
  const selectedNodePath = useStore((s) => s.selectedNodePath);
  const graphPrefix = useStore((s) => s.graphPrefix);

  const active = repositories.find((r) => r.id === activeRepoId);
  const crumb = selectedNodePath
    ? selectedNodePath
    : graphPrefix
      ? graphPrefix
      : 'Repository root';

  return (
    <header className="title-bar">
      <div className="title-bar__brand">
        <i className="codicon codicon-symbol-class" />
        <span>CodeAtlas</span>
      </div>
      <select
        className="title-bar__repo-select"
        value={activeRepoId ?? ''}
        onChange={(e) => {
          const v = e.target.value;
          setActiveRepo(v ? Number(v) : null);
        }}
        aria-label="Repository"
      >
        <option value="">Select repository…</option>
        {repositories.map((r) => (
          <option key={r.id} value={r.id}>
            {r.name}
          </option>
        ))}
      </select>
      <div className="title-bar__search-slot">
        <CommandPaletteTrigger onOpen={onOpenPalette} />
      </div>
      <span className="title-bar__breadcrumb" title={crumb}>
        {active ? `${active.name} / ` : ''}
        {selectedNodePath ? basename(selectedNodePath) : crumb}
      </span>
    </header>
  );
}
