import { useEffect, useState } from 'react';
import { getSettings, updateSettings, testConnection } from '../hooks/useTauri';
import type { Settings } from '../types';
import InitialSyncDialog from '../components/InitialSyncDialog';
import { $showShortcuts } from '../store';

export default function SettingsView() {
  const [settings, setSettings] = useState<Settings | null>(null);
  const [serverUrl, setServerUrl] = useState('');
  const [apiKey, setApiKey] = useState('');
  const [syncEnabled, setSyncEnabled] = useState(false);
  const [showApiKey, setShowApiKey] = useState(false);
  const [saving, setSaving] = useState(false);
  const [testStatus, setTestStatus] = useState<'idle' | 'testing' | 'success' | 'error'>('idle');
  const [dirty, setDirty] = useState(false);
  const [showInitialSync, setShowInitialSync] = useState(false);

  useEffect(() => {
    getSettings().then(s => {
      setSettings(s);
      setServerUrl(s.serverUrl);
      setApiKey(s.apiKey);
      setSyncEnabled(s.syncEnabled);
    });
  }, []);

  const handleSave = async () => {
    setSaving(true);
    const updated = await updateSettings({ serverUrl, apiKey, syncEnabled });
    setSettings(updated);
    setDirty(false);
    setSaving(false);
  };

  const handleTest = async () => {
    setTestStatus('testing');
    try {
      const ok = await testConnection();
      setTestStatus(ok ? 'success' : 'error');
    } catch {
      setTestStatus('error');
    }
  };

  if (!settings) {
    return <div style={{ padding: 'var(--sp-5)', color: 'var(--ink-tertiary)' }}>Loading settings...</div>;
  }

  return (
    <div style={{ padding: 'var(--sp-5)', maxWidth: 480 }}>
      <h2 style={{ fontSize: 'var(--text-lg)', fontWeight: 700, color: 'var(--ink-primary)', marginBottom: 'var(--sp-5)' }}>
        Settings
      </h2>

      {/* Sync Section */}
      <div style={{ marginBottom: 'var(--sp-6)' }}>
        <div className="detail-field-label" style={{ marginBottom: 'var(--sp-3)' }}>Sync</div>

        {/* Sync toggle */}
        <label style={{ display: 'flex', alignItems: 'center', gap: 'var(--sp-3)', cursor: 'pointer', marginBottom: 'var(--sp-4)' }}>
          <input
            type="checkbox"
            checked={syncEnabled}
            onChange={e => { setSyncEnabled(e.target.checked); setDirty(true); }}
            style={{ width: 16, height: 16, accentColor: 'var(--accent)' }}
          />
          <span style={{ fontSize: 'var(--text-sm)', color: 'var(--ink-primary)' }}>
            Enable sync with atask server
          </span>
        </label>

        {/* Server URL */}
        <div style={{ marginBottom: 'var(--sp-3)' }}>
          <label style={{ display: 'block', fontSize: 'var(--text-xs)', fontWeight: 700, color: 'var(--ink-tertiary)', textTransform: 'uppercase', letterSpacing: '0.5px', marginBottom: 'var(--sp-1)' }}>
            Server URL
          </label>
          <input
            type="url"
            value={serverUrl}
            onChange={e => { setServerUrl(e.target.value); setDirty(true); }}
            placeholder="https://api.atask.app"
            disabled={!syncEnabled}
            style={{
              width: '100%',
              padding: 'var(--sp-2) var(--sp-3)',
              border: '1px solid var(--border-strong)',
              borderRadius: 'var(--radius-md)',
              background: syncEnabled ? 'var(--canvas-elevated)' : 'var(--canvas-sunken)',
              color: 'var(--ink-primary)',
              fontFamily: 'inherit',
              fontSize: 'var(--text-sm)',
              outline: 'none',
              boxSizing: 'border-box',
            }}
          />
        </div>

        {/* API Key */}
        <div style={{ marginBottom: 'var(--sp-3)' }}>
          <label style={{ display: 'block', fontSize: 'var(--text-xs)', fontWeight: 700, color: 'var(--ink-tertiary)', textTransform: 'uppercase', letterSpacing: '0.5px', marginBottom: 'var(--sp-1)' }}>
            API Key
          </label>
          <div style={{ display: 'flex', gap: 'var(--sp-2)' }}>
            <input
              type={showApiKey ? 'text' : 'password'}
              value={apiKey}
              onChange={e => { setApiKey(e.target.value); setDirty(true); }}
              placeholder="ak_..."
              disabled={!syncEnabled}
              style={{
                flex: 1,
                padding: 'var(--sp-2) var(--sp-3)',
                border: '1px solid var(--border-strong)',
                borderRadius: 'var(--radius-md)',
                background: syncEnabled ? 'var(--canvas-elevated)' : 'var(--canvas-sunken)',
                color: 'var(--ink-primary)',
                fontFamily: "'SF Mono', 'Menlo', monospace",
                fontSize: 'var(--text-sm)',
                outline: 'none',
              }}
            />
            <button
              onClick={() => setShowApiKey(!showApiKey)}
              className="bulk-bar-btn"
              style={{ padding: 'var(--sp-2) var(--sp-3)', border: '1px solid var(--border)' }}
            >
              {showApiKey ? 'Hide' : 'Show'}
            </button>
          </div>
        </div>

        {/* Action buttons */}
        <div style={{ display: 'flex', gap: 'var(--sp-2)', marginTop: 'var(--sp-4)', alignItems: 'center' }}>
          <button
            className="btn btn-primary"
            onClick={handleSave}
            disabled={!dirty || saving}
            style={{
              padding: 'var(--sp-2) var(--sp-4)',
              background: dirty ? 'var(--accent)' : 'var(--ink-quaternary)',
              color: 'var(--ink-on-accent)',
              border: 'none',
              borderRadius: 'var(--radius-md)',
              cursor: dirty ? 'pointer' : 'default',
              fontFamily: 'inherit',
              fontSize: 'var(--text-sm)',
              fontWeight: 600,
            }}
          >
            {saving ? 'Saving...' : 'Save'}
          </button>
          <button
            className="bulk-bar-btn"
            onClick={handleTest}
            disabled={!syncEnabled || !serverUrl}
            style={{ padding: 'var(--sp-2) var(--sp-3)', border: '1px solid var(--border)' }}
          >
            {testStatus === 'testing' ? 'Testing...' : 'Test Connection'}
          </button>
          {testStatus === 'success' && (
            <span style={{ fontSize: 'var(--text-xs)', color: 'var(--success)', alignSelf: 'center' }}>Connected</span>
          )}
          {testStatus === 'error' && (
            <span style={{ fontSize: 'var(--text-xs)', color: 'var(--deadline-red)', alignSelf: 'center' }}>Not connected</span>
          )}
        </div>

        {/* Initial sync */}
        {syncEnabled && (
          <div style={{ marginTop: 'var(--sp-3)' }}>
            <button
              className="bulk-bar-btn"
              onClick={() => setShowInitialSync(true)}
              style={{ padding: 'var(--sp-2) var(--sp-3)', border: '1px solid var(--border)' }}
            >
              Run Initial Sync
            </button>
          </div>
        )}
      </div>

      {/* About Section */}
      <div style={{ borderTop: '1px solid var(--separator)', paddingTop: 'var(--sp-4)' }}>
        <div className="detail-field-label" style={{ marginBottom: 'var(--sp-2)' }}>About</div>
        <div style={{ fontSize: 'var(--text-sm)', color: 'var(--ink-secondary)', lineHeight: 'var(--leading-relaxed)' }}>
          <div>atask v4 — AI-first task manager</div>
          <div style={{ color: 'var(--ink-tertiary)', marginTop: 'var(--sp-1)' }}>Local-first, Tauri + React</div>
        </div>
        <div style={{ marginTop: 'var(--sp-3)' }}>
          <button
            className="bulk-bar-btn"
            onClick={() => $showShortcuts.set(true)}
            style={{ padding: 'var(--sp-2) var(--sp-3)', border: '1px solid var(--border)' }}
          >
            Keyboard Shortcuts
          </button>
        </div>
      </div>

      <InitialSyncDialog open={showInitialSync} onClose={() => setShowInitialSync(false)} />
    </div>
  );
}
