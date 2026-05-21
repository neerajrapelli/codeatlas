import type { ElkLayoutEdge, ElkLayoutRequest, ElkLayoutResponse } from '../workers/elkLayout.worker';

let worker: Worker | null = null;
let seq = 0;

function getWorker(): Worker {
  if (!worker) {
    worker = new Worker(new URL('../workers/elkLayout.worker.ts', import.meta.url), {
      type: 'module',
    });
  }
  return worker;
}

export async function runElkLayout(
  children: ElkLayoutRequest['children'],
  edges: ElkLayoutEdge[],
  layoutOptions?: Record<string, string>,
): Promise<Record<string, { x: number; y: number }>> {
  const id = `elk-${String(++seq)}`;
  const w = getWorker();
  return new Promise((resolve, reject) => {
    const onMessage = (ev: MessageEvent<ElkLayoutResponse>) => {
      if (ev.data.id !== id) return;
      w.removeEventListener('message', onMessage);
      w.removeEventListener('error', onError);
      if (ev.data.error) reject(new Error(ev.data.error));
      else resolve(ev.data.positions);
    };
    const onError = () => {
      w.removeEventListener('message', onMessage);
      w.removeEventListener('error', onError);
      reject(new Error('ELK worker error'));
    };
    w.addEventListener('message', onMessage);
    w.addEventListener('error', onError);
    w.postMessage({ id, children, edges, layoutOptions } satisfies ElkLayoutRequest);
  });
}

export function terminateElkWorker(): void {
  worker?.terminate();
  worker = null;
}
