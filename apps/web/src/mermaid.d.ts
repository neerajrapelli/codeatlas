declare module 'mermaid' {
  interface MermaidAPI {
    initialize(config: Record<string, unknown>): void;
    run(options: { nodes: HTMLElement[] }): Promise<void>;
  }
  const mermaid: MermaidAPI;
  export default mermaid;
}
