import type { ChatMessage, BackendMessage, ClaudeEvent, ContentBlock, PermissionRequest } from '~/types'
import { backendUrl } from '~/utils/api'

interface SessionMessageRaw {
  uuid: string
  type: string
  role: string
  content: any
  timestamp: string
  model?: string
}

export function useChatSession() {
  const graph = useWorkGraph()
  const { clientId, onMessage, onReconnect } = useWebSocket()

  const messages = ref<ChatMessage[]>([])
  const isStreaming = ref(false)
  const conversationId = ref<string | null>(null)
  const conversationName = ref<string | null>(null)
  const activeQuestion = ref<Record<string, any> | null>(null)
  const pendingPermission = ref<PermissionRequest | null>(null)
  const selectedModel = ref('sonnet')
  const permissionMode = ref('agent')
  const permissionStyle = ref('ask')
  const cwd = ref('')
  const loadingSession = ref(false)

  async function tryClaimActiveSession(sessionId: string) {
    const claim = async (cId: string) => {
      try {
        const res = await fetch(backendUrl(`/api/sessions/${sessionId}/claim`), {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ clientId: cId }),
        })
        const data = await res.json()
        if (data.active) {
          isStreaming.value = true
        }
      } catch {}
    }

    if (clientId.value) {
      await claim(clientId.value)
    } else {
      const stop = watch(clientId, async (id) => {
        if (id) {
          stop()
          await claim(id)
        }
      })
    }
  }

  async function loadPrefs() {
    try {
      const res = await fetch(backendUrl('/api/prefs'))
      if (!res.ok) return
      const prefs = await res.json()
      if (prefs.model) selectedModel.value = prefs.model
      if (prefs.permissionMode) permissionMode.value = prefs.permissionMode
      if (prefs.permissionStyle) permissionStyle.value = prefs.permissionStyle
      if (prefs.cwd) cwd.value = prefs.cwd
      if (prefs.conversationId && prefs.cwd) {
        const encodedDir = prefs.cwd.replace(/\//g, '-')
        await loadSession(encodedDir, prefs.conversationId)
        await tryClaimActiveSession(prefs.conversationId)
      }
    } catch {}
  }

  async function savePrefs() {
    try {
      await fetch(backendUrl('/api/prefs'), {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          model: selectedModel.value,
          permissionMode: permissionMode.value,
          permissionStyle: permissionStyle.value,
          cwd: cwd.value,
          conversationId: conversationId.value ?? '',
        }),
      })
    } catch {}
  }

  let restoringPrefs = false

  watch([selectedModel, permissionMode, permissionStyle, cwd, conversationId], () => {
    if (!restoringPrefs) savePrefs()
  })

  onMounted(async () => {
    restoringPrefs = true
    await loadPrefs()
    restoringPrefs = false
  })

  let assistantMsg: ChatMessage | null = null
  let activeResponseNodeId = ''
  let toolsInTurn: string[] = []
  let lastSourceNodeId = ''

  function ensureResponseNode() {
    if (activeResponseNodeId) return

    activeResponseNodeId = graph.nextId('response')
    graph.addNode(activeResponseNodeId, 'Claude', 'response', 0, 0, 'running')

    if (toolsInTurn.length > 0) {
      for (const tid of toolsInTurn) {
        graph.addEdge(tid, activeResponseNodeId, true)
      }
      toolsInTurn = []
    } else if (lastSourceNodeId) {
      graph.addEdge(lastSourceNodeId, activeResponseNodeId, true)
    }
    lastSourceNodeId = activeResponseNodeId
  }

  function updateNodeLabelFromContent(nodeId: string, content: string) {
    const details = graph.nodeDetails.value[nodeId]
    if (details) {
      details.result = content
        .replace(/<span class="thinking-text">/g, '> [Thinking]: ')
        .replace(/<\/span>/g, '\n\n')
    }
    const plainText = content.replace(/<[^>]*>/g, '').replace(/\n/g, ' ').trim()
    if (plainText) {
      graph.updateNodeLabel(nodeId, plainText.slice(0, 40) + (plainText.length > 40 ? '...' : ''))
    }
  }

  function appendToAssistant(text: string) {
    const isRateLimit = /^You're out of extra usage/i.test(text)
    if (!assistantMsg) {
      assistantMsg = { id: graph.nextId('msg'), role: 'assistant', content: text, toolUses: [], isError: isRateLimit }
      messages.value = [...messages.value, assistantMsg]
    } else {
      assistantMsg.content += text
      if (isRateLimit) assistantMsg.isError = true
      messages.value = [...messages.value]
    }
  }

  function handleAssistantEvent(evt: ClaudeEvent) {
    ensureResponseNode()
    graph.updateNodeStatus(activeResponseNodeId, 'success')

    const content = evt.message?.content
    if (!Array.isArray(content)) return

    for (const block of content) {
      switch (block.type) {
        case 'thinking':
          appendToAssistant(`<span class="thinking-text">${block.thinking}</span>`)
          updateNodeLabelFromContent(activeResponseNodeId, assistantMsg!.content)
          break

        case 'text':
          appendToAssistant(block.text || '')
          updateNodeLabelFromContent(activeResponseNodeId, assistantMsg!.content)
          break

        case 'tool_use':
          handleToolUseBlock(block)
          break
      }
    }
  }

  function handleToolUseBlock(block: ContentBlock) {
    if (!assistantMsg) {
      assistantMsg = { id: graph.nextId('msg'), role: 'assistant', content: '', toolUses: [] }
      messages.value = [...messages.value, assistantMsg]
    }

    if (assistantMsg.toolUses?.some(t => t.id === block.id)) return

    const label = toolLabel(block.name!, block.input)

    assistantMsg.toolUses = [...(assistantMsg.toolUses || []), {
      id: block.id!,
      name: block.name!,
      input: block.input,
      status: 'running',
    }]
    messages.value = [...messages.value]

    const toolNodeId = `tool-${block.id}`
    graph.addNode(toolNodeId, label.slice(0, 30), block.name!, 0, 0, 'running', block.input as Record<string, any>)
    graph.addEdge(activeResponseNodeId, toolNodeId, true)

    toolsInTurn.push(toolNodeId)
    graph.nodeDetails.value[toolNodeId] = { id: toolNodeId, type: block.name!, label, input: block.input }
  }

  function handleUserEvent(evt: ClaudeEvent) {
    const content = evt.message?.content
    if (!Array.isArray(content)) return

    let hasToolResult = false
    for (const block of content) {
      if (block.type !== 'tool_result') continue
      hasToolResult = true

      const toolNodeId = `tool-${block.tool_use_id}`
      const isError = typeof block.content === 'string' && block.content.startsWith('Error')

      for (let i = messages.value.length - 1; i >= 0; i--) {
        const msg = messages.value[i]
        if (!msg.toolUses) continue
        const entry = msg.toolUses.find(t => t.id === block.tool_use_id)
        if (entry) {
          entry.result = typeof block.content === 'string' ? block.content : JSON.stringify(block.content)
          entry.status = 'done'
          entry.isError = isError
          messages.value = [...messages.value]
          break
        }
      }
      graph.updateNodeStatus(toolNodeId, isError ? 'error' : 'success')

      const details = graph.nodeDetails.value[toolNodeId]
      if (details) {
        let resultText = typeof block.content === 'string' ? block.content : JSON.stringify(block.content)
        if (details.type === 'Read' && typeof resultText === 'string') {
          resultText = resultText.replace(/^ *\d+[→\t]/gm, '')
        }
        details.result = resultText
      }
    }

    if (hasToolResult) {
      assistantMsg = null
      activeResponseNodeId = ''
    }
  }

  function handleResultEvent(evt: ClaudeEvent) {
    if (!evt.result) return
    ensureResponseNode()
    graph.updateNodeStatus(activeResponseNodeId, 'success')
    const isRateLimit = /^You're out of extra usage/i.test(evt.result)
    if (!assistantMsg) {
      assistantMsg = { id: graph.nextId('msg'), role: 'assistant', content: evt.result, isError: isRateLimit }
      messages.value = [...messages.value, assistantMsg]
    }
    updateNodeLabelFromContent(activeResponseNodeId, assistantMsg.content)
  }

  function handleBackendMessage(msg: BackendMessage) {
    switch (msg.type) {
      case 'claude_event': {
        const evt = msg.data as ClaudeEvent
        if (evt.type === 'system' && evt.subtype === 'init' && evt.session_id) {
          conversationId.value = evt.session_id
        }
        switch (evt.type) {
          case 'assistant': handleAssistantEvent(evt); break
          case 'user': handleUserEvent(evt); break
          case 'result': handleResultEvent(evt); break
        }
        break
      }

      case 'log':
        if (!assistantMsg) {
          assistantMsg = {
            id: graph.nextId('msg'),
            role: 'assistant',
            content: `\`\`\`\n${msg.data}\n\`\`\``,
            toolUses: [],
          }
          messages.value = [...messages.value, assistantMsg]
        } else {
          assistantMsg.content += `\n${msg.data}`
          messages.value = [...messages.value]
        }
        break

      case 'permission_request': {
        const req = msg.data as { requestId: string; toolName: string; toolInput: Record<string, any> }

        if (req.toolName === 'AskUserQuestion') {
          activeQuestion.value = req.toolInput
          pendingPermission.value = {
            requestId: req.requestId,
            toolName: req.toolName,
            input: req.toolInput,
          }
        } else if (req.toolName === 'ExitPlanMode') {
          pendingPermission.value = {
            requestId: req.requestId,
            toolName: req.toolName,
            input: req.toolInput,
          }
        } else {
          pendingPermission.value = {
            requestId: req.requestId,
            toolName: req.toolName,
            input: req.toolInput,
          }
        }
        break
      }

      case 'stderr': {
        const line = String(msg.data)
        for (let i = messages.value.length - 1; i >= 0; i--) {
          const m = messages.value[i]
          if (!m.toolUses) continue
          const runningBash = m.toolUses.find(t => t.name === 'Bash' && t.status === 'running')
          if (runningBash) {
            runningBash.liveOutput = (runningBash.liveOutput ?? '') + line + '\n'
            messages.value = [...messages.value]
            break
          }
        }
        break
      }

      case 'done':
        if (activeResponseNodeId) graph.updateNodeStatus(activeResponseNodeId, 'success')
        isStreaming.value = false
        break

      case 'error': {
        if (activeResponseNodeId) graph.updateNodeStatus(activeResponseNodeId, 'error')
        isStreaming.value = false
        const errMsg = String(msg.data)
        const isRateLimit = /out of extra usage|rate limit|too many requests/i.test(errMsg)
        const displayMsg = isRateLimit
          ? '⚠️ **Claude Code usage limit reached**\n\nYour API usage limit has been exceeded. Please wait for the limit to reset or check your Claude Code subscription for more credits.'
          : `Error: ${errMsg}`
        messages.value = [...messages.value, {
          id: graph.nextId('msg-err'),
          role: 'assistant',
          content: displayMsg,
          isError: true,
        }]
        break
      }
    }
  }

  onMessage(handleBackendMessage)

  onReconnect(async (newClientId: string) => {
    if (!conversationId.value || !isStreaming.value) return

    try {
      const res = await fetch(backendUrl(`/api/sessions/${conversationId.value}/claim`), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ clientId: newClientId }),
      })
      const data = await res.json()

      if (!data.active) {
        if (cwd.value) {
          const encodedDir = cwd.value.replace(/\//g, '-')
          await loadSession(encodedDir, conversationId.value)
        } else {
          isStreaming.value = false
        }
      }
    } catch {
      isStreaming.value = false
    }
  })

  async function respondToPermission(allow: boolean, answers?: Record<string, string>) {
    const req = pendingPermission.value
    if (!req) return

    const body: Record<string, any> = {
      action: allow ? 'approve' : 'deny',
    }

    if (allow && req.toolName === 'AskUserQuestion' && answers) {
      body.updatedInput = {
        ...req.input,
        answers,
      }
    }

    try {
      await fetch(backendUrl(`/api/permissions/${req.requestId}/decision`), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
    } catch (err) {
      console.error('Permission decision error:', err)
    }

    if (allow && req.toolName === 'ExitPlanMode') {
      permissionMode.value = 'agent'
    }

    pendingPermission.value = null
    activeQuestion.value = null

    if (!allow) {
      try {
        await fetch(backendUrl('/stop'), {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ clientId: clientId.value }),
        })
      } catch {}
      isStreaming.value = false
    }
  }

  async function submitPrompt(payload: { prompt: string; cwd: string; model?: string; mode?: string; permissionStyle?: string }) {
    messages.value = [...messages.value, {
      id: graph.nextId('msg'),
      role: 'user',
      content: payload.prompt,
    }]

    if (!conversationName.value) {
      conversationName.value = payload.prompt.length > 60 ? payload.prompt.slice(0, 60) + '...' : payload.prompt
    }

    const promptNodeId = graph.nextId('prompt')
    graph.addNode(promptNodeId, payload.prompt.slice(0, 40), 'prompt', 0, 0, 'success')

    if (lastSourceNodeId) {
      graph.addEdge(lastSourceNodeId, promptNodeId)
    }

    activeResponseNodeId = ''
    lastSourceNodeId = promptNodeId
    toolsInTurn = []
    assistantMsg = null
    isStreaming.value = true

    try {
      await fetch(backendUrl('/run'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          prompt: payload.prompt,
          cwd: payload.cwd,
          conversationId: conversationId.value,
          model: payload.model || selectedModel.value,
          mode: payload.mode || permissionMode.value,
          permissionStyle: payload.permissionStyle || permissionStyle.value,
          clientId: clientId.value,
        }),
      })
    } catch (err: any) {
      if (activeResponseNodeId) graph.updateNodeStatus(activeResponseNodeId, 'error')
      isStreaming.value = false
      messages.value = [...messages.value, {
        id: graph.nextId('msg-err'),
        role: 'assistant',
        content: `Connection error: ${err.message}`,
      }]
    }
  }

  async function stopExecution() {
    if (pendingPermission.value) {
      try {
        await fetch(backendUrl(`/api/permissions/${pendingPermission.value.requestId}/decision`), {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ action: 'deny', reason: 'User stopped execution' }),
        })
      } catch {}
      pendingPermission.value = null
      activeQuestion.value = null
    }

    try {
      await fetch(backendUrl('/stop'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ clientId: clientId.value }),
      })
    } catch (err: any) {
      console.error('Stop error:', err)
    }

    isStreaming.value = false
  }

  function selectNode(nodeId: string) {
    graph.selectedNode.value = graph.nodeDetails.value[nodeId] || { id: nodeId, type: 'unknown', label: nodeId }
  }

  function resetSession() {
    graph.reset()
    messages.value = []
    conversationId.value = null
    conversationName.value = null
    isStreaming.value = false
    activeQuestion.value = null
    pendingPermission.value = null
    assistantMsg = null
    activeResponseNodeId = ''
    toolsInTurn = []
    lastSourceNodeId = ''
  }

  async function loadSession(encodedDir: string, sessionId: string) {
    loadingSession.value = true
    try {
      const res = await fetch(backendUrl(`/projects/${encodedDir}/sessions/${sessionId}`))
      const rawMessages: SessionMessageRaw[] = await res.json()

      resetSession()
      conversationId.value = sessionId

      for (const raw of rawMessages) {
        if (raw.type === 'assistant') {
          const blocks = raw.content as ContentBlock[]
          if (!Array.isArray(blocks)) continue
          handleAssistantEvent({ type: 'assistant', message: { content: blocks } })
        } else if (raw.type === 'user') {
          if (typeof raw.content === 'string') {
            if (!conversationName.value) {
              conversationName.value = raw.content.length > 60 ? raw.content.slice(0, 60) + '...' : raw.content
            }
            messages.value = [...messages.value, {
              id: graph.nextId('msg'),
              role: 'user',
              content: raw.content,
            }]
            const nodeId = graph.nextId('prompt')
            graph.addNode(nodeId, raw.content.slice(0, 40), 'prompt', 0, 0, 'success')
            if (lastSourceNodeId) graph.addEdge(lastSourceNodeId, nodeId)
            activeResponseNodeId = ''
            lastSourceNodeId = nodeId
            toolsInTurn = []
            assistantMsg = null
          } else if (Array.isArray(raw.content)) {
            handleUserEvent({ type: 'user', message: { content: raw.content as ContentBlock[] } })
          }
        }
      }

      if (activeResponseNodeId) graph.updateNodeStatus(activeResponseNodeId, 'success')
    } finally {
      loadingSession.value = false
    }
  }

  return {
    messages,
    isStreaming,
    conversationId,
    conversationName,
    activeQuestion,
    pendingPermission,
    selectedModel,
    permissionMode,
    permissionStyle,
    cwd,
    loadingSession,
    nodes: graph.nodes,
    edges: graph.edges,
    nodeDetails: graph.nodeDetails,
    selectedNode: graph.selectedNode,
    submitPrompt,
    stopExecution,
    respondToPermission,
    selectNode,
    resetSession,
    loadSession,
    tryClaimActiveSession,
  }
}
