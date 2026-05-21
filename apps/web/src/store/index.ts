import { create } from 'zustand';

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

  sidebarView: SidebarView;
  setSidebarView: (v: SidebarView) => void;
  sidebarVisible: boolean;
  toggleSidebar: () => void;

  inspectorOpen: boolean;
  setInspectorOpen: (open: boolean) => void;
  toggleInspector: () => void;

  bottomPanelOpen: boolean;
  toggleBottomPanel: () => void;
  bottomPanelHeight: number;
  setBottomPanelHeight: (h: number) => void;

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
  setCommandPaletteOpen: (open: boolean) => void;

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
}

export const useStore = create<CodeAtlasStore>((set) => ({
  repositories: [],
  setRepositories: (repositories) => set({ repositories }),
  activeRepoId: null,
  setActiveRepo: (activeRepoId) =>
    set({
      activeRepoId,
      selectedNodeId: null,
      selectedNodePath: null,
      graphPrefix: '',
      clusterLayer: null,
      fileDetail: null,
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

  sidebarView: 'map',
  setSidebarView: (sidebarView) => set({ sidebarView, sidebarVisible: true }),
  sidebarVisible: true,
  toggleSidebar: () => set((s) => ({ sidebarVisible: !s.sidebarVisible })),

  inspectorOpen: false,
  setInspectorOpen: (inspectorOpen) => set({ inspectorOpen }),
  toggleInspector: () => set((s) => ({ inspectorOpen: !s.inspectorOpen })),

  bottomPanelOpen: true,
  toggleBottomPanel: () => set((s) => ({ bottomPanelOpen: !s.bottomPanelOpen })),
  bottomPanelHeight: 280,
  setBottomPanelHeight: (bottomPanelHeight) => set({ bottomPanelHeight }),

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
  setCommandPaletteOpen: (commandPaletteOpen) => set({ commandPaletteOpen }),

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
}));
