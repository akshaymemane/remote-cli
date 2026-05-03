import { type FormEvent, useEffect, useRef, useState } from 'react';
import { Html5Qrcode } from 'html5-qrcode';

interface Props {
  token: string;
  initialCode?: string;
  onPaired: () => void;
  onClose: () => void;
}

const RELAY_URL = import.meta.env.VITE_RELAY_URL ?? window.location.origin;

export function PairDialog({ token, initialCode, onPaired, onClose }: Props) {
  const [code, setCode] = useState(initialCode ?? '');
  const [deviceName, setDeviceName] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [scanning, setScanning] = useState(false);
  const scannerRef = useRef<Html5Qrcode | null>(null);
  const scanDivId = 'qr-scan-region';

  async function redeem(pairCode: string) {
    setLoading(true);
    setError('');
    try {
      const r = await fetch(`${RELAY_URL}/api/pair/redeem`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ code: pairCode, device_name: deviceName || undefined }),
      });
      if (r.status === 410) { setError('Code expired — re-run pair on the machine'); return; }
      if (!r.ok) { setError('Pairing failed'); return; }
      onPaired();
    } catch {
      setError('Cannot reach relay');
    } finally {
      setLoading(false);
    }
  }

  function submitCode(e: FormEvent) {
    e.preventDefault();
    if (code.length === 6) redeem(code);
  }

  async function startScan() {
    setScanning(true);
    await new Promise(r => setTimeout(r, 100)); // allow DOM to render
    const scanner = new Html5Qrcode(scanDivId);
    scannerRef.current = scanner;
    scanner.start(
      { facingMode: 'environment' },
      { fps: 10, qrbox: 250 },
      (decodedText) => {
        // URL form: https://relay/?pair=CODE
        const match = decodedText.match(/[?&]pair=(\d{6})/);
        if (match) {
          scanner.stop().catch(() => {});
          setScanning(false);
          redeem(match[1]);
        }
      },
      () => {}
    ).catch(err => {
      setError(`Camera error: ${err}`);
      setScanning(false);
    });
  }

  function stopScan() {
    scannerRef.current?.stop().catch(() => {});
    setScanning(false);
  }

  useEffect(() => () => { scannerRef.current?.stop().catch(() => {}); }, []);

  return (
    <div className="dialog-overlay">
      <div className="dialog">
        <header>
          <h3>Add device</h3>
          <button className="icon-btn" onClick={onClose}>✕</button>
        </header>

        <p className="hint">
          Run <code>remote-cli pair --relay &lt;url&gt;</code> on the machine, then
          scan the QR code or enter the 6-digit code below.
        </p>

        <input
          placeholder="Device name (optional)"
          value={deviceName}
          onChange={e => setDeviceName(e.target.value)}
          maxLength={40}
          autoFocus={!!initialCode}
        />

        {!scanning ? (
          <>
            <button className="scan-btn" onClick={startScan}>📷 Scan QR code</button>
            <p className="or">— or enter code manually —</p>
            <form onSubmit={submitCode} className="code-form">
              <input
                className="code-input"
                placeholder="000000"
                value={code}
                onChange={e => setCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                maxLength={6}
                inputMode="numeric"
                pattern="\d{6}"
              />
              <button type="submit" disabled={code.length !== 6 || loading}>
                {loading ? '…' : 'Pair'}
              </button>
            </form>
          </>
        ) : (
          <div className="scan-area">
            <div id={scanDivId} />
            <button onClick={stopScan} className="text-btn">Cancel scan</button>
          </div>
        )}

        {error && <p className="error">{error}</p>}
      </div>
    </div>
  );
}
