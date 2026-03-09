export type NodeStatus = 'idle' | 'running' | 'success' | 'error'

export interface ToolUseEntry {
  id: string
  name: string
  input?: Record<string, any>
  result?: string
  liveOutput?: string
  status: 'running' | 'done'
  isError?: boolean
}

export interface ChatMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  toolUses?: ToolUseEntry[]
  isError?: boolean
}

export interface NodeDetail {
  id: string
  type: string
  label: string
  input?: Record<string, any>
  result?: string
  color?: 'blue' | 'green' | 'amber' | 'rose' | 'indigo' | 'slate'
}

export interface ProjectInfo {
  name: string
  path: string
  encodedDir: string
}

export interface SessionSummary {
  id: string
  display: string
  timestamp: number
  project: string
}

export interface PermissionRequest {
  requestId: string
  toolName: string
  input: Record<string, any>
}

export interface BackendMessage {
  type: 'claude_event' | 'log' | 'stderr' | 'done' | 'error' | 'permission_request'
  data?: any
}

export interface ClaudeEvent {
  type: 'assistant' | 'user' | 'result' | 'system'
  subtype?: string
  session_id?: string
  message?: {
    content: ContentBlock[]
  }
  result?: string
}

export interface ContentBlock {
  type: 'text' | 'thinking' | 'tool_use' | 'tool_result'
  text?: string
  thinking?: string
  id?: string
  name?: string
  input?: Record<string, any>
  tool_use_id?: string
  content?: string | any
}
