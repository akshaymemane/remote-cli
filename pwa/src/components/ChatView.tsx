import { type FormEvent, useEffect, useRef, useState } from 'react';
import type { ChatMessage, Device } from '../types';

interface Props {
  device: Device;
  sessionId: string | null;
  wsConnected: boolean;
  messages: ChatMessage[];
  onSend: (text: string) => void;
  onBack: () => void;
  onEndSession: () => void;
}

export function ChatView({
  device, sessionId, wsConnected, messages, onSend, onBack, onEndSession,
}: Props) {
  const [input, setInput] = useState('');
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  function submit(e: FormEvent) {
    e.preventDefault();
    const text = input.trim();
    if (!text || !sessionId) return;
    setInput('');
    onSend(text);
  }

  return (
    <div className="chat-view">
      <header className="topbar">
        <button className="back-btn" onClick={onBack}>‹</button>
        <div className="topbar-center">
          <span className="device-name">{device.name}</span>
          <span className={`status-label ${device.status}`}>
            {device.status === 'busy' ? 'Busy' : device.status === 'online' ? 'Online' : 'Offline'}
          </span>
        </div>
        {sessionId && (
          <button className="text-btn danger" onClick={onEndSession}>End</button>
        )}
      </header>

      <div className="messages">
        {messages.length === 0 && (
          <div className="empty-chat">
            {sessionId
              ? <p>Session started. Send a message.</p>
              : <p>Starting session…</p>}
          </div>
        )}
        {messages.map(msg => (
          <MessageBubble
            key={msg.id}
            msg={msg}
          />
        ))}
        <div ref={bottomRef} />
      </div>

      <form className="input-bar" onSubmit={submit}>
        <input
          value={input}
          onChange={e => setInput(e.target.value)}
          placeholder={!wsConnected ? 'Reconnecting…' : sessionId ? 'Message…' : 'Waiting for session…'}
          disabled={!sessionId || !wsConnected}
        />
        <button type="submit" disabled={!input.trim() || !sessionId || !wsConnected}>⬆</button>
      </form>
    </div>
  );
}

function MessageBubble({ msg }: { msg: ChatMessage }) {
  if (msg.role === 'tool') {
    return (
      <div className="tool-card">
        <div className="tool-header">
          <span className="tool-name">⚙ {msg.toolName}</span>
          {msg.pending && <span className="tool-status running">running</span>}
          {!msg.pending && msg.toolResult !== undefined && (
            <span className="tool-status done">done</span>
          )}
        </div>
        {msg.toolInput && (
          <pre className="tool-input">
            {JSON.stringify(msg.toolInput, null, 2)}
          </pre>
        )}
        {msg.toolResult !== undefined && (
          <pre className="tool-result">{msg.toolResult}</pre>
        )}
      </div>
    );
  }

  return (
    <div className={`bubble ${msg.role}`}>
      <pre>{msg.text}</pre>
    </div>
  );
}
