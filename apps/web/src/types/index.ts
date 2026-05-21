export type SourceType = 'github' | 'gitlab' | 'bitbucket' | 'zip';

export type RepoStatus =
  | 'queued'
  | 'cloning'
  | 'extracting'
  | 'indexing'
  | 'parsing'
  | 'building_graph'
  | 'generating_embeddings'
  | 'ready'
  | 'failed';

export interface Repository {
  id: number;
  name: string;
  sourceType: SourceType;
  sourceUrl: string;
  branch: string;
  workspacePath: string;
  status: RepoStatus;
  progressPercent?: number;
  filesIndexed?: number;
  symbolsIndexed?: number;
  edgesIndexed?: number;
  embeddingsIndexed?: number;
  errorDetails?: string;
  createdAt: string;
  updatedAt: string;
}

export type SidebarView = 'repos' | 'map' | 'hotspots' | 'signals' | 'ownership' | 'timeline';

export type FileType = 'ts' | 'go' | 'css' | 'test' | 'config' | 'other';

export interface FileOverlay {
  fileId: string;
  isHotspot: boolean;
  hasBusFactorRisk: boolean;
  riskLevel?: string;
  architectureSignalCount?: number;
  dominantOwnerLogin?: string;
}

export interface ClusterFile {
  id: string;
  path: string;
  symbolCount: number;
}

export interface ClusterLayer {
  prefix: string;
  clusters: Array<{
    id: string;
    label: string;
    pathPrefix: string;
    level: number;
    fileCount: number;
    internalEdges: number;
    density: number;
    hasChildren: boolean;
  }>;
  files: ClusterFile[];
  edges: Array<{ from: string; to: string; count: number }>;
  socioOverlay?: { fileOverlays?: Record<string, FileOverlay> };
}

export interface GraphFileDetail {
  id: string;
  path: string;
  imports: string[];
  exports: string[];
  symbols: Array<{ name: string; kind: string }>;
}

export interface HotspotEntry {
  fileId: number;
  path: string;
  hotspotScore: number;
  churnScore: number;
  riskLevel: string;
  busFactor: number;
  commitCount90d: number;
}

export interface OwnershipSummary {
  fileId: number;
  path: string;
  contributorCount: number;
  busFactor: number;
  riskLevel: string;
  dominantOwnerShare: number;
  dominantOwner?: { login: string; displayName?: string };
  contributors?: Array<{
    contributor: { login: string };
    share: number;
    commitCount: number;
  }>;
}

export interface IngestionStatusPayload {
  repositoryId: number;
  codeIndex: {
    status: string;
    stage: string;
    progressPercent: number;
    filesIndexed: number;
  };
  socioTechnical: {
    phase: string;
    status: string;
    completionPercent: number;
    staleness: string;
    errorDetails?: string;
    steps?: Array<{ step: string; status: string; itemsProcessed: number }>;
  };
  graphCompleteness: {
    codeGraphReady: boolean;
    socioHistoryReady: boolean;
    partialDataWarning: boolean;
  };
}

export interface RelatedFile {
  fileId: number;
  path: string;
  reason: string;
}

export interface ChatMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  relatedFiles: RelatedFile[];
}

export interface BlastRadiusTarget {
  file_path: string;
  symbol?: string;
  owner?: string;
  bus_factor_score: number;
}

export interface BlastRadiusSummary {
  direct_dependents: number;
  transitive_dependents: number;
  total_files_affected: number;
  risk_score: number;
  teams_affected: string[];
}

export interface BlastRadiusAffectedFile {
  file_path: string;
  depth: number;
  relationship: string;
  owner?: string;
  has_tests: boolean;
  risk_level: string;
}

export interface BlastRadiusResult {
  target: BlastRadiusTarget;
  blast_radius: BlastRadiusSummary;
  files: BlastRadiusAffectedFile[];
  warnings: string[];
}
