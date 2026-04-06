import { useEffect, useState } from 'react';
import { getSettings, updateSettings, testConnection } from '../hooks/useTauri';
import type { Settings } from '../types';
import InitialSyncDialog from '../components/InitialSyncDialog';
import { $showShortcuts } from '../store';
import { Button, Field } from '../ui';

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
    return <div className="settings-loading">Loading settings...</div>;
  }

  return (
    <div className="settings-view">
      <h2 className="settings-title">Settings</h2>

      <div className="settings-section">
        <div className="detail-field-label settings-section-label">Sync</div>

        <label className="settings-toggle">
          <input
            type="checkbox"
            checked={syncEnabled}
            onChange={e => { setSyncEnabled(e.target.checked); setDirty(true); }}
            className="settings-checkbox"
          />
          <span className="settings-toggle-label">
            Enable sync with atask server
          </span>
        </label>

        <div className="settings-field-group">
          <Field
            label="Server URL"
            type="url"
            value={serverUrl}
            onChange={e => { setServerUrl(e.target.value); setDirty(true); }}
            placeholder="https://api.atask.app"
            disabled={!syncEnabled}
            className={!syncEnabled ? 'settings-field-disabled' : undefined}
          />
        </div>

        <div className="settings-field-group">
          <label className="ui-field-block">
            <span className="ui-field-label">API Key</span>
            <div className="settings-inline-field">
              <input
                className={[
                  'ui-field',
                  'settings-api-key',
                  !syncEnabled ? 'settings-field-disabled' : '',
                ].filter(Boolean).join(' ')}
              type={showApiKey ? 'text' : 'password'}
              value={apiKey}
              onChange={e => { setApiKey(e.target.value); setDirty(true); }}
              placeholder="ak_..."
              disabled={!syncEnabled}
              />
              <Button
                variant="secondary"
                size="sm"
                onClick={() => setShowApiKey(!showApiKey)}
              >
                {showApiKey ? 'Hide' : 'Show'}
              </Button>
            </div>
          </label>
        </div>

        <div className="settings-actions">
          <Button
            variant="primary"
            onClick={handleSave}
            disabled={!dirty || saving}
            className={!dirty ? 'settings-save-disabled' : undefined}
          >
            {saving ? 'Saving...' : 'Save'}
          </Button>
          <Button
            variant="secondary"
            size="sm"
            onClick={handleTest}
            disabled={!syncEnabled || !serverUrl}
          >
            {testStatus === 'testing' ? 'Testing...' : 'Test Connection'}
          </Button>
          {testStatus === 'success' && (
            <span className="settings-status settings-status-success">Connected</span>
          )}
          {testStatus === 'error' && (
            <span className="settings-status settings-status-error">Not connected</span>
          )}
        </div>

        {syncEnabled && (
          <div className="settings-sync-row">
            <Button
              variant="secondary"
              size="sm"
              onClick={() => setShowInitialSync(true)}
            >
              Run Initial Sync
            </Button>
          </div>
        )}
      </div>

      <div className="settings-about">
        <div className="detail-field-label settings-about-label">About</div>
        <div className="settings-about-copy">
          <div>atask v4 — AI-first task manager</div>
          <div className="settings-about-subtle">Local-first, Tauri + React</div>
        </div>
        <div className="settings-about-actions">
          <Button
            variant="secondary"
            size="sm"
            onClick={() => $showShortcuts.set(true)}
          >
            Keyboard Shortcuts
          </Button>
        </div>
      </div>

      <InitialSyncDialog open={showInitialSync} onClose={() => setShowInitialSync(false)} />
    </div>
  );
}
