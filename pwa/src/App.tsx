import { useState } from 'react';
import { LoginScreen } from './components/LoginScreen';
import { DeviceList } from './components/DeviceList';
import { ChatView } from './components/ChatView';
import { PairDialog } from './components/PairDialog';
import { useRelay } from './hooks/useRelay';
import './App.css';

const RELAY_URL = (import.meta.env.VITE_RELAY_URL as string | undefined) ?? window.location.origin;

type Screen = 'devices' | 'chat';

function readPairCode(): string | null {
  const code = new URLSearchParams(window.location.search).get('pair');
  if (code && /^\d{6}$/.test(code)) {
    // Remove ?pair= from the URL bar without reloading.
    window.history.replaceState({}, '', window.location.pathname);
    return code;
  }
  return null;
}

export default function App() {
  const [token, setToken] = useState<string | null>(
    () => localStorage.getItem('relay_token')
  );
  const [screen, setScreen] = useState<Screen>('devices');
  const [pendingPairCode, setPendingPairCode] = useState<string | null>(readPairCode);
  // Auto-open pair dialog if a code arrived via QR and the user is already logged in.
  const [showPair, setShowPair] = useState(() => !!(localStorage.getItem('relay_token') && pendingPairCode));

  const relay = useRelay(token);

  function handleLogout() {
    localStorage.removeItem('relay_token');
    setToken(null);
  }

  function openDevice(deviceId: string) {
    relay.startSession(deviceId);
    setScreen('chat');
  }

  function handleBack() {
    relay.endSession();
    setScreen('devices');
  }

  function refreshDevices() {
    relay.send({ type: 'device.list' });
  }

  if (!token) {
    return <LoginScreen onLogin={t => { setToken(t); if (pendingPairCode) setShowPair(true); }} />;
  }

  const activeDevice = relay.devices.find(d => d.id === relay.activeDeviceId);

  return (
    <>
      {relay.wsStatus !== 'connected' && (
        <div className="status-banner">
          {relay.wsStatus === 'connecting' ? 'Connecting…' : 'Reconnecting…'}
        </div>
      )}

      {screen === 'devices' || !activeDevice ? (
        <DeviceList
          devices={relay.devices}
          token={token}
          relayUrl={RELAY_URL}
          onSelect={openDevice}
          onAddDevice={() => setShowPair(true)}
          onDevicesChanged={refreshDevices}
        />
      ) : (
        <ChatView
          device={activeDevice}
          sessionId={relay.sessionId}
          wsConnected={relay.wsStatus === 'connected'}
          messages={relay.messages}
          thinking={relay.thinking}
          onSend={relay.sendMessage}
          onBack={handleBack}
          onEndSession={handleBack}
        />
      )}

      {showPair && (
        <PairDialog
          token={token}
          initialCode={pendingPairCode ?? undefined}
          onPaired={() => {
            setShowPair(false);
            setPendingPairCode(null);
            refreshDevices();
          }}
          onClose={() => {
            setShowPair(false);
            setPendingPairCode(null);
          }}
        />
      )}

      {screen === 'devices' && (
        <button className="logout-btn" onClick={handleLogout} title="Sign out">⏏</button>
      )}
    </>
  );
}
