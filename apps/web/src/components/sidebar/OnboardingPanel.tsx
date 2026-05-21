import { useState } from 'react';

import { api } from '../../lib/api';
import { basename } from '../../lib/fileType';
import { useStore } from '../../store';
import type { OnboardingPlan } from '../../types';
import { EmptyState } from '../ui/EmptyState';
import { ViewSkeleton } from '../ui/ViewSkeleton';

export function OnboardingPanel() {
  const activeRepoId = useStore((s) => s.activeRepoId);
  const clusterLayer = useStore((s) => s.clusterLayer);
  const setSelectedNode = useStore((s) => s.setSelectedNode);
  const setSidebarView = useStore((s) => s.setSidebarView);
  const [role, setRole] = useState('backend');
  const [focus, setFocus] = useState('');
  const [level, setLevel] = useState('mid');
  const [plan, setPlan] = useState<OnboardingPlan | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const openFile = (path: string) => {
    const f = clusterLayer?.files?.find((x) => x.path === path);
    if (f) setSelectedNode(f.id, f.path);
    else setSelectedNode(null, path);
    setSidebarView('map');
  };

  const generate = async () => {
    if (activeRepoId == null) return;
    setLoading(true);
    setError(null);
    setPlan(null);
    try {
      const p = await api.generateOnboardingPlan(activeRepoId, {
        role,
        team: focus || undefined,
        experience_level: level,
      });
      setPlan(p);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to generate plan');
    } finally {
      setLoading(false);
    }
  };

  if (activeRepoId == null) {
    return (
      <div className="sidebar-view">
        <h3 className="sidebar-section-title">ONBOARDING</h3>
        <EmptyState
          icon="codicon-mortar-board"
          title="No repository"
          description="Select a repository to generate an AI ramp-up plan."
        />
      </div>
    );
  }

  return (
    <div className="sidebar-view onboarding-panel">
      <h3 className="sidebar-section-title">ONBOARDING PLAN</h3>
      <p className="sidebar-hint">AI-generated ramp-up from indexed architecture and ownership signals.</p>
      <div className="onboarding-panel__form">
        <label className="field-label">Role</label>
        <select className="field-input" value={role} onChange={(e) => setRole(e.target.value)}>
          <option value="backend">Backend</option>
          <option value="frontend">Frontend</option>
          <option value="fullstack">Full stack</option>
          <option value="platform">Platform</option>
        </select>
        <label className="field-label">Focus area (optional)</label>
        <input
          className="field-input"
          placeholder="e.g. payments, auth, GraphQL"
          value={focus}
          onChange={(e) => setFocus(e.target.value)}
        />
        <label className="field-label">Experience</label>
        <select className="field-input" value={level} onChange={(e) => setLevel(e.target.value)}>
          <option value="junior">Junior</option>
          <option value="mid">Mid</option>
          <option value="senior">Senior</option>
        </select>
        <button
          type="button"
          className="btn-primary btn-primary--block"
          onClick={() => void generate()}
          disabled={loading}
        >
          {loading ? 'Generating…' : 'Generate plan'}
        </button>
      </div>
      {error ? <p className="empty-state">{error}</p> : null}
      {loading ? <ViewSkeleton rows={5} /> : null}
      {plan && !loading ? (
        <div className="onboarding-plan">
          {plan.summary ? <p className="onboarding-plan__summary">{plan.summary}</p> : null}
          {(plan.steps ?? []).map((step) => (
            <section key={step.order} className="onboarding-plan__step">
              <div className="onboarding-plan__step-head">
                <span className="onboarding-plan__order">{step.order}</span>
                <strong>{step.title}</strong>
                {step.estimated_minutes > 0 ? (
                  <span className="muted">~{step.estimated_minutes}m</span>
                ) : null}
              </div>
              <p className="muted">{step.description}</p>
              {step.file_paths?.length ? (
                <ul className="onboarding-plan__files">
                  {step.file_paths.map((p) => (
                    <li key={p}>
                      <button type="button" className="linkish" onClick={() => openFile(p)}>
                        {basename(p)}
                      </button>
                    </li>
                  ))}
                </ul>
              ) : null}
            </section>
          ))}
        </div>
      ) : null}
    </div>
  );
}
