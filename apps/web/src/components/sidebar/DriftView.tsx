import { useCallback, useEffect, useState } from 'react';

import { api } from '../../lib/api';
import { useStore } from '../../store';
import type { ArchitectureRule, ArchitectureRuleType, RuleViolation } from '../../types';

export function DriftView() {
  const activeRepoId = useStore((s) => s.activeRepoId);
  const rules = useStore((s) => s.architectureRules);
  const setRules = useStore((s) => s.setArchitectureRules);
  const violations = useStore((s) => s.ruleViolations);
  const setViolations = useStore((s) => s.setRuleViolations);
  const setSelectedNode = useStore((s) => s.setSelectedNode);
  const setSidebarView = useStore((s) => s.setSidebarView);

  const [modalOpen, setModalOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [form, setForm] = useState({
    name: '',
    ruleType: 'no_import' as ArchitectureRuleType,
    sourcePattern: '',
    targetPattern: '',
    severity: 'error' as 'error' | 'warning' | 'info',
  });

  const refresh = useCallback(async () => {
    if (activeRepoId == null) return;
    const [r, v] = await Promise.all([
      api.listRules(activeRepoId),
      api.getViolations(activeRepoId),
    ]);
    setRules(r);
    setViolations(v);
  }, [activeRepoId, setRules, setViolations]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const jumpToFile = (path: string, fileId?: string) => {
    setSidebarView('map');
    if (fileId) setSelectedNode(fileId, path);
  };

  return (
    <div className="sidebar-view">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h3 className="sidebar-section-title">ARCHITECTURE RULES</h3>
        <button type="button" className="btn-secondary" disabled={activeRepoId == null} onClick={() => setModalOpen(true)}>
          + Add Rule
        </button>
      </div>

      {rules.length === 0 ? (
        <p className="empty-state">No rules yet. Add a rule to detect drift.</p>
      ) : (
        <ul className="drift-rules-list">
          {rules.map((rule) => (
            <li key={rule.id} className="drift-rules-list__item">
              <span>{rule.enabled ? '✓' : '−'} {rule.name}</span>
              <button
                type="button"
                className="btn-icon"
                title="Delete rule"
                onClick={() => {
                  if (activeRepoId == null) return;
                  void api.deleteRule(activeRepoId, rule.id).then(refresh);
                }}
              >
                <i className="codicon codicon-trash" />
              </button>
            </li>
          ))}
        </ul>
      )}

      <h3 className="sidebar-section-title" style={{ marginTop: 16 }}>
        VIOLATIONS ({violations.length} active)
      </h3>
      <button
        type="button"
        className="btn-secondary"
        disabled={activeRepoId == null || busy}
        onClick={() => {
          if (activeRepoId == null) return;
          setBusy(true);
          void api
            .validateRules(activeRepoId)
            .then((v) => setViolations(v))
            .finally(() => setBusy(false));
        }}
      >
        {busy ? 'Validating…' : 'Validate now'}
      </button>

      {violations.length === 0 ? (
        <p className="empty-state">No active violations.</p>
      ) : (
        <ul className="drift-violations-list">
          {violations.map((v) => (
            <ViolationRow key={`${v.ruleId}-${v.sourceFile}-${v.targetFile}`} v={v} onJump={jumpToFile} />
          ))}
        </ul>
      )}

      {modalOpen ? (
        <div className="modal-backdrop" role="presentation" onClick={() => setModalOpen(false)}>
          <div className="modal" role="dialog" onClick={(e) => e.stopPropagation()}>
            <h4>Add architecture rule</h4>
            <label>
              Rule name
              <input
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              />
            </label>
            <label>
              Type
              <select
                value={form.ruleType}
                onChange={(e) =>
                  setForm((f) => ({ ...f, ruleType: e.target.value as ArchitectureRuleType }))
                }
              >
                <option value="no_import">no_import</option>
                <option value="must_import">must_import</option>
                <option value="layer_order">layer_order</option>
                <option value="no_circular">no_circular</option>
              </select>
            </label>
            <label>
              Source pattern
              <input
                value={form.sourcePattern}
                onChange={(e) => setForm((f) => ({ ...f, sourcePattern: e.target.value }))}
                placeholder="src/checkout/**"
              />
            </label>
            <label>
              Target pattern
              <input
                value={form.targetPattern}
                onChange={(e) => setForm((f) => ({ ...f, targetPattern: e.target.value }))}
                placeholder="src/billing/internal/**"
              />
            </label>
            <label>
              Severity
              <select
                value={form.severity}
                onChange={(e) =>
                  setForm((f) => ({
                    ...f,
                    severity: e.target.value as 'error' | 'warning' | 'info',
                  }))
                }
              >
                <option value="error">error</option>
                <option value="warning">warning</option>
                <option value="info">info</option>
              </select>
            </label>
            <div className="modal__actions">
              <button type="button" onClick={() => setModalOpen(false)}>
                Cancel
              </button>
              <button
                type="button"
                className="btn-primary"
                onClick={() => {
                  if (activeRepoId == null || !form.name) return;
                  void api
                    .createRule(activeRepoId, form)
                    .then(() => {
                      setModalOpen(false);
                      return refresh();
                    })
                    .then(() => api.validateRules(activeRepoId))
                    .then(setViolations);
                }}
              >
                Save Rule
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}

function ViolationRow({
  v,
  onJump,
}: {
  v: RuleViolation;
  onJump: (path: string) => void;
}) {
  const sev = v.severity === 'error' ? '🔴' : v.severity === 'warning' ? '🟡' : 'ℹ';
  return (
    <li className={`drift-violation drift-violation--${v.severity}`}>
      <div>
        {sev} {v.severity.toUpperCase()}
      </div>
      <div className="mono">{v.sourceFile}</div>
      <div className="mono">→ {v.targetFile}</div>
      <div className="drift-violation__msg">{v.message}</div>
      <button type="button" className="btn-secondary" onClick={() => onJump(v.sourceFile)}>
        Jump to file
      </button>
    </li>
  );
}
