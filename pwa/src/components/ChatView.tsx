import { type FormEvent, useEffect, useRef, useState } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import type { Components } from 'react-markdown';
import { PrismLight as SyntaxHighlighter } from 'react-syntax-highlighter';
import { oneDark } from 'react-syntax-highlighter/dist/esm/styles/prism';
import bash from 'react-syntax-highlighter/dist/esm/languages/prism/bash';
import python from 'react-syntax-highlighter/dist/esm/languages/prism/python';
import typescript from 'react-syntax-highlighter/dist/esm/languages/prism/typescript';
import javascript from 'react-syntax-highlighter/dist/esm/languages/prism/javascript';
import go from 'react-syntax-highlighter/dist/esm/languages/prism/go';
import json from 'react-syntax-highlighter/dist/esm/languages/prism/json';
import type { ChatMessage, Device } from '../types';

SyntaxHighlighter.registerLanguage('bash', bash);
SyntaxHighlighter.registerLanguage('shell', bash);
SyntaxHighlighter.registerLanguage('sh', bash);
SyntaxHighlighter.registerLanguage('python', python);
SyntaxHighlighter.registerLanguage('typescript', typescript);
SyntaxHighlighter.registerLanguage('javascript', javascript);
SyntaxHighlighter.registerLanguage('go', go);
SyntaxHighlighter.registerLanguage('json', json);

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const mdComponents: Components = {
  code({ className, children }) {
    const lang = /language-(\w+)/.exec(className ?? '')?.[1];
    if (lang) {
      return (
        <SyntaxHighlighter
          style={oneDark as Record<string, React.CSSProperties>}
          language={lang}
          PreTag="div"
          customStyle={{ margin: '6px 0', borderRadius: '6px', fontSize: '12px' }}
        >
          {String(children).replace(/\n$/, '')}
        </SyntaxHighlighter>
      );
    }
    return <code className="inline-code">{children}</code>;
  },
};

interface Props {
  device: Device;
  sessionId: string | null;
  wsConnected: boolean;
  messages: ChatMessage[];
  thinking: boolean;
  onSend: (text: string) => void;
  onBack: () => void;
  onEndSession: () => void;
}

export function ChatView({
  device, sessionId, wsConnected, messages, thinking, onSend, onBack, onEndSession,
}: Props) {
  const [input, setInput] = useState('');
  const messagesRef = useRef<HTMLDivElement>(null);
  const [atBottom, setAtBottom] = useState(true);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // Scroll to bottom when new content arrives, only if already at bottom.
  useEffect(() => {
    if (!atBottom) return;
    const el = messagesRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [messages, thinking, atBottom]);

  function handleScroll() {
    const el = messagesRef.current;
    if (!el) return;
    setAtBottom(el.scrollHeight - el.scrollTop - el.clientHeight < 80);
  }

  function scrollToBottom() {
    const el = messagesRef.current;
    if (el) el.scrollTop = el.scrollHeight;
    setAtBottom(true);
  }

  function autoResize() {
    const el = textareaRef.current;
    if (!el) return;
    el.style.height = 'auto';
    el.style.height = `${Math.min(el.scrollHeight, 120)}px`;
  }

  function submit(e?: FormEvent) {
    e?.preventDefault();
    const text = input.trim();
    if (!text || !sessionId) return;
    setInput('');
    if (textareaRef.current) textareaRef.current.style.height = 'auto';
    onSend(text);
  }

  function onKeyDown(e: React.KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      submit();
    }
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

      <div className="messages" ref={messagesRef} onScroll={handleScroll}>
        {messages.length === 0 && !thinking && (
          <div className="empty-chat">
            {sessionId ? <p>Session started. Send a message.</p> : <p>Starting session…</p>}
          </div>
        )}
        {messages.map(msg => (
          <MessageBubble key={msg.id} msg={msg} />
        ))}
        {thinking && (
          <div className="bubble assistant thinking">
            <span /><span /><span />
          </div>
        )}
      </div>

      {!atBottom && (
        <button className="scroll-fab" onClick={scrollToBottom} title="Scroll to bottom">↓</button>
      )}

      <form className="input-bar" onSubmit={submit}>
        <textarea
          ref={textareaRef}
          rows={1}
          value={input}
          onChange={e => { setInput(e.target.value); autoResize(); }}
          onKeyDown={onKeyDown}
          placeholder={
            !wsConnected ? 'Reconnecting…'
            : sessionId ? 'Message… (Shift+↵ for newline)'
            : 'Waiting for session…'
          }
          disabled={!sessionId || !wsConnected}
        />
        <button type="submit" disabled={!input.trim() || !sessionId || !wsConnected}>⬆</button>
      </form>
    </div>
  );
}

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);
  function copy() {
    navigator.clipboard.writeText(text).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    });
  }
  return (
    <button className="copy-btn" onClick={copy} title="Copy">
      {copied ? '✓' : '⧉'}
    </button>
  );
}

function MessageBubble({ msg }: { msg: ChatMessage }) {
  if (msg.role === 'tool') return <ToolCard msg={msg} />;

  if (msg.role === 'user') {
    return (
      <div className="bubble user">
        <pre>{msg.text}</pre>
      </div>
    );
  }

  return (
    <div className="bubble-outer">
      <div className="bubble assistant md">
        <ReactMarkdown remarkPlugins={[remarkGfm]} components={mdComponents}>
          {msg.text}
        </ReactMarkdown>
      </div>
      {msg.text && (
        <div className="bubble-actions">
          <CopyButton text={msg.text} />
        </div>
      )}
    </div>
  );
}

function ToolCard({ msg }: { msg: ChatMessage }) {
  const [expanded, setExpanded] = useState(true);
  const prevPending = useRef(msg.pending);

  // Auto-collapse when tool finishes (unless user already toggled).
  const userToggled = useRef(false);
  useEffect(() => {
    if (prevPending.current && !msg.pending && !userToggled.current) {
      setExpanded(false);
    }
    prevPending.current = msg.pending;
  }, [msg.pending]);

  function toggle() {
    userToggled.current = true;
    setExpanded(e => !e);
  }

  return (
    <div className="tool-card">
      <button className="tool-header" onClick={toggle}>
        <span className="tool-name">⚙ {msg.toolName}</span>
        {msg.pending
          ? <span className="tool-status running">running</span>
          : <span className="tool-status done">done</span>
        }
        <span className="tool-chevron">{expanded ? '▴' : '▾'}</span>
      </button>
      {expanded && (
        <div className="tool-body">
          {msg.toolInput && (
            <pre className="tool-input">{JSON.stringify(msg.toolInput, null, 2)}</pre>
          )}
          {msg.toolResult !== undefined && (
            <>
              <div className="tool-result-header">
                <span className="tool-result-label">output</span>
                <CopyButton text={msg.toolResult} />
              </div>
              <pre className="tool-result">{msg.toolResult}</pre>
            </>
          )}
        </div>
      )}
    </div>
  );
}
