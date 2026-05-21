import { useEffect, useRef, useState } from 'react';

import { api } from '../../lib/api';
import { useStore } from '../../store';
import { AssistantMessage } from './AssistantMessage';

const PROMPTS = [
  'What breaks if auth changes?',
  'Show highest risk files',
  'Who owns checkout?',
  'Trace this dependency',
];

export function AIPanel() {
  const bottomPanelOpen = useStore((s) => s.bottomPanelOpen);
  const toggleBottomPanel = useStore((s) => s.toggleBottomPanel);
  const setAiPanelWidth = useStore((s) => s.setAiPanelWidth);
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
  const aiPanelDraft = useStore((s) => s.aiPanelDraft);
  const setAiPanelDraft = useStore((s) => s.setAiPanelDraft);

  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const repo = repositories.find((r) => r.id === activeRepoId);

  useEffect(() => {
    if (aiPanelDraft) {
      setInput(aiPanelDraft);
      setAiPanelDraft(null);
    }
  }, [aiPanelDraft, setAiPanelDraft]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' });
  }, [chatMessages, loading]);

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
          if (ev.type === 'validated') {
            const paths = ev.validation?.paths ?? {};
            const rules = ev.validation?.rules ?? {};
            const prev = useStore.getState().chatMessages.find((m) => m.id === aid);
            const related = (prev?.relatedFiles ?? []).filter((f) => paths[f.path] !== false);
            updateChatMessage(aid, {
              content: ev.content ?? prev?.content ?? '',
              relatedFiles: related,
              pathValidation: paths,
              ruleValidation: rules,
            });
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
    <aside className="ai-panel" aria-label="Architecture assistant">
      <div
        className="ai-panel__resize"
        role="separator"
        aria-orientation="vertical"
        aria-label="Resize assistant panel"
        onMouseDown={(e) => {
          const startX = e.clientX;
          const startW = useStore.getState().aiPanelWidth;
          const onMove = (ev: MouseEvent) => {
            setAiPanelWidth(startW + (startX - ev.clientX));
          };
          const onUp = () => {
            window.removeEventListener('mousemove', onMove);
            window.removeEventListener('mouseup', onUp);
          };
          window.addEventListener('mousemove', onMove);
          window.addEventListener('mouseup', onUp);
        }}
      />
      <header className="ai-panel__header">
        <span className="ai-panel__title">
          <i className="codicon codicon-sparkle" aria-hidden /> Assistant
        </span>
        <button type="button" className="btn-icon" onClick={toggleBottomPanel} aria-label="Close assistant">
          <i className="codicon codicon-close" />
        </button>
      </header>
      <div className="ai-grounding">
        <span className="ai-grounding__dot" aria-hidden />
        <span className="ai-grounding__text">
          {repo?.name ?? '—'} · {graphPrefix || 'root'}
          {selectedNodePath ? ` · ${selectedNodePath}` : ''}
        </span>
      </div>
      <div className="ai-messages">
        {chatMessages.length === 0 ? (
          <div className="ai-panel__empty">
            <p className="ai-panel__empty-title">Ask about architecture</p>
            <p className="ai-panel__empty-desc">
              Grounded answers use your indexed graph, ownership, and risk signals.
            </p>
            <div className="prompt-chips">
              {PROMPTS.map((p) => (
                <button key={p} type="button" className="prompt-chip" onClick={() => void send(p)}>
                  {p}
                </button>
              ))}
            </div>
          </div>
        ) : (
          <>
            {chatMessages.map((msg) =>
              msg.role === 'user' ? (
                <div key={msg.id} className="ai-msg-user">
                  <div className="ai-msg-user__bubble">{msg.content}</div>
                </div>
              ) : (
                <AssistantMessage
                  key={msg.id}
                  content={msg.content}
                  relatedFiles={msg.relatedFiles}
                  pathValidation={msg.pathValidation}
                  onSelectFile={setSelectedNode}
                />
              ),
            )}
            {loading ? (
              <div className="ai-msg-assistant ai-msg-assistant--typing">
                <span className="ai-typing-dot" />
                <span className="ai-typing-dot" />
                <span className="ai-typing-dot" />
              </div>
            ) : null}
            <div ref={messagesEndRef} />
          </>
        )}
      </div>
      <footer className="ai-input-footer">
        <div className="ai-input-row">
          <textarea
            className="ai-input"
            rows={2}
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
          <button
            type="button"
            className="ai-input-send"
            disabled={loading || !input.trim()}
            onClick={() => void send(input)}
            aria-label="Send message"
          >
            <i className="codicon codicon-arrow-up" />
          </button>
        </div>
      </footer>
    </aside>
  );
}
