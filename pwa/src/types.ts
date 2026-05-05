export interface Device {
  id: string;
  name: string;
  status: 'online' | 'busy' | 'offline';
  last_seen?: number;
}

export interface ChatMessage {
  id: string;
  role: 'user' | 'assistant' | 'tool';
  text: string;
  toolName?: string;
  toolInput?: Record<string, unknown>;
  toolResult?: string;
  pending?: boolean;          // tool is running (no result yet)
  awaitingApproval?: boolean; // tool needs phone-side approve/deny before running
}

export interface RelayMsg {
  type: string;
  [key: string]: unknown;
}
