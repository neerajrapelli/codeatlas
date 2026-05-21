import { useState } from 'react';

import { api } from '../../lib/api';
import { useStore } from '../../store';
import { NodeCitation } from './NodeCitation';

const PROMPTS = [
  'What breaks if auth changes?',
  'Show highest risk files',
  'Who owns checkout?',
  'Trace this dependency',
];

export function AIPanel() {
  const bottomPanelOpen = useStore((s) => s.bottomPanelOpen);
  const toggleBottomPanel = useStore((s) => s.toggleBottomPanel);
  const bottomPanelHeight = useStore((s) => s.bottomPanelHeight);
  const setBottomPanelHeight = useStore((s) => s.setBottomPanelHeight);
  const activeRepoId = useStore((s) => s.activeRepoId);
  const repositories = useStore((s) => s.repositories);
  const graphPrefix = useStore((s) => s.graphPrefix);
  const selectedNodePath = useStore((s) => s.selectedNodePath);
  const chatMessages = useStore((s) => s.chatMessages);
  const addChatMessage = useStore((s) => s.addChatMessage);
  const updateChatMessage = useStore((s) => s.updateChatMessage);
  const setHighlightedFileIds = useStore((s) => s.setHighlightedFileIds);
  const setSelectedNode = useStore((s) => s.setSelectedNode);
  const pushToast = useStore((s) => s.pushToast);

  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);

  const repo = repositories.find((r) => r.id === activeRepoId);

  if (!bottomPanelOpen) return null;

  const send = async (text: string) => {
    if (!text.trim() || activeRepoId == null || loading) return;
    const query = text.trim();
    const ctx: string[] = [];
    if (repo) ctx.push(`Repository: ${repo.name}`);
    if (graphPrefix) ctx.push(`Scope: ${graphPrefix}`);
    if (selectedNodePath) ctx.push(`File: ${selectedNodePath}`);
    const grounded = ctx.length ? `${query}\n\n---\n${ctx.join('\n')}` : query;

    addChatMessage({ id: `u-${Date.now()}`, role: 'user', content: query, relatedFiles: [] });
    setInput('');
    setLoading(true);
    const aid = `a-${Date.now()}`;
    addChatMessage({ id: aid, role: 'assistant', content: '', relatedFiles: [] });

    try {
      await api.chatStream(
        activeRepoId,
        { query: grounded, provider: 'openai', model: 'gpt-4o-mini' },
        (ev) => {
          if (ev.type === 'meta' && ev.relatedFiles) {
            updateChatMessage(aid, { relatedFiles: ev.relatedFiles });
            setHighlightedFileIds(new Set(ev.relatedFiles.map((f) => String(f.fileId))));
          }
          if (ev.type === 'token' && ev.token) {
            const prev = useStore.getState().chatMessages.find((m) => m.id === aid);
            updateChatMessage(aid, { content: (prev?.content ?? '') + ev.token });
          }
        },
      );
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Chat failed';
      updateChatMessage(aid, { content: msg });
      pushToast(msg, 'error');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="bottom-panel" style={{ height: bottomPanelHeight }}>
      <div
        className="bottom-panel__resize"
        onMouseDown={(e) => {
          const startY = e.clientY;
          const startH = bottomPanelHeight;
          const onMove = (ev: MouseEvent) => {
            setBottomPanelHeight(Math.min(480, Math.max(120, startH + (startY - ev.clientY))));
          };
          const onUp = () => {
            window.removeEventListener('mousemove', onMove);
            window.removeEventListener('mouseup', onUp);
          };
          window.addEventListener('mousemove', onMove);
          window.addEventListener('mouseup', onUp);
        }}
      />
      <div className="bottom-panel__header">
        <span>
          <i className="codicon codicon-sparkle" /> Architecture Assistant
        </span>
        <button type="button" className="btn-icon" onClick={toggleBottomPanel} aria-label="Close panel">
          <i className="codicon codicon-close" />
        </button>
      </div>
      <div className="ai-grounding">
        ● Grounded in: {repo?.name ?? '—'} · Scope: {graphPrefix || 'root'}
        {selectedNodePath ? ` · ${selectedNodePath}` : ''}
      </div>
      <div className="ai-messages">
        {chatMessages.map((msg) =>
          msg.role === 'user' ? (
            <div key={msg.id} className="ai-msg-user">
              <div className="ai-msg-user__bubble">{msg.content}</div>
            </div>
          ) : (
            <div key={msg.id} className="ai-msg-assistant">
              {msg.content}
              {msg.relatedFiles.length > 0 ? (
                <div style={{ marginTop: 8 }}>
                  {msg.relatedFiles.map((f) => (
                    <NodeCitation
                      key={f.path}
                      path={f.path}
                      fileId={String(f.fileId)}
                      onSelect={setSelectedNode}
                    />
                  ))}
                </div>
              ) : null}
            </div>
          ),
        )}
      </div>
      {chatMessages.length === 0 ? (
        <div className="prompt-chips">
          {PROMPTS.map((p) => (
            <button key={p} type="button" className="prompt-chip" onClick={() => void send(p)}>
              {p}
            </button>
          ))}
        </div>
      ) : null}
      <div className="ai-input-row">
        <textarea
          className="ai-input"
          rows={1}
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="Ask about architecture, ownership, risk, or impact…"
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault();
              void send(input);
            }
          }}
        />
        <button type="button" className="title-bar__btn" disabled={loading} onClick={() => void send(input)}>
          ↵
        </button>
      </div>
    </div>
  );
}
