import { useEffect } from 'react';
import { ReactFlowProvider } from 'reactflow';

import { api } from '../../lib/api';
import { useStore } from '../../store';
import { AIPanel } from '../assistant/AIPanel';
import { GraphCanvas } from '../graph/GraphCanvas';
import { InspectorPanel } from '../inspector/InspectorPanel';
import { SidebarContainer } from '../sidebar/SidebarContainer';
import { CommandPalette } from '../ui/CommandPalette';
import { ProgressPopover } from '../ui/ProgressPopover';
import { ActivityBar } from './ActivityBar';
import { StatusBar } from './StatusBar';
import { TitleBar } from './TitleBar';

export function AppShell() {
  const setRepositories = useStore((s) => s.setRepositories);
  const activeRepoId = useStore((s) => s.activeRepoId);
  const setActiveRepo = useStore((s) => s.setActiveRepo);
  const setHotspots = useStore((s) => s.setHotspots);
  const setOwnershipRows = useStore((s) => s.setOwnershipRows);
  const setCommandPaletteOpen = useStore((s) => s.setCommandPaletteOpen);
  const toggleSidebar = useStore((s) => s.toggleSidebar);
  const toggleBottomPanel = useStore((s) => s.toggleBottomPanel);
  const toggleInspector = useStore((s) => s.toggleInspector);
  const setSelectedNode = useStore((s) => s.setSelectedNode);
  const setCommandPaletteOpen2 = useStore((s) => s.setCommandPaletteOpen);
  const setProgressPopoverOpen = useStore((s) => s.setProgressPopoverOpen);
  const setSidebarView = useStore((s) => s.setSidebarView);

  useEffect(() => {
    void api.listRepositories().then((repos) => {
      setRepositories(repos);
      if (repos.length && activeRepoId == null) {
        const ready = repos.find((r) => r.status === 'ready');
        const first = repos[0];
        if (first) setActiveRepo(ready?.id ?? first.id);
      }
    });
    const t = setInterval(() => {
      void api.listRepositories().then(setRepositories).catch(() => undefined);
    }, 8000);
    return () => clearInterval(t);
  }, [setRepositories, activeRepoId, setActiveRepo]);

  useEffect(() => {
    if (activeRepoId == null) return;
    void api.getHotspots(activeRepoId, 30).then(setHotspots).catch(() => setHotspots([]));
    void api.getOwnership(activeRepoId).then(setOwnershipRows).catch(() => setOwnershipRows([]));
    const t = setInterval(() => {
      void api.getHotspots(activeRepoId, 30).then(setHotspots).catch(() => undefined);
    }, 10000);
    return () => clearInterval(t);
  }, [activeRepoId, setHotspots, setOwnershipRows]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const mod = e.metaKey || e.ctrlKey;
      if (mod && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        setCommandPaletteOpen(true);
      }
      if (mod && e.key.toLowerCase() === 'b') {
        e.preventDefault();
        toggleSidebar();
      }
      if (mod && e.key.toLowerCase() === 'j') {
        e.preventDefault();
        toggleBottomPanel();
      }
      if (mod && e.key === '\\') {
        e.preventDefault();
        toggleInspector();
      }
      if (e.key === 'Escape') {
        setCommandPaletteOpen2(false);
        setSelectedNode(null, null);
      }
      if (mod && e.key >= '1' && e.key <= '6') {
        const views = ['repos', 'map', 'hotspots', 'signals', 'ownership', 'drift', 'mcp', 'timeline'] as const;
        const i = Number(e.key) - 1;
        if (views[i]) setSidebarView(views[i]);
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [
    setCommandPaletteOpen,
    toggleSidebar,
    toggleBottomPanel,
    toggleInspector,
    setSelectedNode,
    setCommandPaletteOpen2,
    setSidebarView,
  ]);

  return (
    <div className="app-shell">
      <TitleBar onOpenPalette={() => setCommandPaletteOpen(true)} />
      <div className="app-shell__body">
        <ActivityBar />
        <SidebarContainer />
        <div className="editor-column">
          <ReactFlowProvider>
            <GraphCanvas />
          </ReactFlowProvider>
          <AIPanel />
        </div>
        <InspectorPanel />
      </div>
      <StatusBar onClick={() => setProgressPopoverOpen(true)} />
      <CommandPalette />
      <ProgressPopover />
    </div>
  );
}
