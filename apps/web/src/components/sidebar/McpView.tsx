import { useCallback, useEffect, useState } from 'react';

import { apiFetch } from '../../lib/apiFetch';

const MCP_CONFIG = `{
  "mcpServers": {
    "codeatlas": {
      "url": "http://localhost:8080/mcp"
    }
  }
}`;

export function McpView() {
  const [tools, setTools] = useState<string[]>([]);
  const [logs, setLogs] = useState<Array<{ tool: string; at: string; ok: boolean; error?: string }>>([]);
  const [copied, setCopied] = useState(false);

  const refresh = useCallback(async () => {
    try {
      const m = await apiFetch('/mcp/manifest');
      if (m.ok) {
        const json = (await m.json()) as { tools?: Array<{ name: string }> };
        setTools((json.tools ?? []).map((t) => t.name));
      }
      const l = await apiFetch('/mcp/logs?limit=10');
      if (l.ok) {
        const json = (await l.json()) as {
          calls?: Array<{ tool: string; at: string; ok: boolean; error?: string }>;
        };
        setLogs(json.calls ?? []);
      }
    } catch {
      /* ignore */
    }
  }, []);

  useEffect(() => {
    void refresh();
    const id = window.setInterval(() => void refresh(), 3000);
    return () => window.clearInterval(id);
  }, [refresh]);

  const copy = async () => {
    await navigator.clipboard.writeText(MCP_CONFIG);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="sidebar-view">
      <h3 className="sidebar-section-title">MCP SERVER</h3>
      <p style={{ fontSize: 'var(--font-size-xs)', color: 'var(--text-secondary)' }}>
        Status: <span style={{ color: 'var(--color-success)' }}>●</span> Running on :8080/mcp
      </p>
      <p style={{ fontSize: 'var(--font-size-xs)' }}>Add to Cursor:</p>
      <pre className="mcp-config-block">{MCP_CONFIG}</pre>
      <button type="button" className="btn-secondary" onClick={() => void copy()}>
        {copied ? 'Copied!' : 'Copy to clipboard'}
      </button>
      <h4 style={{ marginTop: 16 }}>Available Tools ({tools.length})</h4>
      <ul className="mcp-tools-list">
        {tools.map((t) => (
          <li key={t}>✓ {t}</li>
        ))}
      </ul>
      <h4>Tool Calls (last 10)</h4>
      <ul className="mcp-logs-list">
        {logs.length === 0 ? (
          <li className="empty-state">No tool calls yet.</li>
        ) : (
          logs.map((c, i) => (
            <li key={`${c.tool}-${c.at}-${String(i)}`}>
              {c.ok ? '✓' : '✗'} {c.tool}{' '}
              <span className="mono" style={{ color: 'var(--text-secondary)' }}>
                {new Date(c.at).toLocaleTimeString()}
              </span>
              {c.error ? ` — ${c.error}` : ''}
            </li>
          ))
        )}
      </ul>
    </div>
  );
}
