import { Suspense, lazy } from 'react';
import { ReactFlowProvider } from 'reactflow';

import { useBackend } from '../../hooks/useBackend';
import { useKeyboardShortcuts } from '../../hooks/useKeyboardShortcuts';
import { InspectorPanel } from '../inspector/InspectorPanel';
import { SidebarContainer } from '../sidebar/SidebarContainer';
import { CommandPalette } from '../ui/CommandPalette';
import { ProductTour } from '../ui/ProductTour';
import { ProgressPopover } from '../ui/ProgressPopover';
import { Toast } from '../ui/Toast';
import { ActivityBar } from './ActivityBar';
import { StatusBar } from './StatusBar';
import { TitleBar } from './TitleBar';
import { GraphSkeleton } from '../graph/GraphSkeleton';
import { useStore } from '../../store';

const GraphCanvas = lazy(() =>
  import('../graph/GraphCanvas').then((m) => ({ default: m.GraphCanvas })),
);
const AIPanel = lazy(() => import('../assistant/AIPanel').then((m) => ({ default: m.AIPanel })));

function GraphArea() {
  return (
    <ReactFlowProvider>
      <Suspense fallback={<GraphSkeleton />}>
        <GraphCanvas />
      </Suspense>
    </ReactFlowProvider>
  );
}

export function AppShell() {
  useBackend();
  useKeyboardShortcuts();
  const openPalette = useStore((s) => s.openPalette);
  const setProgressPopoverOpen = useStore((s) => s.setProgressPopoverOpen);
  const bottomPanelOpen = useStore((s) => s.bottomPanelOpen);
  const aiPanelWidth = useStore((s) => s.aiPanelWidth);

  return (
    <div className="app-shell">
      <TitleBar onOpenPalette={() => openPalette('unified')} />
      <div className="app-shell__body">
        <ActivityBar />
        <SidebarContainer />
        <div className="workspace-main">
          <section className="workspace-canvas" aria-label="Graph canvas">
            <div className="editor-column">
              <GraphArea />
            </div>
          </section>
          {bottomPanelOpen ? (
            <section
              className="workspace-ai"
              style={{ width: aiPanelWidth }}
              aria-label="AI assistant"
            >
              <Suspense fallback={null}>
                <AIPanel />
              </Suspense>
            </section>
          ) : null}
          <InspectorPanel />
        </div>
      </div>
      <StatusBar onClick={() => setProgressPopoverOpen(true)} />
      <CommandPalette />
      <ProgressPopover />
      <ProductTour />
      <Toast />
    </div>
  );
}
