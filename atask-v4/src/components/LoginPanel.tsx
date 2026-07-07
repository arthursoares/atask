import { useState } from 'react';
import type { FormEvent } from 'react';
import { login } from '../hooks/useTauri';
import { $authState } from '../store';
import type { AuthState } from '../store';
import { Button, Field } from '../ui';

interface LoginPanelProps {
  /** Server URL to authenticate against — owned by the parent (SettingsView),
   * since it's shared with the legacy API-key sync config. */
  serverUrl: string;
  onSuccess?: (state: AuthState) => void;
}

export default function LoginPanel({ serverUrl, onSuccess }: LoginPanelProps) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const canSubmit = serverUrl.trim().length > 0 && email.trim().length > 0 && password.length > 0;

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (!canSubmit || loading) return;
    setLoading(true);
    setError('');
    try {
      const state = await login(serverUrl.trim(), email.trim(), password);
      $authState.set(state);
      onSuccess?.(state);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="login-panel">
      <div className="settings-field-group">
        <Field
          label="Email"
          type="email"
          autoComplete="username"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
        />
      </div>
      <div className="settings-field-group">
        <Field
          label="Password"
          type="password"
          autoComplete="current-password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
      </div>
      {error && <div className="login-panel-error">{error}</div>}
      <div className="login-panel-actions">
        <Button variant="primary" type="submit" disabled={!canSubmit || loading}>
          {loading ? 'Signing in...' : 'Sign In'}
        </Button>
      </div>
    </form>
  );
}
