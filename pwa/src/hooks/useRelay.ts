import { useCallback, useEffect, useRef, useState } from 'react';
import type { Device, ChatMessage, RelayMsg } from '../types';

const RELAY_URL = (import.meta.env.VITE_RELAY_URL as string | undefined) ?? window.location.origin;
const WS_URL = RELAY_URL.replace(/^http/, 'ws');

export type WsStatus = 'connecting' | 'connected' | 'disconnected';

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function cast<T>(v: unknown): T { return v as T; }

function uuid(): string {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) return crypto.randomUUID();
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, c => {
    const r = Math.random() * 16 | 0;
    return (c === 'x' ? r : (r & 0x3 | 0x8)).toString(16);
  });
}

export function useRelay(token: string | null) {
  const wsRef = useRef<WebSocket | null>(null);
  const [wsStatus, setWsStatus] = useState<WsStatus>('disconnected');
  const [devices, setDevices] = useState<Device[]>([]);
  const [activeDeviceId, setActiveDeviceId] = useState<string | null>(null);
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const streamingIdRef = useRef<string | null>(null);

  const send = useCallback((msg: RelayMsg) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(msg));
    }
  }, []);

  const handleMsg = useCallback((raw: string) => {
    let msg: RelayMsg;
    try { msg = JSON.parse(raw) as RelayMsg; } catch { return; }

    switch (msg.type) {
      case 'device.list': {
        const devs = (msg.devices as Device[]) ?? [];
        setDevices(devs);
        break;
      }
      case 'device.update': {
        const m = cast<{ device_id: string; status: string; last_seen?: number; name?: string }>(msg);
        if (m.status === 'deleted') {
          setDevices(prev => prev.filter(d => d.id !== m.device_id));
          break;
        }
        setDevices(prev =>
          prev.map(d =>
            d.id === m.device_id
              ? { ...d, status: m.status as Device['status'], last_seen: m.last_seen, ...(m.name ? { name: m.name } : {}) }
              : d
          )
        );
        break;
      }
      case 'session.state': {
        const m = cast<{ session_id: string }>(msg);
        setSessionId(m.session_id);
        break;
      }
      case 'session.started':
        break;
      case 'session.ended': {
        setSessionId(null);
        streamingIdRef.current = null;
        const m = cast<{ reason: string }>(msg);
        setMessages(prev => [
          ...prev,
          { id: uuid(), role: 'assistant', text: `[ Session ended: ${m.reason} ]` },
        ]);
        break;
      }
      case 'message.assistant_chunk': {
        const m = cast<{ content_block: { text: string } }>(msg);
        const text = m.content_block?.text ?? '';
        if (!text) break;

        if (!streamingIdRef.current) {
          streamingIdRef.current = uuid();
        }
        const streamId = streamingIdRef.current;

        setMessages(prev => {
          const existing = prev.find(item => item.id === streamId);
          if (existing) {
            return prev.map(item =>
              item.id === streamId ? { ...item, text: item.text + text } : item
            );
          }
          return [...prev, { id: streamId, role: 'assistant' as const, text }];
        });
        break;
      }
      case 'tool_use.request': {
        const m = cast<{ tool_use_id: string; tool_name: string; tool_input: Record<string, unknown> }>(msg);
        streamingIdRef.current = null;
        setMessages(prev => [
          ...prev,
          {
            id: m.tool_use_id,
            role: 'tool' as const,
            text: '',
            toolName: m.tool_name,
            toolInput: m.tool_input,
            pending: true,
          },
        ]);
        break;
      }
      case 'tool_use.result': {
        const m = cast<{ tool_use_id: string; result: unknown }>(msg);
        setMessages(prev =>
          prev.map(item =>
            item.id === m.tool_use_id
              ? { ...item, toolResult: String(m.result), pending: false }
              : item
          )
        );
        break;
      }
      case 'error': {
        const m = cast<{ message: string }>(msg);
        setMessages(prev => [
          ...prev,
          { id: uuid(), role: 'assistant' as const, text: `⚠ ${m.message}` },
        ]);
        break;
      }
    }
  }, []);

  useEffect(() => {
    if (!token) return;

    let reconnectTimer: ReturnType<typeof setTimeout>;
    let alive = true;

    function connect() {
      if (!alive) return;
      setWsStatus('connecting');
      const ws = new WebSocket(`${WS_URL}/ws/phone`);
      wsRef.current = ws;

      ws.onopen = () => {
        ws.send(JSON.stringify({ type: 'client.auth', token }));
        setWsStatus('connected');
      };
      ws.onmessage = e => handleMsg(e.data as string);
      ws.onclose = () => {
        setWsStatus('disconnected');
        // Clear session on disconnect — stale session ID causes silent message drops after reconnect.
        setSessionId(null);
        streamingIdRef.current = null;
        if (alive) reconnectTimer = setTimeout(connect, 3000);
      };
      ws.onerror = () => ws.close();
    }

    connect();
    return () => {
      alive = false;
      clearTimeout(reconnectTimer);
      wsRef.current?.close();
    };
  }, [token, handleMsg]);

  const startSession = useCallback((deviceId: string) => {
    setActiveDeviceId(deviceId);
    setMessages([]);
    streamingIdRef.current = null;
    send({ type: 'session.start', device_id: deviceId });
  }, [send]);

  const endSession = useCallback(() => {
    if (sessionId && activeDeviceId) {
      send({ type: 'session.end', session_id: sessionId, device_id: activeDeviceId });
    }
    setSessionId(null);
    setActiveDeviceId(null);
    setMessages([]);
    streamingIdRef.current = null;
  }, [send, sessionId, activeDeviceId]);

  const sendMessage = useCallback((content: string) => {
    if (!sessionId) return;
    const id = uuid();
    streamingIdRef.current = null; // new user turn → next assistant chunk starts fresh bubble
    setMessages(prev => [...prev, { id, role: 'user', text: content }]);
    send({ type: 'message.user', session_id: sessionId, content });
  }, [send, sessionId]);

  const approveTool = useCallback((toolUseId: string) => {
    send({ type: 'tool_use.approve', tool_use_id: toolUseId });
  }, [send]);

  const denyTool = useCallback((toolUseId: string) => {
    send({ type: 'tool_use.deny', tool_use_id: toolUseId, reason: 'user denied' });
  }, [send]);

  return {
    wsStatus, devices, activeDeviceId, sessionId, messages,
    startSession, endSession, sendMessage, approveTool, denyTool, send,
  };
}
