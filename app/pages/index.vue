<script setup lang="ts">
import type { NodeDetail } from '~/types'
import { backendUrl } from '~/utils/api'

const {
  messages, isStreaming, nodes, edges, selectedNode, nodeDetails,
  conversationId, conversationName, activeQuestion, pendingPermission, selectedModel, permissionMode, permissionStyle,
  cwd, loadingSession,
  submitPrompt, stopExecution, respondToPermission, selectNode, resetSession, loadSession, tryClaimActiveSession,
} = useChatSession()

const { sessions, loading: sessionsLoading, fetchSessionsForPath } = useProjects()

const isSidebarOpen = ref(false)
const showGraph = ref(false)
const { isLargeScreen } = useScreenSize()

const mobileNodeDetail = ref<NodeDetail | null>(null)

interface ActiveSession { sessionId: string; cwd: string; startedAt: string }
const activeSessions = ref<ActiveSession[]>([])

async function fetchActiveSessions() {
  try {
    const res = await fetch(backendUrl('/api/active-sessions'))
    if (res.ok) activeSessions.value = await res.json()
  } catch {}
}

let pollTimer: ReturnType<typeof setInterval> | null = null
onMounted(() => {
  fetchActiveSessions()
  pollTimer = setInterval(fetchActiveSessions, 3000)
})
onUnmounted(() => { if (pollTimer) clearInterval(pollTimer) })

const CHAT_WIDTH_KEY = 'chat-panel-width'
const MIN_CHAT_WIDTH = 280
const MAX_CHAT_WIDTH = 900

const chatWidth = ref(420)
const isDragging = ref(false)

onMounted(() => {
  const saved = localStorage.getItem(CHAT_WIDTH_KEY)
  if (saved) chatWidth.value = Math.min(MAX_CHAT_WIDTH, Math.max(MIN_CHAT_WIDTH, parseInt(saved)))
})

function startResize(e: MouseEvent) {
  isDragging.value = true
  const startX = e.clientX
  const startWidth = chatWidth.value

  function onMove(e: MouseEvent) {
    const delta = e.clientX - startX
    chatWidth.value = Math.min(MAX_CHAT_WIDTH, Math.max(MIN_CHAT_WIDTH, startWidth + delta))
  }

  function onUp() {
    isDragging.value = false
    localStorage.setItem(CHAT_WIDTH_KEY, String(chatWidth.value))
    window.removeEventListener('mousemove', onMove)
    window.removeEventListener('mouseup', onUp)
  }

  window.addEventListener('mousemove', onMove)
  window.addEventListener('mouseup', onUp)
}

watch(showGraph, (v) => { if (!v) mobileNodeDetail.value = null })

watch(cwd, (path, oldPath) => {
  if (path) {
    fetchSessionsForPath(path)
    if (oldPath && path !== oldPath) {
      resetSession()
    }
  }
})

function handleLoadSession(sessionId: string) {
  const encoded = cwd.value.replace(/\//g, '-')
  loadSession(encoded, sessionId)
}

function handleNewConversation() {
  resetSession()
}

function handleProjectSelect(sessionId: string, projectPath: string, encodedDir: string) {
  cwd.value = projectPath
  fetchSessionsForPath(projectPath)
  loadSession(encodedDir, sessionId)
}

async function handleRestoreSession(sessionId: string, sessionCwd: string) {
  cwd.value = sessionCwd
  fetchSessionsForPath(sessionCwd)
  const encoded = sessionCwd.replace(/\//g, '-')
  await loadSession(encoded, sessionId)
  await tryClaimActiveSession(sessionId)
}

function handlePermissionResponse(allow: boolean, answers?: Record<string, string>) {
  respondToPermission(allow, answers)
}
</script>

<template>
  <div class="h-full w-full overflow-hidden flex font-sans bg-background text-foreground">

    <USlideover
      v-model="isSidebarOpen"
      side="left"
      :ui="{ width: 'max-w-64', wrapper: 'z-[100]' }"
    >
      <Sidebar @navigate="isSidebarOpen = false" class="h-full" />
    </USlideover>

    <div class="flex-1 flex overflow-hidden relative" :class="isDragging ? 'select-none cursor-col-resize' : ''">

      <ChatPanel
        :messages="messages"
        :isStreaming="isStreaming"
        :loadingSession="loadingSession"
        :activeQuestion="activeQuestion"
        :pendingPermission="pendingPermission"
        :conversationId="conversationId"
        :conversationName="conversationName"
        :sessions="sessions"
        :sessionsLoading="sessionsLoading"
        :activeSessions="activeSessions"
        v-model:activeModel="selectedModel"
        v-model:activeMode="permissionMode"
        v-model:permissionStyle="permissionStyle"
        v-model:cwd="cwd"
        @menuClick="isSidebarOpen = true"
        @showGraphClick="showGraph = true"
        @submit="submitPrompt"
        @stop="stopExecution"
        @permissionResponse="handlePermissionResponse"
        @loadSession="handleLoadSession"
        @newConversation="handleNewConversation"
        @restoreSession="handleRestoreSession"
        class="w-full lg:flex-shrink-0 lg:border-r lg:border-border/30 z-20"
        :style="isLargeScreen ? { width: chatWidth + 'px' } : {}"
      />

      <div
        v-if="isLargeScreen"
        class="w-1 flex-shrink-0 relative z-30 group cursor-col-resize"
        @mousedown="startResize"
      >
        <div class="absolute inset-y-0 -left-1 -right-1 group-hover:bg-sky-500/20 transition-colors" :class="isDragging ? 'bg-sky-500/30' : ''" />
      </div>

      <div class="hidden lg:block flex-1 relative z-0 h-full bg-slate-50 dark:bg-slate-950">
        <WorkMap
          :nodes="nodes"
          :edges="edges"
          @nodeClick="selectNode"
        />
      </div>

    </div>

    <USlideover
      v-model="showGraph"
      class="lg:hidden"
      :ui="{ width: 'w-full sm:max-w-[85vw]', wrapper: 'z-[90]', overlay: { background: 'bg-background/60 backdrop-blur-sm' } }"
    >
      <div class="flex flex-col h-full bg-background">
        <div class="h-12 border-b border-border/30 flex items-center justify-between px-4 shrink-0">
          <div class="flex items-center gap-2">
            <UIcon name="i-lucide-map" class="w-4 h-4 text-sky-400" />
            <span class="text-sm font-heading font-semibold text-foreground">Work Map</span>
            <span class="text-[10px] bg-sky-500/10 text-sky-400 px-1.5 py-0.5 rounded-md font-mono">{{ nodes.length }} nodes</span>
          </div>
          <UButton color="gray" variant="ghost" icon="i-lucide-x" size="sm" class="rounded-lg" @click="showGraph = false" />
        </div>
        <div class="flex-1 relative">
          <WorkMap
            :nodes="nodes"
            :edges="edges"
            @nodeClick="(id: string) => { mobileNodeDetail = nodeDetails[id] || { id, type: 'unknown', label: id } }"
          />
          <NodeBottomSheet
            :node="mobileNodeDetail"
            @close="mobileNodeDetail = null"
          />
        </div>
      </div>
    </USlideover>

    <LogPanel
      :node="isLargeScreen ? selectedNode : null"
      @close="selectedNode = null"
      class="z-[100]"
    />

  </div>
</template>
