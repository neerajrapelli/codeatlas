import { useState } from 'react';

import { api } from '../../lib/api';
import { clearAuthToken, getAuthToken, setAuthToken } from '../../lib/authToken';
import { useStore } from '../../store';

export function AuthSettings() {
  const pushToast = useStore((s) => s.pushToast);
  const [token, setToken] = useState(() => getAuthToken() ?? '');
  const [subject, setSubject] = useState('dev-user');
  const [tenant, setTenant] = useState('default');
  const [bootstrap, setBootstrap] = useState('');

  const save = () => {
    if (token.trim()) {
      setAuthToken(token.trim());
      pushToast('API token saved', 'success');
    } else {
      clearAuthToken();
      pushToast('API token cleared', 'info');
    }
  };

  const mint = async () => {
    if (!bootstrap.trim()) {
      pushToast('Enter AUTH_BOOTSTRAP_SECRET to mint a token', 'error');
      return;
    }
    try {
      const res = await api.issueToken({ subject, tenant_id: tenant }, bootstrap.trim());
      setToken(res.token);
      setAuthToken(res.token);
      pushToast('Dev token issued and saved', 'success');
    } catch (e) {
      pushToast(e instanceof Error ? e.message : 'Token request failed', 'error');
    }
  };

  return (
    <div className="auth-settings">
      <label className="field-label" htmlFor="auth-token">
        Bearer token
      </label>
      <input
        id="auth-token"
        className="field-input"
        type="password"
        value={token}
        onChange={(e) => setToken(e.target.value)}
        placeholder="Paste JWT"
      />
      <button type="button" className="btn-secondary btn-primary--block" onClick={save}>
        Save token
      </button>

      <p className="auth-settings__divider">Dev token mint</p>
      <label className="field-label" htmlFor="auth-subject">
        Subject
      </label>
      <input
        id="auth-subject"
        className="field-input"
        value={subject}
        onChange={(e) => setSubject(e.target.value)}
      />
      <label className="field-label" htmlFor="auth-tenant">
        Tenant
      </label>
      <input
        id="auth-tenant"
        className="field-input"
        value={tenant}
        onChange={(e) => setTenant(e.target.value)}
      />
      <label className="field-label" htmlFor="auth-bootstrap">
        Bootstrap secret
      </label>
      <input
        id="auth-bootstrap"
        className="field-input"
        type="password"
        value={bootstrap}
        onChange={(e) => setBootstrap(e.target.value)}
        placeholder="AUTH_BOOTSTRAP_SECRET"
      />
      <button type="button" className="btn-primary btn-primary--block" onClick={() => void mint()}>
        Mint dev token
      </button>
    </div>
  );
}
