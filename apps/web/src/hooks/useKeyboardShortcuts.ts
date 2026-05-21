import { useEffect } from 'react';

import { useStore } from '../store';

export function useKeyboardShortcuts() {
  const openPalette = useStore((s) => s.openPalette);
  const setCommandPaletteOpen = useStore((s) => s.setCommandPaletteOpen);
  const toggleSidebar = useStore((s) => s.toggleSidebar);
  const toggleBottomPanel = useStore((s) => s.toggleBottomPanel);
  const toggleInspector = useStore((s) => s.toggleInspector);
  const setSelectedNode = useStore((s) => s.setSelectedNode);
  const setSidebarView = useStore((s) => s.setSidebarView);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const mod = e.metaKey || e.ctrlKey;
      const key = e.key.toLowerCase();
      if (mod && key === 'k') {
        e.preventDefault();
        openPalette('commands');
      }
      if (mod && key === 'p' && e.shiftKey) {
        e.preventDefault();
        openPalette('commands');
      }
      if (mod && key === 'p' && !e.shiftKey) {
        e.preventDefault();
        openPalette('files');
      }
      if (mod && key === 'b') {
        e.preventDefault();
        toggleSidebar();
      }
      if (mod && key === 'j') {
        e.preventDefault();
        toggleBottomPanel();
      }
      if (mod && e.key === '\\') {
        e.preventDefault();
        toggleInspector();
      }
      if (mod && e.key === ',') {
        e.preventDefault();
        setSidebarView('settings');
      }
      if (e.key === 'Escape') {
        if (useStore.getState().commandPaletteOpen) {
          setCommandPaletteOpen(false);
          return;
        }
        setSelectedNode(null, null);
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [
    openPalette,
    setCommandPaletteOpen,
    toggleSidebar,
    toggleBottomPanel,
    toggleInspector,
    setSelectedNode,
    setSidebarView,
  ]);
}
