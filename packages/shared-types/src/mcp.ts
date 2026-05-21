export interface McpToolCallLog {
  readonly tool: string;
  readonly repositoryId?: string;
  readonly at: string;
  readonly ok: boolean;
  readonly error?: string;
}

export interface McpManifest {
  readonly name: string;
  readonly version: string;
  readonly description: string;
  readonly tools: ReadonlyArray<{ readonly name: string; readonly description: string }>;
}
