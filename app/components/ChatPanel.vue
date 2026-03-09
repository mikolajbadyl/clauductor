<script setup lang="ts">
import type { ChatMessage, SessionSummary, PermissionRequest } from '~/types'
import { backendUrl } from '~/utils/api'

interface ActiveSession { sessionId: string; cwd: string; startedAt: string }

const props = defineProps<{
  messages: ChatMessage[]
  isStreaming: boolean
  loadingSession: boolean
  activeQuestion?: Record<string, any> | null | undefined
  pendingPermission?: PermissionRequest | null | undefined
  activeModel: string
  activeMode: string
  permissionStyle: string
  conversationId: string | null
  conversationName: string | null
  sessions: SessionSummary[]
  sessionsLoading: boolean
  cwd: string
  activeSessions?: ActiveSession[]
}>()

const emit = defineEmits<{
  (e: 'submit', payload: { prompt: string; cwd: string; model?: string; mode?: string; permissionStyle?: string }): void
  (e: 'update:activeModel', model: string): void
  (e: 'update:activeMode', mode: string): void
  (e: 'update:permissionStyle', style: string): void
  (e: 'update:cwd', cwd: string): void
  (e: 'menuClick'): void
  (e: 'showGraphClick'): void
  (e: 'loadSession', sessionId: string): void
  (e: 'newConversation'): void
  (e: 'stop'): void
  (e: 'permissionResponse', allow: boolean, answers?: Record<string, string>): void
  (e: 'restoreSession', sessionId: string, cwd: string): void
}>()

const prompt = ref('')
const promptInputRef = ref<HTMLTextAreaElement | null>(null)
const cwd = computed({
  get: () => props.cwd,
  set: (val) => emit('update:cwd', val),
})
const cwdError = ref(false)
const cwdOpen = ref(false)
const cwdSearch = ref('')
const cwdInputRef = ref<HTMLInputElement | null>(null)
const chatContainer = ref<HTMLElement | null>(null)
const selectedAnswers = ref<Record<string, string>>({})

const convOpen = ref(false)
const convSearch = ref('')
const convInputRef = ref<HTMLInputElement | null>(null)

const activeSessionsOpen = ref(false)

const expandedTools = ref(new Set<string>())
const collapsedByUser = ref(new Set<string>())

function isToolExpanded(toolId: string) {
  return expandedTools.value.has(toolId)
}

function toggleTool(toolId: string) {
  if (expandedTools.value.has(toolId)) {
    expandedTools.value.delete(toolId)
    collapsedByUser.value.add(toolId)
  } else {
    expandedTools.value.add(toolId)
    collapsedByUser.value.delete(toolId)
  }
}

function expandTool(toolId: string) {
  if (!collapsedByUser.value.has(toolId)) {
    expandedTools.value.add(toolId)
  }
}

function formatBashOutput(output: string): string {
  if (!output) return ''
  return output.replace(/^[\s\n]*/, '').slice(0, 2000)
}

const usageOpen = ref(false)
const usageData = ref<any>(null)
const usageLoading = ref(false)
const usageError = ref('')
let usageCacheTime = 0
const USAGE_CACHE_MS = 5 * 60 * 1000

async function fetchUsage() {
  const now = Date.now()
  if (usageData.value && now - usageCacheTime < USAGE_CACHE_MS) return

  usageLoading.value = true
  usageError.value = ''
  try {
    const res = await fetch(backendUrl('/api/usage'))
    if (res.ok) {
      usageData.value = await res.json()
      usageCacheTime = now
    } else if (res.status === 401) {
      usageError.value = 'Not logged in (run: claude login)'
    } else {
      usageError.value = 'Failed to load'
    }
  } catch (e: any) {
    usageError.value = e?.message || 'Error'
  } finally {
    usageLoading.value = false
  }
}

function usagePercent(usage: any) {
  if (!usage || usage.utilization == null) return 0
  return Math.min(100, Math.round(usage.utilization))
}

function formatResetDate(iso: string) {
  if (!iso) return ''
  const date = new Date(iso)
  return date.toLocaleString('en-GB', { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit' })
}

function openUsage() {
  usageOpen.value = !usageOpen.value
  if (usageOpen.value) fetchUsage()
}

function formatCwd(cwd: string) {
  const home = cwd.match(/^\/home\/[^/]+\/(.+)$/)
  return home ? '~/' + home[1] : cwd
}

function formatAge(iso: string) {
  if (!iso) return ''
  const diff = Date.now() - new Date(iso).getTime()
  const s = Math.floor(diff / 1000)
  if (s < 60) return `${s}s ago`
  const m = Math.floor(s / 60)
  if (m < 60) return `${m}m ago`
  return `${Math.floor(m / 60)}h ago`
}

const filteredSessions = computed(() => {
  const q = convSearch.value.toLowerCase()
  if (!q) return props.sessions
  return props.sessions.filter(s => s.display?.toLowerCase().includes(q))
})

const currentSessionName = computed(() => {
  if (!props.conversationId) return null
  const session = props.sessions.find(s => s.id === props.conversationId)
  return session?.display || props.conversationName || null
})

function openConvPicker() {
  convOpen.value = true
  convSearch.value = ''
  nextTick(() => convInputRef.value?.focus())
}

function selectSession(session: SessionSummary) {
  convOpen.value = false
  emit('loadSession', session.id)
}

function newConversation() {
  convOpen.value = false
  emit('newConversation')
}

function formatConvDate(ts: number): string {
  if (!ts) return ''
  return new Date(ts).toLocaleDateString('en-GB', { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit' })
}

const { projects, loading: projectsLoading, fetchProjects, checkPathExists, createProjectFolder } = useProjects()

const pathNotFound = ref(false)

watch(cwd, async (newVal) => {
  if (!newVal) {
    pathNotFound.value = false
    return
  }
  const exists = await checkPathExists(newVal)
  pathNotFound.value = !exists
})

const filteredProjects = computed(() => {
  const q = cwdSearch.value.toLowerCase()
  if (!q) return projects.value
  return projects.value.filter(p =>
    p.name.toLowerCase().includes(q) || p.path.toLowerCase().includes(q),
  )
})

function openCwdPicker() {
  cwdOpen.value = true
  cwdSearch.value = ''
  if (projects.value.length === 0) fetchProjects()
  nextTick(() => cwdInputRef.value?.focus())
}

function selectCwd(path: string) {
  cwd.value = path
  cwdOpen.value = false
  cwdError.value = false
}

function confirmCustomPath() {
  if (cwdSearch.value.trim()) {
    selectCwd(cwdSearch.value.trim())
  }
}

function handleCwdKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    cwdOpen.value = false
  } else if (e.key === 'Enter') {
    e.preventDefault()
    confirmCustomPath()
  }
}

const verbs = [
  "All aboard", "Departing platform 9¾", "Checking the timetable", "Validating tickets",
  "Pulling into the station", "Accelerating to full speed", "Passing through the tunnel",
  "Switching tracks", "Coupling the wagons", "Sounding the horn", "Inspecting the locomotive",
  "Loading the cargo", "Clearing the signals", "Laying new tracks", "Stoking the boiler",
  "Consulting the conductor", "Scheduling the departure", "Announcing next stop",
  "Crossing the bridge", "Entering the depot", "Oiling the pistons", "Checking the manifest",
  "Shunting the cars", "Blowing off steam", "Signaling ahead", "Releasing the brakes",
  "Requesting clearance", "Awaiting green light", "Decoupling at junction",
  "Arriving on schedule", "Punching the tickets", "Whistling through the valley",
]
const currentVerb = ref(verbs[0])
let verbInterval: ReturnType<typeof setInterval> | null = null

watch(() => props.isStreaming, (streaming) => {
  if (streaming) {
    verbInterval = setInterval(() => {
      currentVerb.value = verbs[Math.floor(Math.random() * verbs.length)]
    }, 6000)
  } else if (verbInterval) {
    clearInterval(verbInterval)
    verbInterval = null
  }
})

type TodoItem = { content: string; status: 'pending' | 'in_progress' | 'completed'; activeForm: string }

const currentTodos = computed((): TodoItem[] => {
  let startIdx = 0
  for (let i = props.messages.length - 1; i >= 0; i--) {
    if (props.messages[i].role === 'user') {
      startIdx = i + 1
      break
    }
  }
  for (let i = props.messages.length - 1; i >= startIdx; i--) {
    const msg = props.messages[i]
    if (msg.toolUses) {
      for (let j = msg.toolUses.length - 1; j >= 0; j--) {
        const tool = msg.toolUses[j]
        if (tool.name === 'TodoWrite' && Array.isArray(tool.input?.todos)) {
          return tool.input.todos
        }
      }
    }
  }
  return []
})

const sortedTodos = computed(() => [
  ...currentTodos.value.filter(t => t.status !== 'completed'),
  ...currentTodos.value.filter(t => t.status === 'completed'),
])

const activeTodoText = computed(() =>
  currentTodos.value.find(t => t.status === 'in_progress')?.activeForm ?? null
)

onUnmounted(() => {
  if (verbInterval) clearInterval(verbInterval)
})

const models = [
  { label: 'Sonnet', value: 'sonnet', icon: 'i-lucide-zap' },
  { label: 'Haiku', value: 'haiku', icon: 'i-lucide-mouse-pointer' },
  { label: 'Opus', value: 'opus', icon: 'i-lucide-graduation-cap' },
]

const modes = [
  { label: 'Agent', value: 'agent', icon: 'i-lucide-terminal' },
  { label: 'Plan', value: 'plan', icon: 'i-lucide-pen-square' },
]

const permStyles = [
  { label: 'Ask', value: 'ask', icon: 'i-lucide-shield-question' },
  { label: 'YOLO', value: 'yolo', icon: 'i-lucide-skull' },
]

interface ProfileEntry {
  name: string
  env: Record<string, string>
}

interface ProfilesConfig {
  active: string
  profiles: Record<string, ProfileEntry>
}

const profilesConfig = ref<ProfilesConfig>({ active: '', profiles: {} })
const modelDropdownOpen = ref(false)

const profileEntries = computed(() => Object.entries(profilesConfig.value.profiles))

async function fetchProfiles() {
  try {
    const res = await fetch(backendUrl('/api/profiles'))
    if (res.ok) profilesConfig.value = await res.json()
  } catch (e) {
    console.error('Failed to fetch profiles', e)
  }
}

async function setActiveProfile(id: string) {
  try {
    const res = await fetch(backendUrl('/api/profiles/active'), {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ active: id }),
    })
    if (res.ok) {
      profilesConfig.value.active = id
    }
  } catch (e) {
    console.error('Failed to set active profile', e)
  }
}

function selectModel(value: string) {
  emit('update:activeModel', value)
  modelDropdownOpen.value = false
}

onMounted(() => {
  fetchProfiles()
})

watch(prompt, () => {
  nextTick(() => {
    if (promptInputRef.value) {
      promptInputRef.value.style.height = 'auto'
      promptInputRef.value.style.height = Math.min(promptInputRef.value.scrollHeight, 192) + 'px'
    }
  })
})

const editDiffLines = computed(() => {
  const old = (props.pendingPermission?.input?.old_string as string || '').split('\n')
  const nw = (props.pendingPermission?.input?.new_string as string || '').split('\n')
  const lines: { key: string; text: string; cls: string }[] = []
  let i = 0
  for (const l of old) {
    lines.push({ key: `r${i++}`, text: '- ' + l, cls: 'text-rose-400/80 bg-rose-500/5' })
  }
  for (const l of nw) {
    lines.push({ key: `a${i++}`, text: '+ ' + l, cls: 'text-emerald-400/80 bg-emerald-500/5' })
  }
  return lines
})

const planContent = computed(() => {
  for (let i = props.messages.length - 1; i >= 0; i--) {
    const msg = props.messages[i]
    if (msg.role !== 'assistant' || !msg.toolUses) continue
    for (let j = msg.toolUses.length - 1; j >= 0; j--) {
      const tool = msg.toolUses[j]
      if (tool.name === 'Write' && /\.(md|mdx|markdown)$/i.test(tool.input?.file_path || '')) {
        return tool.input?.content as string || null
      }
    }
  }
  return null
})

const planExpanded = ref(false)

const canSubmit = computed(() => prompt.value.trim() && cwd.value.trim() && !pathNotFound.value)

function onSubmit() {
  if (!cwd.value.trim() || pathNotFound.value) {
    cwdError.value = !cwd.value.trim()
    return
  }
  cwdError.value = false
  if (canSubmit.value) {
    emit('submit', {
      prompt: prompt.value.trim(),
      cwd: cwd.value.trim(),
      model: props.activeModel,
      mode: props.activeMode,
      permissionStyle: props.permissionStyle,
    })
    prompt.value = ''
  }
}

function handleEnterKey(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    if (!props.isStreaming) {
      onSubmit()
    }
  }
}

async function handleCreateFolder() {
  if (!cwd.value) return
  if (confirm(`Do you want to create directory: ${cwd.value}?`)) {
    const success = await createProjectFolder(cwd.value)
    if (success) {
      pathNotFound.value = false
      fetchProjects()
    } else {
      alert('Failed to create directory')
    }
  }
}


function selectAnswer(questionIdx: string | number, value: string) {
  selectedAnswers.value[questionIdx] = value
}

function submitAllAnswers() {
  const qs = props.activeQuestion?.questions || (Array.isArray(props.activeQuestion) ? props.activeQuestion : [])
  const answers: Record<string, string> = {}
  for (let idx = 0; idx < qs.length; idx++) {
    const q = qs[idx]
    answers[q.question] = selectedAnswers.value[idx] || 'No answer'
  }

  emit('permissionResponse', true, answers)
  selectedAnswers.value = {}
}

watch(() => props.messages, async () => {
  for (const msg of props.messages) {
    if (msg.toolUses) {
      for (const tool of msg.toolUses) {
        if ((tool.name === 'Edit' || tool.name === 'Write' || tool.name === 'Bash') && tool.status === 'running') {
          expandTool(tool.id)
        } else if (tool.name === 'Edit' || tool.name === 'Write' || tool.name === 'Bash') {
          if (!collapsedByUser.value.has(tool.id)) {
            expandedTools.value.add(tool.id)
          }
        }
      }
    }
  }

  await nextTick()
  if (chatContainer.value) {
    chatContainer.value.scrollTop = chatContainer.value.scrollHeight
  }
}, { deep: true })
</script>

<template>
  <div class="h-full bg-background flex flex-col">

    <div class="h-14 border-b border-border/40 flex items-center justify-between px-4 shrink-0">
      <div class="flex items-center gap-3 min-w-0 flex-1">
        <button
          @click="emit('menuClick')"
          class="w-9 h-9 rounded-lg flex items-center justify-center text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-900 hover:text-slate-700 dark:hover:text-slate-100 transition-colors shrink-0"
        >
          <UIcon name="i-lucide-menu" class="w-5 h-5" />
        </button>

        <div class="relative min-w-0 flex-1">
          <button
            @click="openConvPicker"
            class="flex items-center gap-2 text-sm font-medium text-slate-700 dark:text-slate-200 hover:text-slate-900 dark:hover:text-white transition-colors max-w-full"
          >
            <span class="truncate">{{ currentSessionName || (conversationId ? 'Conversation' : 'New conversation') }}</span>
            <UIcon name="i-lucide-chevron-down" class="w-3.5 h-3.5 text-slate-500 shrink-0" />
          </button>

          <Transition
            enter-active-class="transition duration-150 ease-out"
            enter-from-class="opacity-0 -translate-y-1"
            enter-to-class="opacity-100 translate-y-0"
            leave-active-class="transition duration-100 ease-in"
            leave-from-class="opacity-100 translate-y-0"
            leave-to-class="opacity-0 -translate-y-1"
          >
            <div
              v-if="convOpen"
              class="absolute left-0 top-full mt-2 w-80 bg-white dark:bg-slate-900 border border-slate-200 dark:border-border/40 rounded-xl shadow-xl z-50 overflow-hidden max-h-80 flex flex-col"
            >
              <div class="flex items-center gap-2 px-3 py-2.5 border-b border-border/30">
                <UIcon name="i-lucide-search" class="w-4 h-4 text-slate-500 shrink-0" />
                <input
                  ref="convInputRef"
                  v-model="convSearch"
                  type="text"
                  placeholder="Search conversations..."
                  class="flex-1 bg-transparent outline-none text-sm text-slate-700 dark:text-slate-200 placeholder:text-slate-400 dark:placeholder:text-slate-600"
                  @keydown.escape="convOpen = false"
                />
              </div>

              <div class="overflow-y-auto flex-1">
                <button
                  @click="newConversation"
                  class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-sky-500/10 transition-colors border-b border-border/20"
                  :class="!conversationId ? 'bg-sky-500/5' : ''"
                >
                  <UIcon name="i-lucide-plus-circle" class="w-4 h-4 text-sky-400 shrink-0" />
                  <span class="text-sm text-sky-400 font-medium">New conversation</span>
                </button>

                <div v-if="sessionsLoading" class="flex items-center justify-center py-6">
                  <UIcon name="i-lucide-refresh-cw" class="w-5 h-5 text-slate-500 animate-spin" />
                </div>

                <template v-else>
                  <button
                    v-for="session in filteredSessions"
                    :key="session.id"
                    @click="selectSession(session)"
                    class="w-full flex items-start gap-3 px-3 py-2.5 text-left hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors group"
                    :class="conversationId === session.id ? 'bg-slate-100 dark:bg-slate-800/60' : ''"
                  >
                    <UIcon name="i-lucide-message-square" class="w-4 h-4 text-slate-500 group-hover:text-violet-400 shrink-0 mt-0.5" />
                    <div class="min-w-0 flex-1">
                      <div class="text-sm text-slate-600 dark:text-slate-300 group-hover:text-slate-900 dark:group-hover:text-white truncate leading-snug">
                        {{ session.display || 'Untitled' }}
                      </div>
                      <div class="text-[11px] text-slate-600 font-mono mt-0.5">
                        {{ formatConvDate(session.timestamp) }}
                      </div>
                    </div>
                    <UIcon v-if="conversationId === session.id" name="i-lucide-check" class="w-4 h-4 text-sky-400 shrink-0 mt-0.5" />
                  </button>

                  <div v-if="filteredSessions.length === 0 && !sessionsLoading" class="py-6 text-center text-sm text-slate-600">
                    {{ cwd ? 'No conversations for this project' : 'Select a project first' }}
                  </div>
                </template>
              </div>
            </div>
          </Transition>

          <div v-if="convOpen" class="fixed inset-0 z-40" @click="convOpen = false" />
        </div>
      </div>

      <div class="relative shrink-0">
        <button
          @click="openUsage"
          class="w-9 h-9 rounded-lg flex items-center justify-center text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-900 hover:text-slate-700 dark:hover:text-slate-100 transition-colors relative"
          title="Usage"
        >
          <UIcon name="i-lucide-gauge" class="w-5 h-5" />
        </button>

        <Transition name="fade-slide">
          <div
            v-if="usageOpen"
            class="absolute right-0 top-11 w-80 bg-white dark:bg-slate-900 border border-slate-200 dark:border-border/30 rounded-2xl shadow-xl z-50 overflow-hidden"
          >
            <div class="px-4 py-3 border-b border-slate-100 dark:border-border/20 flex items-center gap-2">
              <UIcon name="i-lucide-gauge" class="w-3.5 h-3.5 text-sky-500" />
              <span class="text-xs font-semibold text-slate-700 dark:text-slate-300">Usage</span>
              <span v-if="usageData" class="ml-auto text-[10px] text-slate-400">
                {{ new Date(usageCacheTime).toLocaleTimeString() }}
              </span>
            </div>
            <div v-if="usageLoading" class="px-4 py-6 flex items-center justify-center">
              <UIcon name="i-lucide-loader-circle" class="w-5 h-5 text-slate-400 animate-spin" />
            </div>
            <div v-else-if="usageError" class="px-4 py-6 text-center text-xs text-rose-400">
              {{ usageError }}
            </div>
            <div v-else-if="usageData && !usageData.error" class="p-4 space-y-4 text-xs">
              <div v-if="usageData.five_hour" class="space-y-1.5">
                <div class="flex items-center justify-between">
                  <span class="font-medium text-slate-600 dark:text-slate-300">5 hour</span>
                  <span class="text-slate-400">{{ usagePercent(usageData.five_hour) }}%</span>
                </div>
                <div class="h-2 bg-slate-100 dark:bg-slate-800 rounded-full overflow-hidden">
                  <div
                    class="h-full rounded-full transition-all duration-500"
                    :class="usagePercent(usageData.five_hour) >= 90 ? 'bg-rose-500' : usagePercent(usageData.five_hour) >= 70 ? 'bg-amber-500' : 'bg-sky-500'"
                    :style="{ width: usagePercent(usageData.five_hour) + '%' }"
                  />
                </div>
                <div v-if="usageData.five_hour.resets_at" class="text-[10px] text-slate-400">
                  Resets: {{ formatResetDate(usageData.five_hour.resets_at) }}
                </div>
              </div>

              <div v-if="usageData.seven_day" class="space-y-1.5">
                <div class="flex items-center justify-between">
                  <span class="font-medium text-slate-600 dark:text-slate-300">7 day</span>
                  <span class="text-slate-400">{{ usagePercent(usageData.seven_day) }}%</span>
                </div>
                <div class="h-2 bg-slate-100 dark:bg-slate-800 rounded-full overflow-hidden">
                  <div
                    class="h-full rounded-full transition-all duration-500"
                    :class="usagePercent(usageData.seven_day) >= 90 ? 'bg-rose-500' : usagePercent(usageData.seven_day) >= 70 ? 'bg-amber-500' : 'bg-emerald-500'"
                    :style="{ width: usagePercent(usageData.seven_day) + '%' }"
                  />
                </div>
                <div v-if="usageData.seven_day.resets_at" class="text-[10px] text-slate-400">
                  Resets: {{ formatResetDate(usageData.seven_day.resets_at) }}
                </div>
              </div>

              <div v-if="usageData.extra_usage && usageData.extra_usage.is_enabled" class="space-y-1.5">
                <div class="flex items-center justify-between">
                  <span class="font-medium text-slate-600 dark:text-slate-300">Extra credits</span>
                  <span class="text-slate-400">
                    {{ usageData.extra_usage.used_credits?.toFixed(0) ?? 0 }} / {{ usageData.extra_usage.monthly_limit }}
                  </span>
                </div>
                <div class="h-2 bg-slate-100 dark:bg-slate-800 rounded-full overflow-hidden">
                  <div
                    class="h-full bg-violet-500 rounded-full transition-all duration-500"
                    :style="{ width: ((usageData.extra_usage.used_credits || 0) / usageData.extra_usage.monthly_limit * 100) + '%' }"
                  />
                </div>
              </div>
            </div>
            <div v-else-if="usageData?.error" class="p-4 text-xs text-rose-400">
              {{ usageData.error }}
            </div>
            <div v-else class="px-4 py-6 text-center text-xs text-slate-400">
              Click to load usage
            </div>
          </div>
        </Transition>

        <div v-if="usageOpen" class="fixed inset-0 z-40" @click="usageOpen = false" />
      </div>

      <div class="relative shrink-0">
        <button
          @click="activeSessionsOpen = !activeSessionsOpen"
          class="w-9 h-9 rounded-lg flex items-center justify-center text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-900 hover:text-slate-700 dark:hover:text-slate-100 transition-colors relative"
          :class="activeSessions?.length ? 'text-emerald-500 dark:text-emerald-400' : ''"
          title="Active sessions"
        >
          <UIcon name="i-lucide-activity" class="w-5 h-5" />
          <span
            v-if="activeSessions?.length"
            class="absolute -top-0.5 -right-0.5 w-4 h-4 rounded-full bg-emerald-500 text-white text-[9px] font-bold flex items-center justify-center"
          >{{ activeSessions.length }}</span>
        </button>

        <Transition name="fade-slide">
          <div
            v-if="activeSessionsOpen"
            class="absolute right-0 top-11 w-72 bg-white dark:bg-slate-900 border border-slate-200 dark:border-border/30 rounded-2xl shadow-xl z-50 overflow-hidden"
          >
            <div class="px-4 py-3 border-b border-slate-100 dark:border-border/20 flex items-center gap-2">
              <UIcon name="i-lucide-activity" class="w-3.5 h-3.5 text-emerald-500" />
              <span class="text-xs font-semibold text-slate-700 dark:text-slate-300">Active Sessions</span>
              <span class="ml-auto text-[10px] text-slate-400">{{ activeSessions?.length ?? 0 }} running</span>
            </div>
            <div v-if="!activeSessions?.length" class="px-4 py-6 text-center text-xs text-slate-400">
              No active sessions
            </div>
            <div v-else class="max-h-64 overflow-y-auto divide-y divide-slate-100 dark:divide-border/10">
              <button
                v-for="s in activeSessions"
                :key="s.sessionId"
                @click="emit('restoreSession', s.sessionId, s.cwd); activeSessionsOpen = false"
                class="w-full px-4 py-3 flex flex-col gap-0.5 text-left hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors"
                :class="s.sessionId === props.conversationId ? 'bg-emerald-50/50 dark:bg-emerald-900/10' : ''"
              >
                <div class="flex items-center gap-2">
                  <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse shrink-0" />
                  <span class="text-xs font-mono text-slate-500 dark:text-slate-400 truncate">{{ formatCwd(s.cwd) }}</span>
                  <span v-if="s.sessionId === props.conversationId" class="ml-auto text-[10px] text-emerald-500 shrink-0">active</span>
                </div>
                <div class="flex items-center gap-2 pl-3.5">
                  <span class="text-[10px] text-slate-400 font-mono truncate">{{ s.sessionId.slice(0, 16) }}…</span>
                  <span class="ml-auto text-[10px] text-slate-400 shrink-0">{{ formatAge(s.startedAt) }}</span>
                </div>
              </button>
            </div>
          </div>
        </Transition>

        <div v-if="activeSessionsOpen" class="fixed inset-0 z-40" @click="activeSessionsOpen = false" />
      </div>

      <button
        @click="emit('showGraphClick')"
        class="lg:hidden w-9 h-9 rounded-lg flex items-center justify-center text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-900 hover:text-slate-700 dark:hover:text-slate-100 transition-colors shrink-0"
      >
        <UIcon name="i-lucide-map" class="w-5 h-5" />
      </button>
    </div>

    <div ref="chatContainer" class="flex-1 overflow-y-auto scroll-smooth relative">
      <div v-if="messages.length === 0" class="h-full flex flex-col items-center justify-center px-6">
        <div class="max-w-sm text-center space-y-5">
          <div class="w-16 h-16 rounded-2xl bg-gradient-to-br from-sky-500/10 to-violet-500/10 border border-sky-500/10 flex items-center justify-center mx-auto">
            <UIcon name="i-lucide-messages-square" class="w-8 h-8 text-sky-400/60" />
          </div>
          <div class="space-y-2">
            <h3 class="text-foreground font-heading font-semibold">Start a conversation</h3>
            <p class="text-sm text-slate-500 leading-relaxed">Set a working directory and describe what you want to build, fix, or explore.</p>
          </div>
        </div>
      </div>

      <div v-else class="p-4 space-y-6 pb-24">
        <div
          v-for="msg in messages"
          :key="msg.id"
          class="flex flex-col"
          :class="msg.role === 'user' ? 'items-end' : 'items-start'"
        >
          <div class="flex flex-col min-w-0 w-full" :class="msg.role === 'user' ? 'items-end' : 'items-start'">
            <div
              v-if="msg.role === 'user'"
              class="px-4 py-2.5 rounded-xl text-sm whitespace-pre-wrap leading-relaxed text-slate-600 dark:text-slate-300 border border-slate-200 dark:border-border/60 max-w-[85%]"
            >
              {{ msg.content }}
            </div>

            <div v-else class="space-y-3 w-full max-w-[95%]">
              <div
                v-if="msg.content"
                class="text-sm leading-relaxed prose dark:prose-invert prose-sm max-w-none
                  prose-p:my-2 prose-headings:my-3 prose-li:my-1
                  prose-pre:bg-slate-100 dark:prose-pre:bg-slate-900/80 prose-pre:border prose-pre:border-slate-200 dark:prose-pre:border-border/20 prose-pre:rounded-xl
                  prose-code:text-sky-600 dark:prose-code:text-sky-300 prose-code:font-mono prose-code:text-[13px]
                  prose-a:text-sky-400 prose-a:no-underline hover:prose-a:underline
                  prose-strong:text-foreground"
                :class="msg.isError
                  ? 'border border-rose-500/40 rounded-xl px-4 py-3 text-rose-600 dark:text-rose-300'
                  : 'text-slate-700 dark:text-slate-200'"
                v-html="renderMarkdown(msg.content)"
              />

              <div v-if="msg.toolUses && msg.toolUses.length > 0" class="space-y-2">
                <div
                  v-for="tool in msg.toolUses"
                  :key="tool.id"
                  class="rounded-xl border overflow-hidden transition-colors"
                  :class="tool.isError ? 'bg-rose-500/5 border-rose-500/30' : (toolColorClasses[tool.name] || 'bg-card/50 border-border/30')"
                >
                  <div class="flex items-center gap-2.5 px-3 py-2 text-xs font-mono cursor-pointer"
                    :class="tool.isError ? 'text-rose-400' : ''"
                    @click="toggleTool(tool.id)"
                  >
                    <UIcon :name="toolIcons[tool.name] || 'i-lucide-wrench'" class="w-3.5 h-3.5 flex-shrink-0 opacity-60" />
                    <span class="truncate flex-1">{{ toolLabel(tool.name, tool.input) }}</span>
                    <template v-if="!tool.isError && tool.status === 'done' && (tool.name === 'Edit' || tool.name === 'Write')">
                      <span v-if="tool.input?.new_string || tool.input?.content" class="text-emerald-400/80 font-semibold">+{{ (tool.input?.new_string || tool.input?.content || '').split('\n').length }}</span>
                      <span v-if="tool.name === 'Edit' && tool.input?.old_string" class="text-rose-400/80 font-semibold">-{{ tool.input.old_string.split('\n').length }}</span>
                    </template>
                    <UIcon
                      v-if="tool.status === 'done'"
                      :name="tool.isError ? 'i-lucide-x-circle' : 'i-lucide-check-circle-2'"
                      class="w-3.5 h-3.5 flex-shrink-0"
                      :class="tool.isError ? 'text-rose-500/70' : 'text-emerald-500/70'"
                    />
                    <UIcon v-else name="i-lucide-loader-2" class="w-3.5 h-3.5 flex-shrink-0 text-sky-500/50 animate-spin" />
                    <UIcon name="i-lucide-chevron-down" class="w-3.5 h-3.5 flex-shrink-0 opacity-30 transition-transform" :class="isToolExpanded(tool.id) ? 'rotate-180' : ''" />
                  </div>

                  <div v-if="tool.name === 'Bash' && tool.status === 'running' && tool.liveOutput && isToolExpanded(tool.id)" class="border-t border-black/5 dark:border-white/5 max-h-40 overflow-y-auto">
                    <div class="p-3 bg-slate-900/50 dark:bg-slate-950/50 font-mono text-[11px] overflow-x-auto">
                      <pre class="text-slate-400 whitespace-pre-wrap">{{ tool.liveOutput }}</pre>
                    </div>
                  </div>

                  <div v-if="tool.status === 'done' && isToolExpanded(tool.id)" class="border-t border-black/5 dark:border-white/5 max-h-24 overflow-y-auto">
                      <div v-if="tool.name === 'Edit' && (tool.input?.old_string || tool.input?.new_string)" class="p-3 bg-slate-900/50 dark:bg-slate-950/50 font-mono text-[11px] overflow-x-auto">
                        <div v-if="tool.input?.old_string" class="text-rose-400/80 mb-1">--- {{ tool.input.file_path }}</div>
                        <div v-if="tool.input?.new_string" class="text-emerald-400/80">+++ {{ tool.input.file_path }}</div>
                        <div class="mt-2 space-y-0.5">
                          <div v-for="(line, i) in (tool.input.old_string || '').split('\n')" :key="'old-'+i" class="text-rose-400/70">- {{ line }}</div>
                          <div v-for="(line, i) in (tool.input.new_string || '').split('\n')" :key="'new-'+i" class="text-emerald-400/70">+ {{ line }}</div>
                        </div>
                      </div>

                      <div v-else-if="tool.name === 'Write' && tool.input?.content" class="p-3 bg-slate-900/50 dark:bg-slate-950/50 font-mono text-[11px] overflow-x-auto">
                        <div class="text-sky-400/80 mb-1">+++ {{ tool.input.file_path }}</div>
                        <pre class="text-slate-300 whitespace-pre-wrap">{{ tool.input.content.slice(0, 1000) }}{{ tool.input.content.length > 1000 ? '\n... (truncated)' : '' }}</pre>
                      </div>

                      <div v-else-if="tool.name === 'Bash' && tool.result" class="p-3 bg-slate-900/50 dark:bg-slate-950/50 font-mono text-[11px] overflow-x-auto">
                        <pre class="text-slate-300 whitespace-pre-wrap">{{ formatBashOutput(tool.result) }}</pre>
                      </div>

                      <div v-else-if="tool.name === 'Read' && tool.result" class="p-3 bg-slate-900/50 dark:bg-slate-950/50 font-mono text-[11px] overflow-x-auto">
                        <pre class="text-slate-300 whitespace-pre-wrap">{{ tool.result.slice(0, 1500) }}{{ tool.result.length > 1500 ? '\n... (truncated)' : '' }}</pre>
                      </div>

                      <div v-else-if="tool.result" class="p-3 bg-slate-900/50 dark:bg-slate-950/50 font-mono text-[11px] overflow-x-auto">
                        <pre class="text-slate-300 whitespace-pre-wrap">{{ typeof tool.result === 'string' ? tool.result.slice(0, 1500) : JSON.stringify(tool.result, null, 2).slice(0, 1500) }}</pre>
                      </div>

                      <div v-else class="p-3 text-slate-400 text-xs italic">
                        No output
                      </div>
                    </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div v-if="isStreaming && (messages.length === 0 || messages[messages.length - 1]?.role === 'user')" class="flex gap-2 items-center py-2">
          <div class="flex items-center gap-1.5 opacity-40">
            <div class="w-1.5 h-1.5 rounded-full bg-sky-400 animate-bounce" style="animation-delay: 0ms"></div>
            <div class="w-1.5 h-1.5 rounded-full bg-sky-400 animate-bounce" style="animation-delay: 150ms"></div>
            <div class="w-1.5 h-1.5 rounded-full bg-sky-400 animate-bounce" style="animation-delay: 300ms"></div>
          </div>
        </div>

        <div v-if="pendingPermission?.toolName === 'ExitPlanMode'" class="bg-emerald-500/5 border-2 border-emerald-500/30 rounded-2xl p-5 mt-4 space-y-4 shadow-[0_0_25px_rgba(16,185,129,0.1)]">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2 text-emerald-400">
              <UIcon name="i-lucide-clipboard-check" class="w-5 h-5" />
              <span class="text-sm font-bold uppercase tracking-wider">Plan Ready</span>
            </div>
            <button
              v-if="planContent"
              @click="planExpanded = !planExpanded"
              class="flex items-center gap-1.5 text-xs text-emerald-400/70 hover:text-emerald-300 transition-colors"
            >
              <span>{{ planExpanded ? 'Collapse' : 'Show plan' }}</span>
              <UIcon name="i-lucide-chevron-down" class="w-3.5 h-3.5 transition-transform duration-200" :class="{ 'rotate-180': planExpanded }" />
            </button>
          </div>

          <Transition
            enter-active-class="transition-all duration-300 ease-out"
            enter-from-class="max-h-0 opacity-0"
            enter-to-class="max-h-[60vh] opacity-100"
            leave-active-class="transition-all duration-200 ease-in"
            leave-from-class="max-h-[60vh] opacity-100"
            leave-to-class="max-h-0 opacity-0"
          >
            <div
              v-if="planContent && planExpanded"
              class="overflow-hidden"
            >
              <div
                class="text-sm leading-relaxed overflow-y-auto max-h-[50vh] rounded-xl bg-slate-50 dark:bg-slate-900/60 border border-emerald-500/15 p-4 custom-scrollbar
                  prose dark:prose-invert prose-sm max-w-none
                  prose-p:my-2 prose-headings:my-3 prose-li:my-1
                  prose-pre:bg-slate-100 dark:prose-pre:bg-slate-900/80 prose-pre:border prose-pre:border-slate-200 dark:prose-pre:border-border/20 prose-pre:rounded-xl
                  prose-code:text-emerald-600 dark:prose-code:text-emerald-300 prose-code:font-mono prose-code:text-[12px]
                  prose-a:text-emerald-400 prose-a:no-underline hover:prose-a:underline
                  prose-strong:text-foreground"
                v-html="renderMarkdown(planContent)"
              />
            </div>
          </Transition>

          <div v-if="!planContent" class="text-sm text-slate-600 dark:text-slate-300">
            Claude has finished planning. Review the plan above and approve to start implementation.
          </div>

          <div class="flex gap-3">
            <UButton color="green" @click="emit('permissionResponse', true)" class="font-bold">Approve & Implement</UButton>
            <UButton color="red" variant="outline" @click="emit('permissionResponse', false)" class="font-bold">Reject</UButton>
          </div>
        </div>

        <div v-else-if="activeQuestion" class="bg-sky-500/5 border-2 border-sky-500/30 rounded-2xl p-5 mt-4 space-y-8 shadow-[0_0_25px_rgba(14,165,233,0.1)]">
          <div v-for="(q, idx) in (activeQuestion.questions || (Array.isArray(activeQuestion) ? activeQuestion : []))" :key="idx" class="space-y-4">
            <div class="flex items-center gap-2 text-sky-400">
              <UIcon name="i-lucide-help-circle" class="w-5 h-5" />
              <span class="text-sm font-bold uppercase tracking-wider">{{ q.header || 'Action Required' }}</span>
            </div>

            <div class="text-[15px] font-medium text-slate-800 dark:text-slate-100 leading-snug">{{ q.question }}</div>

            <div v-if="q.options && q.options.length > 0" class="grid grid-cols-1 gap-2.5">
              <button
                v-for="opt in q.options"
                :key="opt.label"
                @click="selectAnswer(idx, opt.label)"
                class="group px-4 py-3 rounded-xl border transition-all text-left w-full"
                :class="selectedAnswers[idx] === opt.label
                  ? 'bg-sky-500/20 border-sky-500 shadow-[0_0_15px_rgba(14,165,233,0.15)]'
                  : 'bg-slate-50 dark:bg-slate-900/50 border-slate-200 dark:border-border/60 hover:border-sky-500/50 hover:bg-sky-500/10'"
              >
                <div class="flex items-center justify-between">
                  <div class="font-bold text-sm transition-colors" :class="selectedAnswers[idx] === opt.label ? 'text-sky-300' : 'text-slate-700 dark:text-slate-200 group-hover:text-sky-300'">
                    {{ opt.label }}
                  </div>
                  <UIcon v-if="selectedAnswers[idx] === opt.label" name="i-lucide-check-circle-2" class="w-5 h-5 text-sky-400" />
                  <UIcon v-else name="i-lucide-chevron-right" class="w-4 h-4 text-slate-600 group-hover:text-sky-400" />
                </div>
                <div v-if="opt.description" class="text-xs mt-1 leading-normal" :class="selectedAnswers[idx] === opt.label ? 'text-sky-400/70' : 'text-slate-500'">
                  {{ opt.description }}
                </div>
              </button>
            </div>

            <div class="space-y-2 pt-2">
              <div class="text-[10px] text-slate-500 uppercase font-bold tracking-widest px-1">Custom Response</div>
              <UInput
                :value="selectedAnswers[idx]"
                @input="selectAnswer(idx, ($event.target as HTMLInputElement).value)"
                placeholder="Type your own answer..."
                size="sm"
                :ui="{
                  base: '!rounded-xl !bg-slate-50 dark:!bg-slate-900/80 !border-slate-200 dark:!border-border/40 focus:!border-sky-500/40 focus:!ring-sky-500/10',
                }"
              />
            </div>
          </div>

          <div class="pt-4 border-t border-sky-500/20">
            <UButton
              block
              size="lg"
              color="sky"
              variant="solid"
              class="rounded-xl font-bold"
              @click="submitAllAnswers"
            >
              Submit All Answers
            </UButton>
          </div>
        </div>

        <div v-else-if="pendingPermission?.toolName === 'Bash'" class="bg-amber-500/5 border-2 border-amber-500/30 rounded-2xl p-5 mt-4 space-y-4 shadow-[0_0_25px_rgba(245,158,11,0.1)]">
          <div class="flex items-center gap-2 text-amber-400">
            <UIcon name="i-lucide-terminal" class="w-5 h-5" />
            <span class="text-sm font-bold uppercase tracking-wider">Run Command</span>
          </div>

          <div v-if="pendingPermission.input?.description" class="text-xs text-slate-500 dark:text-slate-400 italic">
            {{ pendingPermission.input.description }}
          </div>

          <div class="bg-slate-900 rounded-xl p-4 border border-slate-700/50">
            <div class="flex items-center gap-2 mb-2 text-slate-500 text-[10px] font-bold uppercase tracking-widest">
              <UIcon name="i-lucide-chevron-right" class="w-3 h-3" />
              <span>{{ pendingPermission.input?.cwd || '/tmp' }}</span>
            </div>
            <pre class="text-sm text-emerald-300 font-mono whitespace-pre-wrap break-all">{{ pendingPermission.input?.command }}</pre>
          </div>

          <div class="flex gap-3">
            <UButton color="green" @click="emit('permissionResponse', true)" class="font-bold">Run</UButton>
            <UButton color="red" variant="outline" @click="emit('permissionResponse', false)" class="font-bold">Deny</UButton>
          </div>
        </div>

        <div v-else-if="pendingPermission?.toolName === 'Write'" class="bg-amber-500/5 border-2 border-amber-500/30 rounded-2xl p-5 mt-4 space-y-4 shadow-[0_0_25px_rgba(245,158,11,0.1)]">
          <div class="flex items-center gap-2 text-amber-400">
            <UIcon name="i-lucide-file-plus" class="w-5 h-5" />
            <span class="text-sm font-bold uppercase tracking-wider">Create File</span>
          </div>

          <div class="flex items-center gap-2 text-sm text-slate-300">
            <UIcon name="i-lucide-file" class="w-4 h-4 text-sky-400/70 shrink-0" />
            <span class="font-mono text-sky-300/90 truncate">{{ pendingPermission.input?.file_path }}</span>
          </div>

          <div class="bg-slate-900 rounded-xl border border-slate-700/50 overflow-hidden">
            <div class="flex items-center gap-2 px-4 py-2 border-b border-slate-700/50 text-[10px] text-slate-500 font-bold uppercase tracking-widest">
              <UIcon name="i-lucide-code" class="w-3 h-3" />
              Content
            </div>
            <pre class="text-xs text-slate-200 font-mono p-4 overflow-x-auto max-h-80 whitespace-pre-wrap">{{ pendingPermission.input?.content }}</pre>
          </div>

          <div class="flex gap-3">
            <UButton color="green" @click="emit('permissionResponse', true)" class="font-bold">Create</UButton>
            <UButton color="red" variant="outline" @click="emit('permissionResponse', false)" class="font-bold">Deny</UButton>
          </div>
        </div>

        <div v-else-if="pendingPermission?.toolName === 'Edit'" class="bg-amber-500/5 border-2 border-amber-500/30 rounded-2xl p-5 mt-4 space-y-4 shadow-[0_0_25px_rgba(245,158,11,0.1)]">
          <div class="flex items-center gap-2 text-amber-400">
            <UIcon name="i-lucide-file-pen-line" class="w-5 h-5" />
            <span class="text-sm font-bold uppercase tracking-wider">Edit File</span>
          </div>

          <div class="flex items-center gap-2 text-sm text-slate-300">
            <UIcon name="i-lucide-file" class="w-4 h-4 text-sky-400/70 shrink-0" />
            <span class="font-mono text-sky-300/90 truncate">{{ pendingPermission.input?.file_path }}</span>
          </div>

          <div class="bg-slate-900 rounded-xl border border-slate-700/50 overflow-hidden">
            <div class="flex items-center gap-2 px-4 py-2 border-b border-slate-700/50 text-[10px] text-slate-400 font-bold uppercase tracking-widest">
              <UIcon name="i-lucide-git-compare" class="w-3 h-3" />
              Diff
            </div>
            <pre class="text-xs font-mono p-4 overflow-x-auto max-h-60 whitespace-pre-wrap"><template v-for="line in editDiffLines" :key="line.key"><span :class="line.cls">{{ line.text }}
</span></template></pre>
          </div>

          <div class="flex gap-3">
            <UButton color="green" @click="emit('permissionResponse', true)" class="font-bold">Apply</UButton>
            <UButton color="red" variant="outline" @click="emit('permissionResponse', false)" class="font-bold">Deny</UButton>
          </div>
        </div>

        <div v-else-if="pendingPermission" class="bg-amber-500/5 border-2 border-amber-500/30 rounded-2xl p-5 mt-4 space-y-4 shadow-[0_0_25px_rgba(245,158,11,0.1)]">
          <div class="flex items-center gap-2 text-amber-400">
            <UIcon name="i-lucide-shield-alert" class="w-5 h-5" />
            <span class="text-sm font-bold uppercase tracking-wider">Permission Required</span>
          </div>

          <div class="text-sm text-slate-600 dark:text-slate-300">
            Claude wants to use <strong class="text-slate-800 dark:text-slate-100">{{ pendingPermission.toolName }}</strong>
          </div>

          <pre class="text-xs bg-slate-100 dark:bg-slate-900/60 rounded-lg p-3 overflow-x-auto text-slate-600 dark:text-slate-300 border border-slate-200 dark:border-border/20">{{ JSON.stringify(pendingPermission.input, null, 2) }}</pre>

          <div class="flex gap-3">
            <UButton color="green" @click="emit('permissionResponse', true)" class="font-bold">Allow</UButton>
            <UButton color="red" variant="outline" @click="emit('permissionResponse', false)" class="font-bold">Deny</UButton>
          </div>
        </div>

      </div>
    </div>

    <div class="border-t border-border/40 bg-card/30 shrink-0 pb-[env(safe-area-inset-bottom)] relative">

      <div
        class="absolute top-0 left-0 right-0 transform -translate-y-full h-10 flex items-center px-6 pointer-events-none z-10 bg-gradient-to-t from-background/90 via-background/40 to-transparent"
      >
        <Transition name="fade-slide" mode="out-in">
          <div v-if="isStreaming" class="flex items-center gap-3">
            <div class="relative w-3.5 h-3.5">
              <div class="absolute inset-0 border-2 border-sky-500/20 rounded-full"></div>
              <div class="absolute inset-0 border-2 border-t-sky-500 rounded-full animate-spin"></div>
            </div>
            <Transition name="fade-slide" mode="out-in">
              <span :key="activeTodoText ?? currentVerb" class="text-[11px] font-bold uppercase tracking-tight text-slate-500 italic drop-shadow-sm">
                {{ activeTodoText ?? currentVerb }}...
              </span>
            </Transition>
          </div>
        </Transition>
      </div>

      <Transition
        enter-active-class="transition-all duration-200 ease-out"
        enter-from-class="opacity-0 -translate-y-1 max-h-0"
        enter-to-class="opacity-100 translate-y-0 max-h-80"
        leave-active-class="transition-all duration-150 ease-in"
        leave-from-class="opacity-100 translate-y-0 max-h-80"
        leave-to-class="opacity-0 -translate-y-1 max-h-0"
      >
        <div v-if="isStreaming && currentTodos.length > 0" class="border-b border-border/20">
          <div class="px-3 pt-2 pb-1.5 space-y-0.5 overflow-y-auto custom-scrollbar" style="max-height: 5.5rem">
            <div
              v-for="todo in sortedTodos"
              :key="todo.content"
              class="flex items-center gap-2 px-2.5 py-1.5 rounded-lg text-xs transition-colors"
              :class="todo.status === 'in_progress' ? 'bg-sky-500/8' : ''"
            >
              <UIcon
                v-if="todo.status === 'completed'"
                name="i-lucide-check-circle-2"
                class="w-3.5 h-3.5 flex-shrink-0 text-emerald-500/70"
              />
              <div v-else class="w-3.5 h-3.5 flex-shrink-0 rounded-full border"
                :class="todo.status === 'in_progress' ? 'border-sky-400/60' : 'border-slate-500/30'"
              ></div>
              <span
                class="flex-1 truncate font-mono"
                :class="todo.status === 'completed'
                  ? 'text-slate-500 line-through'
                  : todo.status === 'in_progress'
                    ? 'text-sky-300'
                    : 'text-slate-500'"
              >{{ todo.status === 'in_progress' ? todo.activeForm : todo.content }}</span>
            </div>
          </div>
        </div>
      </Transition>

      <div class="px-3 sm:px-4 pt-3 pb-2 relative">
        <button
          @click="openCwdPicker"
          class="w-full flex items-center gap-2 rounded-xl px-3 py-2.5 text-sm sm:text-xs transition-colors border text-left group"
          :class="cwdError || pathNotFound
            ? 'bg-rose-500/5 border-rose-500/30'
            : cwd ? 'bg-emerald-500/5 border-emerald-500/15' : 'bg-slate-50 dark:bg-slate-900/30 border-slate-200 dark:border-border/40'"
        >
          <UIcon
            name="i-lucide-folder"
            class="w-4 h-4 flex-shrink-0"
            :class="cwdError || pathNotFound ? 'text-rose-400' : cwd ? 'text-emerald-400/70' : 'text-slate-400'"
          />
          <span class="flex-1 truncate" :class="cwd && !pathNotFound ? 'text-slate-600 dark:text-slate-300' : 'text-slate-400 dark:text-slate-600'">
            {{ cwd || 'Select working directory...' }}
          </span>
          <button v-if="pathNotFound" @click.stop="handleCreateFolder" title="Create folder" class="flex items-center gap-1 text-rose-500 hover:text-rose-600 dark:text-rose-400 dark:hover:text-rose-300 bg-rose-500/10 hover:bg-rose-500/20 px-2 py-0.5 rounded-md transition-colors">
            <UIcon name="i-lucide-folder-plus" class="w-3.5 h-3.5" />
            <span class="text-[10px] font-medium uppercase tracking-wider">Create</span>
          </button>
          <span v-else-if="cwdError" class="text-rose-400 flex-shrink-0 text-xs">required</span>
          <UIcon v-else name="i-lucide-chevrons-up-down" class="w-3.5 h-3.5 text-slate-500 flex-shrink-0" />
        </button>

        <Transition
          enter-active-class="transition duration-150 ease-out"
          enter-from-class="opacity-0 translate-y-1"
          enter-to-class="opacity-100 translate-y-0"
          leave-active-class="transition duration-100 ease-in"
          leave-from-class="opacity-100 translate-y-0"
          leave-to-class="opacity-0 translate-y-1"
        >
          <div
            v-if="cwdOpen"
            class="absolute left-3 right-3 bottom-full mb-1 bg-white dark:bg-slate-900 border border-slate-200 dark:border-border/40 rounded-xl shadow-xl z-50 overflow-hidden max-h-72 flex flex-col"
          >
            <div class="flex items-center gap-2 px-3 py-2.5 border-b border-border/30">
              <UIcon name="i-lucide-search" class="w-4 h-4 text-slate-500 flex-shrink-0" />
              <input
                ref="cwdInputRef"
                v-model="cwdSearch"
                type="text"
                placeholder="Search projects or type a path..."
                class="flex-1 bg-transparent outline-none text-sm text-slate-700 dark:text-slate-200 placeholder:text-slate-400 dark:placeholder:text-slate-600"
                @keydown="handleCwdKeydown"
              />
            </div>

            <div class="overflow-y-auto flex-1">
              <div v-if="projectsLoading" class="flex items-center justify-center py-6">
                <UIcon name="i-lucide-refresh-cw" class="w-5 h-5 text-slate-500 animate-spin" />
              </div>

              <template v-else>
                <button
                  v-if="cwdSearch.trim() && !filteredProjects.find(p => p.path === cwdSearch.trim())"
                  @click="confirmCustomPath"
                  class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-sky-500/10 transition-colors border-b border-border/20"
                >
                  <UIcon name="i-lucide-plus-circle" class="w-4 h-4 text-sky-400 flex-shrink-0" />
                  <div class="min-w-0 flex-1">
                    <div class="text-xs text-sky-400 font-medium">Use custom path</div>
                    <div class="text-[11px] text-slate-400 font-mono truncate">{{ cwdSearch.trim() }}</div>
                  </div>
                  <kbd class="text-[10px] text-slate-600 border border-border/30 rounded px-1 py-0.5">Enter</kbd>
                </button>

                <button
                  v-for="project in filteredProjects"
                  :key="project.encodedDir"
                  @click="selectCwd(project.path)"
                  class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors group"
                  :class="cwd === project.path ? 'bg-slate-100 dark:bg-slate-800/60' : ''"
                >
                  <UIcon name="i-lucide-folder" class="w-4 h-4 text-slate-500 group-hover:text-sky-400 flex-shrink-0" />
                  <div class="min-w-0 flex-1">
                    <div class="text-sm text-slate-600 dark:text-slate-200 group-hover:text-slate-900 dark:group-hover:text-white truncate">{{ project.name }}</div>
                    <div class="text-[11px] text-slate-600 font-mono truncate">{{ project.path }}</div>
                  </div>
                  <UIcon v-if="cwd === project.path" name="i-lucide-check" class="w-4 h-4 text-sky-400 flex-shrink-0" />
                </button>

                <div v-if="filteredProjects.length === 0 && !cwdSearch.trim()" class="py-6 text-center text-sm text-slate-600">
                  No projects found
                </div>
              </template>
            </div>
          </div>
        </Transition>

        <div v-if="cwdOpen" class="fixed inset-0 z-40" @click="cwdOpen = false" />
      </div>

      <div class="px-3 sm:px-4 pb-3 sm:pb-4 pt-1 space-y-2">
        <div class="relative">
          <textarea
            ref="promptInputRef"
            v-model="prompt"
            :placeholder="cwd ? 'Ask Claude anything... (Enter to send, Shift+Enter for new line)' : 'Set a working directory first...'"
            rows="2"
            class="w-full px-4 py-3 pr-16 pb-3 resize-none rounded-xl bg-card/50 border border-border/40 focus:border-sky-500/40 focus:ring-sky-500/10 text-sm sm:text-sm max-h-48 overflow-y-auto outline-none custom-scrollbar"
            @keydown="handleEnterKey"
          />
          <button
            v-if="isStreaming"
            type="button"
            @click="emit('stop')"
            class="absolute right-3 bottom-3 w-9 h-9 sm:w-8 sm:h-8 rounded-lg flex items-center justify-center transition-all bg-rose-500 text-white hover:bg-rose-400 shadow-sm shadow-rose-500/20"
          >
            <UIcon name="i-lucide-square" class="w-4 h-4" />
          </button>
          <button
            v-else
            type="button"
            :disabled="!canSubmit"
            @click="onSubmit"
            class="absolute right-3 bottom-3 w-9 h-9 sm:w-8 sm:h-8 rounded-lg flex items-center justify-center transition-all"
            :class="canSubmit
              ? 'bg-sky-500 text-white hover:bg-sky-400 shadow-sm shadow-sky-500/20'
              : 'bg-slate-200 dark:bg-slate-800 text-slate-400 dark:text-slate-600 cursor-not-allowed'"
          >
            <UIcon name="i-lucide-arrow-up" class="w-4 h-4" />
          </button>
        </div>

        <div class="flex items-center justify-between px-1">
          <div class="flex items-center gap-2 relative">
            <button
              @click="modelDropdownOpen = !modelDropdownOpen"
              class="flex items-center gap-2 text-[11px] font-bold uppercase tracking-tight transition-all py-1.5 px-3 rounded-lg border"
              :class="modelDropdownOpen
                ? 'bg-sky-500/20 border-sky-500/50 text-sky-300'
                : 'bg-slate-50 dark:bg-slate-900/50 border-slate-200 dark:border-border/40 text-slate-500 dark:text-slate-400 hover:border-slate-400 dark:hover:border-slate-500 hover:text-slate-700 dark:hover:text-slate-200'"
            >
              <UIcon :name="models.find(m => m.value === activeModel)?.icon ?? 'i-lucide-zap'" class="w-3.5 h-3.5" :class="modelDropdownOpen ? 'text-sky-400' : 'text-sky-500/70'" />
              {{ models.find(m => m.value === activeModel)?.label ?? 'Model' }}
              <UIcon name="i-lucide-chevron-down" class="w-3 h-3 transition-transform duration-200" :class="{ 'rotate-180': modelDropdownOpen, 'opacity-50': !modelDropdownOpen }" />
            </button>

            <Transition
              enter-active-class="transition duration-150 ease-out"
              enter-from-class="opacity-0 translate-y-1"
              enter-to-class="opacity-100 translate-y-0"
              leave-active-class="transition duration-100 ease-in"
              leave-from-class="opacity-100 translate-y-0"
              leave-to-class="opacity-0 translate-y-1"
            >
              <div
                v-if="modelDropdownOpen"
                class="absolute left-0 bottom-full mb-2 w-56 bg-white dark:bg-slate-900 border border-slate-200 dark:border-border/40 rounded-xl shadow-xl z-50 overflow-hidden"
              >
                <div class="px-2.5 py-2 border-b border-border/20">
                  <div class="text-[9px] font-bold uppercase tracking-widest text-slate-400 px-1">Model</div>
                </div>
                <div class="py-1">
                  <button
                    v-for="m in models"
                    :key="m.value"
                    @click="selectModel(m.value)"
                    class="w-full flex items-center gap-2.5 px-3 py-2 text-left text-xs transition-colors hover:bg-slate-100 dark:hover:bg-slate-800"
                    :class="activeModel === m.value ? 'text-sky-400 font-bold' : 'text-slate-600 dark:text-slate-300'"
                  >
                    <UIcon :name="m.icon" class="w-3.5 h-3.5" :class="activeModel === m.value ? 'text-sky-400' : 'text-slate-400'" />
                    {{ m.label }}
                    <UIcon v-if="activeModel === m.value" name="i-lucide-check" class="w-3.5 h-3.5 ml-auto text-sky-400" />
                  </button>
                </div>

                <template v-if="profileEntries.length > 0">
                  <div class="px-2.5 py-2 border-t border-border/20">
                    <div class="text-[9px] font-bold uppercase tracking-widest text-slate-400 px-1">Profile</div>
                  </div>
                  <div class="py-1">
                    <button
                      v-for="[id, profile] in profileEntries"
                      :key="id"
                      @click="setActiveProfile(id)"
                      class="w-full flex items-center gap-2.5 px-3 py-2 text-left text-xs transition-colors hover:bg-slate-100 dark:hover:bg-slate-800"
                      :class="profilesConfig.active === id ? 'text-violet-400 font-bold' : 'text-slate-600 dark:text-slate-300'"
                    >
                      <UIcon name="i-lucide-user" class="w-3.5 h-3.5" :class="profilesConfig.active === id ? 'text-violet-400' : 'text-slate-400'" />
                      {{ profile.name || id }}
                      <span class="ml-auto text-[9px] font-mono text-slate-500">{{ Object.keys(profile.env).length }} vars</span>
                      <UIcon v-if="profilesConfig.active === id" name="i-lucide-check" class="w-3.5 h-3.5 text-violet-400" />
                    </button>
                  </div>
                </template>
              </div>
            </Transition>

            <div v-if="modelDropdownOpen" class="fixed inset-0 z-40" @click="modelDropdownOpen = false" />
          </div>

          <div class="flex items-center gap-2">
            <USelectMenu
              :model-value="activeMode"
              @update:model-value="emit('update:activeMode', $event)"
              :options="modes"
              value-attribute="value"
              :ui-menu="{
                background: 'bg-white dark:bg-slate-900',
                border: 'border-border/40',
                rounded: 'rounded-xl'
              }"
            >
              <template #default="{ open }">
                <button
                  class="flex items-center gap-2 text-[11px] font-bold uppercase tracking-tight transition-all py-1.5 px-3 rounded-lg border"
                  :class="open
                    ? 'bg-emerald-500/20 border-emerald-500/50 text-emerald-300'
                    : 'bg-slate-50 dark:bg-slate-900/50 border-slate-200 dark:border-border/40 text-slate-500 dark:text-slate-400 hover:border-slate-400 dark:hover:border-slate-500 hover:text-slate-700 dark:hover:text-slate-200'"
                >
                  <UIcon :name="modes.find(m => m.value === activeMode)?.icon ?? 'i-lucide-terminal'" class="w-3.5 h-3.5" :class="open ? 'text-emerald-400' : 'text-emerald-500/70'" />
                  {{ modes.find(m => m.value === activeMode)?.label ?? 'Mode' }}
                  <UIcon name="i-lucide-chevron-down" class="w-3 h-3 transition-transform duration-200" :class="{ 'rotate-180': open, 'opacity-50': !open }" />
                </button>
              </template>
            </USelectMenu>

            <USelectMenu
              :model-value="permissionStyle"
              @update:model-value="emit('update:permissionStyle', $event)"
              :options="permStyles"
              value-attribute="value"
              :ui-menu="{
                background: 'bg-white dark:bg-slate-900',
                border: 'border-border/40',
                rounded: 'rounded-xl'
              }"
            >
              <template #default="{ open }">
                <button
                  class="flex items-center gap-2 text-[11px] font-bold uppercase tracking-tight transition-all py-1.5 px-3 rounded-lg border"
                  :class="permissionStyle === 'yolo'
                    ? (open ? 'bg-rose-500/20 border-rose-500/50 text-rose-300' : 'bg-rose-500/10 border-rose-500/30 text-rose-400 hover:border-rose-400 hover:text-rose-300')
                    : (open ? 'bg-violet-500/20 border-violet-500/50 text-violet-300' : 'bg-slate-50 dark:bg-slate-900/50 border-slate-200 dark:border-border/40 text-slate-500 dark:text-slate-400 hover:border-slate-400 dark:hover:border-slate-500 hover:text-slate-700 dark:hover:text-slate-200')"
                >
                  <UIcon :name="permStyles.find(m => m.value === permissionStyle)?.icon ?? 'i-lucide-shield-question'" class="w-3.5 h-3.5" :class="permissionStyle === 'yolo' ? 'text-rose-400' : (open ? 'text-violet-400' : 'text-violet-500/70')" />
                  {{ permStyles.find(m => m.value === permissionStyle)?.label ?? 'Perms' }}
                  <UIcon name="i-lucide-chevron-down" class="w-3 h-3 transition-transform duration-200" :class="{ 'rotate-180': open, 'opacity-50': !open }" />
                </button>
              </template>
            </USelectMenu>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
:deep(.thinking-text) {
  display: block;
  opacity: 0.5;
  font-style: italic;
  margin-bottom: 0.75rem;
}

.fade-slide-enter-active,
.fade-slide-leave-active {
  transition: all 0.3s ease;
}

.fade-slide-enter-from {
  opacity: 0;
  transform: translateY(5px);
}

.fade-slide-leave-to {
  opacity: 0;
  transform: translateY(-5px);
}

.expand-enter-active,
.expand-leave-active {
  transition: all 0.2s ease-out;
  overflow: hidden;
}

.expand-enter-from,
.expand-leave-to {
  max-height: 0;
  opacity: 0;
}

.expand-enter-to,
.expand-leave-from {
  max-height: 500px;
  opacity: 1;
}
</style>
