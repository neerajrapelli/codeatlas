import { create } from 'zustand';

import { applyTheme } from '../lib/theme';
import type { ThemeMode } from '../lib/theme';
import type {
  ArchitectureRule,
  BlastRadiusResult,
  ChatMessage,
  ClusterLayer,
  HotspotEntry,
  IngestionStatusPayload,
  OwnershipSummary,
  Repository,
  RuleViolation,
  SidebarView,
} from '../types';

export type ApiStatus = 'checking' | 'online' | 'degraded' | 'offline';

interface CodeAtlasStore {
  repositories: Repository[];
  setRepositories: (repos: Repository[]) => void;
  activeRepoId: number | null;
  setActiveRepo: (id: number | null) => void;

  graphPrefix: string;
  setGraphPrefix: (prefix: string) => void;
  clusterLayer: ClusterLayer | null;
  setClusterLayer: (layer: ClusterLayer | null) => void;

  selectedNodeId: string | null;
  selectedNodePath: string | null;
  setSelectedNode: (id: string | null, path?: string | null) => void;

  highlightedFileIds: Set<string>;
  setHighlightedFileIds: (ids: Set<string>) => void;

  graphHoverFileId: string | null;
  setGraphHoverFileId: (id: string | null) => void;

  sidebarView: SidebarView;
  setSidebarView: (v: SidebarView) => void;
  sidebarVisible: boolean;
  toggleSidebar: () => void;

  inspectorOpen: boolean;
  setInspectorOpen: (open: boolean) => void;
  toggleInspector: () => void;

  bottomPanelOpen: boolean;
  toggleBottomPanel: () => void;
  aiPanelWidth: number;
  setAiPanelWidth: (w: number) => void;

  ingestionStatus: IngestionStatusPayload | null;
  setIngestionStatus: (s: IngestionStatusPayload | null) => void;

  hotspots: HotspotEntry[];
  setHotspots: (h: HotspotEntry[]) => void;

  ownershipRows: OwnershipSummary[];
  setOwnershipRows: (rows: OwnershipSummary[]) => void;

  fileDetail: import('../types').GraphFileDetail | null;
  setFileDetail: (f: import('../types').GraphFileDetail | null) => void;

  chatMessages: ChatMessage[];
  addChatMessage: (msg: ChatMessage) => void;
  updateChatMessage: (id: string, patch: Partial<ChatMessage>) => void;
  clearChat: () => void;

  commandPaletteOpen: boolean;
  paletteMode: 'files' | 'unified';
  setCommandPaletteOpen: (open: boolean) => void;
  openPalette: (mode: 'files' | 'unified') => void;

  aiPanelDraft: string | null;
  setAiPanelDraft: (draft: string | null) => void;

  progressPopoverOpen: boolean;
  setProgressPopoverOpen: (open: boolean) => void;

  graphLoading: boolean;
  setGraphLoading: (v: boolean) => void;

  blastRadius: BlastRadiusResult | null;
  blastDepthByPath: Record<string, number>;
  blastTargetPath: string | null;
  setBlastRadius: (result: BlastRadiusResult | null) => void;
  clearBlastRadius: () => void;

  architectureRules: ArchitectureRule[];
  setArchitectureRules: (rules: ArchitectureRule[]) => void;
  ruleViolations: RuleViolation[];
  setRuleViolations: (v: RuleViolation[]) => void;

  toast: { message: string; variant: 'info' | 'success' | 'error' } | null;
  pushToast: (message: string, variant?: 'info' | 'success' | 'error') => void;
  clearToast: () => void;

  theme: ThemeMode;
  setTheme: (theme: ThemeMode) => void;

  apiStatus: ApiStatus;
  setApiStatus: (status: ApiStatus) => void;

  socioLoading: boolean;
  setSocioLoading: (loading: boolean) => void;

  tourStep: number | null;
  setTourStep: (step: number | null) => void;
  completeTour: () => void;

  focusRepoInput: boolean;
  setFocusRepoInput: (focus: boolean) => void;
}

export const useStore = create<CodeAtlasStore>((set) => ({
  repositories: [],
  setRepositories: (repositories) => set({ repositories }),
  activeRepoId: null,
  setActiveRepo: (activeRepoId) =>
    set((state) => {
      const changed = activeRepoId !== state.activeRepoId;
      return {
        activeRepoId,
        selectedNodeId: changed ? null : state.selectedNodeId,
        selectedNodePath: changed ? null : state.selectedNodePath,
        graphPrefix: changed ? '' : state.graphPrefix,
        clusterLayer: changed ? null : state.clusterLayer,
        fileDetail: changed ? null : state.fileDetail,
      };
    }),

  graphPrefix: '',
  setGraphPrefix: (graphPrefix) => set({ graphPrefix }),
  clusterLayer: null,
  setClusterLayer: (clusterLayer) => set({ clusterLayer }),

  selectedNodeId: null,
  selectedNodePath: null,
  setSelectedNode: (selectedNodeId, selectedNodePath = null) =>
    set({
      selectedNodeId,
      selectedNodePath,
      inspectorOpen: selectedNodeId !== null,
    }),

  highlightedFileIds: new Set(),
  setHighlightedFileIds: (highlightedFileIds) => set({ highlightedFileIds }),
  graphHoverFileId: null,
  setGraphHoverFileId: (graphHoverFileId) => set({ graphHoverFileId }),

  sidebarView: 'map',
  setSidebarView: (sidebarView) => set({ sidebarView, sidebarVisible: true }),
  sidebarVisible: true,
  toggleSidebar: () => set((s) => ({ sidebarVisible: !s.sidebarVisible })),

  inspectorOpen: false,
  setInspectorOpen: (inspectorOpen) => set({ inspectorOpen }),
  toggleInspector: () => set((s) => ({ inspectorOpen: !s.inspectorOpen })),

  bottomPanelOpen: true,
  toggleBottomPanel: () => set((s) => ({ bottomPanelOpen: !s.bottomPanelOpen })),
  aiPanelWidth: 384,
  setAiPanelWidth: (aiPanelWidth) =>
    set({ aiPanelWidth: Math.min(560, Math.max(280, aiPanelWidth)) }),

  ingestionStatus: null,
  setIngestionStatus: (ingestionStatus) => set({ ingestionStatus }),

  hotspots: [],
  setHotspots: (hotspots) => set({ hotspots }),

  ownershipRows: [],
  setOwnershipRows: (ownershipRows) => set({ ownershipRows }),

  fileDetail: null,
  setFileDetail: (fileDetail) => set({ fileDetail }),

  chatMessages: [],
  addChatMessage: (msg) => set((s) => ({ chatMessages: [...s.chatMessages, msg] })),
  updateChatMessage: (id, patch) =>
    set((s) => ({
      chatMessages: s.chatMessages.map((m) => (m.id === id ? { ...m, ...patch } : m)),
    })),
  clearChat: () => set({ chatMessages: [] }),

  commandPaletteOpen: false,
  paletteMode: 'unified',
  setCommandPaletteOpen: (commandPaletteOpen) =>
    set(commandPaletteOpen ? { commandPaletteOpen } : { commandPaletteOpen, paletteMode: 'unified' }),
  openPalette: (paletteMode) => set({ commandPaletteOpen: true, paletteMode }),
  aiPanelDraft: null,
  setAiPanelDraft: (aiPanelDraft) => set({ aiPanelDraft }),

  progressPopoverOpen: false,
  setProgressPopoverOpen: (progressPopoverOpen) => set({ progressPopoverOpen }),

  graphLoading: true,
  setGraphLoading: (graphLoading) => set({ graphLoading }),

  blastRadius: null,
  blastDepthByPath: {},
  blastTargetPath: null,
  setBlastRadius: (blastRadius) => {
    if (!blastRadius) {
      set({ blastRadius: null, blastDepthByPath: {}, blastTargetPath: null });
      return;
    }
    const blastDepthByPath: Record<string, number> = {};
    for (const f of blastRadius.files) {
      blastDepthByPath[f.file_path] = f.depth;
    }
    set({
      blastRadius,
      blastDepthByPath,
      blastTargetPath: blastRadius.target.file_path,
      inspectorOpen: true,
    });
  },
  clearBlastRadius: () =>
    set({ blastRadius: null, blastDepthByPath: {}, blastTargetPath: null }),

  architectureRules: [],
  setArchitectureRules: (architectureRules) => set({ architectureRules }),
  ruleViolations: [],
  setRuleViolations: (ruleViolations) => set({ ruleViolations }),

  toast: null,
  pushToast: (message, variant = 'info') => set({ toast: { message, variant } }),
  clearToast: () => set({ toast: null }),

  theme: 'system',
  setTheme: (theme) => {
    applyTheme(theme);
    set({ theme });
  },

  apiStatus: 'checking',
  setApiStatus: (apiStatus) => set({ apiStatus }),

  socioLoading: false,
  setSocioLoading: (socioLoading) => set({ socioLoading }),

  tourStep: null,
  setTourStep: (tourStep) => set({ tourStep }),
  completeTour: () => {
    try {
      localStorage.setItem('codeatlas-tour-done', '1');
    } catch {
      /* ignore */
    }
    set({ tourStep: null });
  },

  focusRepoInput: false,
  setFocusRepoInput: (focusRepoInput) => set({ focusRepoInput }),
}));
