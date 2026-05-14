/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_URL?: string;
  /** Dev-only: proxy target for `/api` (default http://localhost:8080). See vite.config.ts */
  readonly VITE_API_PROXY_TARGET?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}

declare const __CODEATLAS_API_URL__: string;

declare module 'elkjs/lib/elk.bundled.js' {
  export default class ELK {
    layout(graph: unknown): Promise<{
      children?: Array<{ id: string; x?: number; y?: number }>;
    }>;
  }
}
