import { useState } from 'react';
import type { Device } from '../types';

interface Props {
  devices: Device[];
  token: string;
  relayUrl: string;
  onSelect: (deviceId: string) => void;
  onAddDevice: () => void;
  onDevicesChanged: () => void;
}

export function DeviceList({ devices, token, relayUrl, onSelect, onAddDevice, onDevicesChanged }: Props) {
  const [actionDevice, setActionDevice] = useState<Device | null>(null);
  const [renaming, setRenaming] = useState(false);
  const [newName, setNewName] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  function openActions(e: React.MouseEvent, device: Device) {
    e.stopPropagation();
    setActionDevice(device);
    setRenaming(false);
    setNewName(device.name);
    setError('');
  }

  function closeSheet() {
    setActionDevice(null);
    setRenaming(false);
    setError('');
  }

  async function doRename() {
    if (!actionDevice || !newName.trim()) return;
    setBusy(true);
    setError('');
    try {
      const r = await fetch(`${relayUrl}/api/devices/${actionDevice.id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ name: newName.trim() }),
      });
      if (!r.ok) { setError('Rename failed'); return; }
      onDevicesChanged();
      closeSheet();
    } catch {
      setError('Cannot reach relay');
    } finally {
      setBusy(false);
    }
  }

  async function doDelete() {
    if (!actionDevice) return;
    setBusy(true);
    setError('');
    try {
      const r = await fetch(`${relayUrl}/api/devices/${actionDevice.id}`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!r.ok) { setError('Delete failed'); return; }
      onDevicesChanged();
      closeSheet();
    } catch {
      setError('Cannot reach relay');
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="device-list">
      <header className="topbar">
        <h2>Devices</h2>
        <button className="icon-btn" onClick={onAddDevice} title="Add device">＋</button>
      </header>

      {devices.length === 0 && (
        <div className="empty-state">
          <p>No devices paired yet.</p>
          <button onClick={onAddDevice}>Add your first device</button>
        </div>
      )}

      <ul>
        {devices.map(d => (
          <li key={d.id} onClick={() => onSelect(d.id)}>
            <div className={`status-dot ${d.status}`} />
            <div className="device-info">
              <span className="device-name">{d.name}</span>
              <span className="device-meta">
                {d.status === 'online'
                  ? 'Online'
                  : d.status === 'busy'
                  ? 'Busy'
                  : d.last_seen
                  ? `Last seen ${timeAgo(d.last_seen)}`
                  : 'Offline'}
              </span>
            </div>
            <button
              className="icon-btn device-more-btn"
              title="Device options"
              onClick={e => openActions(e, d)}
            >⋯</button>
          </li>
        ))}
      </ul>

      {actionDevice && (
        <div className="dialog-overlay" onClick={closeSheet}>
          <div className="dialog" onClick={e => e.stopPropagation()}>
            <header>
              <h3>{renaming ? 'Rename device' : actionDevice.name}</h3>
              <button className="icon-btn" onClick={closeSheet}>✕</button>
            </header>

            {renaming ? (
              <>
                <input
                  autoFocus
                  value={newName}
                  onChange={e => setNewName(e.target.value)}
                  onKeyDown={e => e.key === 'Enter' && doRename()}
                  maxLength={40}
                  placeholder="Device name"
                />
                {error && <p className="error">{error}</p>}
                <div className="sheet-actions">
                  <button onClick={() => setRenaming(false)} className="text-btn">Cancel</button>
                  <button onClick={doRename} disabled={busy || !newName.trim()}>
                    {busy ? '…' : 'Save'}
                  </button>
                </div>
              </>
            ) : (
              <>
                <div className="sheet-actions col">
                  <button className="sheet-btn" onClick={() => setRenaming(true)}>✏ Rename</button>
                  <button className="sheet-btn danger" onClick={doDelete} disabled={busy}>
                    {busy ? '…' : '🗑 Delete device'}
                  </button>
                </div>
                {error && <p className="error">{error}</p>}
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

function timeAgo(ts: number): string {
  const secs = Math.floor((Date.now() / 1000) - ts);
  if (secs < 60) return `${secs}s ago`;
  if (secs < 3600) return `${Math.floor(secs / 60)}m ago`;
  if (secs < 86400) return `${Math.floor(secs / 3600)}h ago`;
  return `${Math.floor(secs / 86400)}d ago`;
}
