import type { ThemeMode } from '../../lib/theme';
import { useStore } from '../../store';
import { AuthSettings } from '../shell/AuthSettings';

const THEMES: Array<{ id: ThemeMode; label: string }> = [
  { id: 'dark', label: 'Dark' },
  { id: 'light', label: 'Light' },
  { id: 'system', label: 'System' },
];

export function SettingsView() {
  const theme = useStore((s) => s.theme);
  const setTheme = useStore((s) => s.setTheme);
  const apiStatus = useStore((s) => s.apiStatus);
  const setTourStep = useStore((s) => s.setTourStep);

  return (
    <div className="sidebar-view">
      <h3 className="sidebar-section-title">SETTINGS</h3>

      <p className="sidebar-hint">Appearance</p>
      <div className="theme-picker" role="radiogroup" aria-label="Theme">
        {THEMES.map((t) => (
          <button
            key={t.id}
            type="button"
            role="radio"
            aria-checked={theme === t.id}
            className={`theme-picker__btn ${theme === t.id ? 'theme-picker__btn--active' : ''}`}
            onClick={() => setTheme(t.id)}
          >
            {t.label}
          </button>
        ))}
      </div>

      <p className="sidebar-hint" style={{ marginTop: 16 }}>
        API connection:{' '}
        <strong>
          {apiStatus === 'online'
            ? 'Connected'
            : apiStatus === 'degraded'
              ? 'Degraded'
              : apiStatus === 'checking'
                ? 'Checking…'
                : 'Offline'}
        </strong>
      </p>

      <p className="sidebar-hint">
        When <code>AUTH_DISABLED=false</code>, configure a JWT below.
      </p>
      <AuthSettings />

      <button type="button" className="btn-secondary btn-primary--block" onClick={() => setTourStep(0)}>
        Replay product tour
      </button>
    </div>
  );
}
