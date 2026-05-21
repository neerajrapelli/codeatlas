import ELK from 'elkjs/lib/elk.bundled.js';

const elk = new ELK();

export interface ElkLayoutChild {
  id: string;
  width: number;
  height: number;
}

export interface ElkLayoutEdge {
  id: string;
  sources: string[];
  targets: string[];
}

export interface ElkLayoutRequest {
  id: string;
  children: ElkLayoutChild[];
  edges: ElkLayoutEdge[];
  layoutOptions?: Record<string, string>;
}

export interface ElkLayoutResponse {
  id: string;
  positions: Record<string, { x: number; y: number }>;
  error?: string;
}

self.onmessage = (ev: MessageEvent<ElkLayoutRequest>) => {
  const msg = ev.data;
  void (async () => {
    try {
      const graph = {
        id: 'root',
        layoutOptions: {
          'elk.algorithm': 'layered',
          'elk.direction': 'RIGHT',
          'elk.spacing.nodeNode': '72',
          ...msg.layoutOptions,
        },
        children: msg.children,
        edges: msg.edges,
      };
      const laid = await elk.layout(graph);
      const positions: Record<string, { x: number; y: number }> = {};
      for (const ch of laid.children ?? []) {
        if (ch.id != null && ch.x != null && ch.y != null) {
          positions[ch.id] = { x: ch.x, y: ch.y };
        }
      }
      const out: ElkLayoutResponse = { id: msg.id, positions };
      self.postMessage(out);
    } catch (e) {
      const out: ElkLayoutResponse = {
        id: msg.id,
        positions: {},
        error: e instanceof Error ? e.message : 'ELK layout failed',
      };
      self.postMessage(out);
    }
  })();
};
