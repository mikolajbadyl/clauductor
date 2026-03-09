<script setup lang="ts">
import { renderMarkdown } from '~/utils/markdown'
import { backendUrl } from '~/utils/api'

const isSidebarOpen = ref(false)
const activeTab = ref('clauductor')
const claudeCodeChangelog = ref('')
const loading = ref(false)
const error = ref<string | null>(null)

const installedVersion = ref('')
const latestVersion = ref('')
const updateAvailable = computed(() => {
  if (!installedVersion.value || !latestVersion.value) return false
  return compareVersions(latestVersion.value, installedVersion.value) > 0
})

// Update modal state
const updateModalOpen = ref(false)
const updateLogs = ref<string[]>([])
const updateRunning = ref(false)
const updateDone = ref(false)
const updateError = ref('')
const logsContainer = ref<HTMLElement | null>(null)

const tabs = [
  { id: 'clauductor', label: 'Clauductor', icon: 'i-lucide-sparkles' },
  { id: 'claude-code', label: 'Claude Code', icon: 'i-lucide-terminal' },
]

function parseVersion(str: string): string {
  const match = str.match(/(\d+\.\d+\.\d+(?:-[\w.]+)?)/)
  return match ? match[1] : ''
}

function compareVersions(a: string, b: string): number {
  const pa = a.split('.').map(Number)
  const pb = b.split('.').map(Number)
  for (let i = 0; i < 3; i++) {
    if ((pa[i] || 0) > (pb[i] || 0)) return 1
    if ((pa[i] || 0) < (pb[i] || 0)) return -1
  }
  return 0
}

async function fetchInstalledVersion() {
  try {
    const res = await fetch(backendUrl('/api/claude-version'))
    if (res.ok) {
      const data = await res.json()
      installedVersion.value = parseVersion(data.version || '')
    }
  } catch {
    // ignore
  }
}

async function fetchClaudeCodeChangelog() {
  if (claudeCodeChangelog.value) return

  loading.value = true
  error.value = null
  try {
    const response = await fetch('https://raw.githubusercontent.com/anthropics/claude-code/main/CHANGELOG.md')
    if (!response.ok) throw new Error('Failed to fetch changelog')
    const text = await response.text()
    claudeCodeChangelog.value = text
    const match = text.match(/##\s+v?(\d+\.\d+\.\d+(?:-[\w.]+)?)/)
    if (match) latestVersion.value = match[1]
  } catch (e) {
    console.error('Error fetching Claude Code changelog:', e)
    error.value = 'Could not load release notes from GitHub.'
  } finally {
    loading.value = false
  }
}

const renderedClaudeCode = computed(() => {
  if (!claudeCodeChangelog.value) return ''
  return renderMarkdown(claudeCodeChangelog.value)
})

async function startUpdate() {
  updateModalOpen.value = true
  updateLogs.value = []
  updateRunning.value = true
  updateDone.value = false
  updateError.value = ''

  try {
    const res = await fetch(backendUrl('/api/claude-update'), { method: 'POST' })
    if (!res.ok || !res.body) {
      updateError.value = 'Failed to start update.'
      updateRunning.value = false
      return
    }

    const reader = res.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''

    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })
      const parts = buffer.split('\n\n')
      buffer = parts.pop() ?? ''

      for (const part of parts) {
        const eventMatch = part.match(/^event: (\w+)/)
        const dataMatch = part.match(/^data: (.*)$/m)
        const eventType = eventMatch?.[1]
        const data = dataMatch?.[1] ?? ''

        if (eventType === 'log') {
          updateLogs.value.push(data)
          await nextTick()
          if (logsContainer.value) {
            logsContainer.value.scrollTop = logsContainer.value.scrollHeight
          }
        } else if (eventType === 'done') {
          updateDone.value = true
          updateRunning.value = false
          // Refresh installed version
          await fetchInstalledVersion()
        } else if (eventType === 'error') {
          updateError.value = data
          updateRunning.value = false
        }
      }
    }
  } catch (e: any) {
    updateError.value = e?.message || 'Connection error.'
    updateRunning.value = false
  }
}

function closeUpdateModal() {
  updateModalOpen.value = false
}

watch(activeTab, (newTab) => {
  if (newTab === 'claude-code') {
    fetchClaudeCodeChangelog()
    fetchInstalledVersion()
  }
})

onMounted(() => {
  if (activeTab.value === 'claude-code') {
    fetchClaudeCodeChangelog()
    fetchInstalledVersion()
  }
})
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

    <!-- Update modal -->
    <Teleport to="body">
      <Transition name="modal">
        <div
          v-if="updateModalOpen"
          class="fixed inset-0 z-50 flex items-center justify-center p-4"
        >
          <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" @click="!updateRunning && closeUpdateModal()" />
          <div class="relative w-full max-w-2xl bg-slate-900 border border-slate-700 rounded-2xl shadow-2xl flex flex-col overflow-hidden">
            <div class="flex items-center justify-between px-5 py-4 border-b border-slate-700/60 shrink-0">
              <div class="flex items-center gap-2.5">
                <div class="w-2 h-2 rounded-full" :class="updateRunning ? 'bg-sky-400 animate-pulse' : updateError ? 'bg-red-400' : 'bg-emerald-400'" />
                <span class="text-sm font-semibold text-slate-100">
                  {{ updateRunning ? 'Updating Claude Code...' : updateError ? 'Update failed' : 'Update complete' }}
                </span>
              </div>
              <button
                v-if="!updateRunning"
                @click="closeUpdateModal"
                class="w-7 h-7 rounded-md flex items-center justify-center text-slate-400 hover:text-slate-100 hover:bg-slate-700 transition-colors"
              >
                <UIcon name="i-lucide-x" class="w-4 h-4" />
              </button>
            </div>

            <div
              ref="logsContainer"
              class="flex-1 overflow-y-auto font-mono text-[12.5px] leading-relaxed p-5 space-y-0.5 min-h-64 max-h-96"
            >
              <div
                v-for="(line, i) in updateLogs"
                :key="i"
                class="text-slate-300 whitespace-pre-wrap break-all"
              >{{ line || '\u00A0' }}</div>

              <div v-if="updateRunning" class="flex items-center gap-2 text-slate-500 mt-2">
                <UIcon name="i-lucide-loader-circle" class="w-3.5 h-3.5 animate-spin" />
                <span>Running...</span>
              </div>

              <div v-if="updateError" class="flex items-center gap-2 text-red-400 mt-2">
                <UIcon name="i-lucide-triangle-alert" class="w-3.5 h-3.5" />
                <span>{{ updateError }}</span>
              </div>

              <div v-if="updateDone && !updateError" class="flex items-center gap-2 text-emerald-400 mt-2">
                <UIcon name="i-lucide-check-circle" class="w-3.5 h-3.5" />
                <span>Done!</span>
              </div>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <div class="flex-1 flex flex-col bg-background overflow-hidden">
      <div class="h-14 border-b border-border/40 flex items-center px-4 gap-3 shrink-0">
        <button
          @click="isSidebarOpen = true"
          class="w-9 h-9 rounded-lg flex items-center justify-center text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-900 hover:text-slate-700 dark:hover:text-slate-100 transition-colors shrink-0"
        >
          <UIcon name="i-lucide-menu" class="w-5 h-5" />
        </button>
        <h2 class="text-sm font-heading font-bold text-slate-900 dark:text-slate-100 tracking-tight">Updates</h2>
        <div v-if="loading" class="flex items-center gap-2 text-xs text-slate-400">
          <UIcon name="i-lucide-refresh-cw" class="w-3.5 h-3.5 animate-spin" />
          Fetching...
        </div>
      </div>

      <div class="flex-1 p-6 sm:p-10 overflow-y-auto">
      <div class="max-w-4xl mx-auto space-y-6">
        <div class="border-b border-slate-200 dark:border-border/20 pb-4 space-y-4">
          <div class="flex bg-slate-100/50 dark:bg-slate-800/40 p-0.5 rounded-lg border border-slate-200 dark:border-border/10 shadow-sm w-fit">
            <button
              v-for="tab in tabs"
              :key="tab.id"
              @click="activeTab = tab.id"
              class="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-[13px] font-medium transition-all"
              :class="activeTab === tab.id
                ? 'bg-white dark:bg-slate-700/80 text-sky-500 dark:text-sky-400 shadow-sm'
                : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300'"
            >
              <UIcon :name="tab.icon" class="w-3.5 h-3.5" />
              {{ tab.label }}
            </button>
          </div>
        </div>

        <Transition name="fade-slide" mode="out-in">
          <div v-if="activeTab === 'clauductor'" class="space-y-8 py-10 text-center">
            <div class="flex flex-col items-center gap-4">
              <div class="w-16 h-16 rounded-2xl bg-slate-100 dark:bg-slate-900 flex items-center justify-center border border-slate-200 dark:border-border/20">
                <UIcon name="i-lucide-sparkles" class="w-8 h-8 text-slate-400 dark:text-slate-500" />
              </div>
              <div class="space-y-1">
                <h3 class="text-lg font-bold text-foreground">Clauductor Releases</h3>
                <p class="text-sm text-slate-500 max-w-xs mx-auto">
                  Changelog will appear here soon. Stay tuned for the first official release notes.
                </p>
              </div>
            </div>
          </div>

          <div v-else-if="activeTab === 'claude-code'" class="space-y-6">
            <Transition name="fade-slide">
              <div
                v-if="updateAvailable"
                class="p-4 bg-sky-500/10 border border-sky-500/20 rounded-2xl flex items-center justify-between gap-4 flex-wrap"
              >
                <div class="flex items-center gap-3">
                  <UIcon name="i-lucide-arrow-up-circle" class="w-5 h-5 text-sky-500 shrink-0" />
                  <div>
                    <p class="text-sm font-semibold text-sky-600 dark:text-sky-400">Update available: v{{ latestVersion }}</p>
                    <p class="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                      Installed: v{{ installedVersion }}
                    </p>
                  </div>
                </div>
                <button
                  @click="startUpdate"
                  class="flex items-center gap-2 px-4 py-2 bg-sky-500 hover:bg-sky-600 text-white text-sm font-semibold rounded-xl transition-colors shrink-0"
                >
                  <UIcon name="i-lucide-download" class="w-4 h-4" />
                  Update now
                </button>
              </div>
            </Transition>

            <div v-if="error" class="p-4 bg-red-500/10 border border-red-500/20 rounded-2xl flex items-center gap-3 text-red-500 text-sm">
              <UIcon name="i-lucide-triangle-alert" class="w-5 h-5" />
              {{ error }}
            </div>

            <div v-else-if="!loading && renderedClaudeCode" class="markdown-content p-6 sm:p-8 bg-slate-50 dark:bg-slate-900/50 rounded-2xl border border-slate-200 dark:border-border/20 shadow-sm overflow-x-auto">
              <div v-html="renderedClaudeCode"></div>
            </div>

            <div v-else-if="loading" class="space-y-4">
              <div v-for="i in 5" :key="i" class="h-20 bg-slate-100 dark:bg-slate-800 animate-pulse rounded-xl"></div>
            </div>
          </div>
        </Transition>
      </div>
    </div>
    </div>
  </div>
</template>

<style scoped>
.fade-slide-enter-active,
.fade-slide-leave-active {
  transition: all 0.25s ease-out;
}

.fade-slide-enter-from {
  opacity: 0;
  transform: translateY(12px);
}

.fade-slide-leave-to {
  opacity: 0;
  transform: translateY(-12px);
}

.modal-enter-active,
.modal-leave-active {
  transition: all 0.2s ease-out;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .relative,
.modal-leave-to .relative {
  transform: scale(0.96) translateY(8px);
}

.markdown-content :deep(h1) {
  @apply text-2xl font-bold mb-6 border-b border-slate-200 dark:border-border/20 pb-2 text-foreground;
}

.markdown-content :deep(h2) {
  @apply text-xl font-bold mt-8 mb-4 text-foreground;
}

.markdown-content :deep(h3) {
  @apply text-lg font-bold mt-6 mb-2 text-foreground;
}

.markdown-content :deep(p) {
  @apply mb-4 text-sm leading-relaxed text-slate-600 dark:text-slate-300;
}

.markdown-content :deep(ul) {
  @apply list-disc list-inside mb-4 ml-2 space-y-1 text-sm text-slate-600 dark:text-slate-300;
}

.markdown-content :deep(li) {
  @apply leading-relaxed;
}

.markdown-content :deep(code) {
  @apply px-1.5 py-0.5 bg-slate-200 dark:bg-slate-800 rounded-md font-mono text-xs text-slate-800 dark:text-slate-200;
}

.markdown-content :deep(a) {
  @apply text-sky-500 hover:underline;
}

.markdown-content :deep(hr) {
  @apply my-8 border-slate-200 dark:border-border/20;
}
</style>
