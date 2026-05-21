import { useState } from 'react';

import { api } from '../lib/api';
import { useStore } from '../store';
import type { OnboardingPlan } from '../types';

export function OnboardingPanel() {
  const activeRepoId = useStore((s) => s.activeRepoId);
  const setSelectedNode = useStore((s) => s.setSelectedNode);
  const [role, setRole] = useState('backend');
  const [team, setTeam] = useState('');
  const [level, setLevel] = useState('mid');
  const [plan, setPlan] = useState<OnboardingPlan | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const generate = async () => {
    if (activeRepoId == null) return;
    setLoading(true);
    setError(null);
    try {
      const p = await api.generateOnboardingPlan(activeRepoId, {
        role,
        team: team || undefined,
        experience_level: level,
      });
      setPlan(p);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to generate plan');
    } finally {
      setLoading(false);
    }
  };

  const openFile = (path: string) => {
    setSelectedNode(path, path);
  };

  if (activeRepoId == null) {
    return (
      <div className="sidebar-view">
        <h3 className="sidebar-section-title">ONBOARDING</h3>
        <p className="sidebar-hint">Select a repository to generate a ramp-up plan.</p>
      </div>
    );
  }

  return (
    <div className="sidebar-view onboarding-panel">
      <h3 className="sidebar-section-title">ONBOARDING PLAN</h3>
      <div className="onboarding-panel__form">
        <select className="field-input" value={role} onChange={(e) => setRole(e.target.value)}>
          <option value="backend">Backend</option>
          <option value="frontend">Frontend</option>
          <option value="fullstack">Full stack</option>
          <option value="platform">Platform</option>
        </select>
        <input
          className="field-input"
          placeholder="Team (optional)"
          value={team}
          onChange={(e) => setTeam(e.target.value)}
        />
        <select className="field-input" value={level} onChange={(e) => setLevel(e.target.value)}>
          <option value="junior">Junior</option>
          <option value="mid">Mid</option>
          <option value="senior">Senior</option>
        </select>
        <button type="button" className="btn-primary btn-primary--block" onClick={() => void generate()} disabled={loading}>
          {loading ? 'Generating…' : 'Generate plan'}
        </button>
      </div>
      {error ? <p className="empty-state">{error}</p> : null}
      {plan ? (
        <div>
          {(['week_1', 'week_2', 'week_3_to_4'] as const).map((wk) => {
            const block = plan[wk];
            if (!block) return null;
            return (
              <section key={wk} style={{ marginBottom: 12 }}>
                <strong>{wk.replaceAll('_', ' ').toUpperCase()}</strong>
                <p className="muted">{block.goal}</p>
                <div>Start here:</div>
                <ul>
                  {block.start_here?.map((f) => (
                    <li key={f.file_path}>
                      <button type="button" className="linkish" onClick={() => openFile(f.file_path)}>
                        {f.file_path}
                      </button>
                      <span className="muted"> — {f.reason}</span>
                    </li>
                  ))}
                </ul>
              </section>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}
