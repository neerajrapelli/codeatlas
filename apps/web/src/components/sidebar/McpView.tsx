import { useMcpLogsQuery, useMcpManifestQuery } from '../../hooks/queries/useMcp';

const MCP_CONFIG = `{
  "mcpServers": {
    "codeatlas": {
      "url": "http://localhost:8080/mcp"
    }
  }
}`;

export function McpView() {
  const manifest = useMcpManifestQuery();
  const logs = useMcpLogsQuery(10);
  const tools = manifest.data ?? [];
  const logRows = logs.data ?? [];

  const copy = async () => {
    await navigator.clipboard.writeText(MCP_CONFIG);
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
        Copy to clipboard
      </button>
      <h4 style={{ marginTop: 16 }}>Available Tools ({tools.length})</h4>
      <ul className="mcp-tools-list">
        {tools.map((t) => (
          <li key={t}>✓ {t}</li>
        ))}
      </ul>
      <h4>Tool Calls (last 10)</h4>
      <ul className="mcp-logs-list">
        {logRows.length === 0 ? (
          <li className="empty-state">No tool calls yet.</li>
        ) : (
          logRows.map((c, i) => (
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
