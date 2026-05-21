import { ReactFlowProvider } from 'reactflow';

import { useBackend } from '../../hooks/useBackend';
import { useKeyboardShortcuts } from '../../hooks/useKeyboardShortcuts';
import { AIPanel } from '../assistant/AIPanel';
import { GraphCanvas } from '../graph/GraphCanvas';
import { InspectorPanel } from '../inspector/InspectorPanel';
import { SidebarContainer } from '../sidebar/SidebarContainer';
import { CommandPalette } from '../ui/CommandPalette';
import { ProductTour } from '../ui/ProductTour';
import { ProgressPopover } from '../ui/ProgressPopover';
import { Toast } from '../ui/Toast';
import { ActivityBar } from './ActivityBar';
import { StatusBar } from './StatusBar';
import { TitleBar } from './TitleBar';
import { useStore } from '../../store';

export function AppShell() {
  useBackend();
  useKeyboardShortcuts();
  const openPalette = useStore((s) => s.openPalette);
  const setProgressPopoverOpen = useStore((s) => s.setProgressPopoverOpen);

  return (
    <div className="app-shell">
      <TitleBar onOpenPalette={() => openPalette('commands')} />
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
      <ProductTour />
      <Toast />
    </div>
  );
}
