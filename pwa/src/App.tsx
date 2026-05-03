import { useState } from 'react';
import { LoginScreen } from './components/LoginScreen';
import { DeviceList } from './components/DeviceList';
import { ChatView } from './components/ChatView';
import { PairDialog } from './components/PairDialog';
import { useRelay } from './hooks/useRelay';
import './App.css';

const RELAY_URL = (import.meta.env.VITE_RELAY_URL as string | undefined) ?? window.location.origin;

type Screen = 'devices' | 'chat';

export default function App() {
  const [token, setToken] = useState<string | null>(
    () => localStorage.getItem('relay_token')
  );
  const [screen, setScreen] = useState<Screen>('devices');
  const [showPair, setShowPair] = useState(false);

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
    return <LoginScreen onLogin={t => setToken(t)} />;
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
          onSend={relay.sendMessage}
          onBack={handleBack}
          onEndSession={handleBack}
        />
      )}

      {showPair && (
        <PairDialog
          token={token}
          onPaired={() => {
            setShowPair(false);
            refreshDevices();
          }}
          onClose={() => setShowPair(false)}
        />
      )}

      {screen === 'devices' && (
        <button className="logout-btn" onClick={handleLogout} title="Sign out">⏏</button>
      )}
    </>
  );
}
