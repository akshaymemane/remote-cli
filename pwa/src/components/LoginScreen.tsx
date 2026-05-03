import { type FormEvent, useState } from 'react';

interface Props {
  onLogin: (token: string) => void;
}

const RELAY_URL = import.meta.env.VITE_RELAY_URL ?? window.location.origin;

export function LoginScreen({ onLogin }: Props) {
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const r = await fetch(`${RELAY_URL}/api/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ password }),
      });
      if (!r.ok) { setError('Invalid password'); return; }
      const { token } = await r.json() as { token: string };
      localStorage.setItem('relay_token', token);
      onLogin(token);
    } catch {
      setError('Cannot reach relay');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="login-screen">
      <div className="login-card">
        <h1>Remote CLI</h1>
        <p className="subtitle">Connect to your machines</p>
        <form onSubmit={submit}>
          <input
            type="password"
            placeholder="Admin password"
            value={password}
            onChange={e => setPassword(e.target.value)}
            autoFocus
          />
          {error && <p className="error">{error}</p>}
          <button type="submit" disabled={loading || !password}>
            {loading ? 'Signing in…' : 'Sign in'}
          </button>
        </form>
      </div>
    </div>
  );
}
