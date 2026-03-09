<script setup lang="ts">
import type { SessionSummary } from '~/types'

const emit = defineEmits<{
  (e: 'select', sessionId: string, projectPath: string, encodedDir: string): void
  (e: 'close'): void
}>()

const { projects, sessions, selectedProject, loading, fetchProjects, selectProject, clearProject } = useProjects()

onMounted(fetchProjects)

function formatDate(ts: number): string {
  if (!ts) return ''
  return new Date(ts).toLocaleDateString('en-GB', { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit' })
}

function truncate(text: string, max = 80): string {
  if (!text) return 'No prompt'
  return text.length > max ? text.slice(0, max) + '...' : text
}

function handleSessionClick(session: SessionSummary) {
  if (!selectedProject.value) return
  emit('select', session.id, selectedProject.value.path, selectedProject.value.encodedDir)
}
</script>

<template>
  <div class="flex flex-col h-full bg-background">
    <div class="h-12 border-b border-border/30 flex items-center justify-between px-4 shrink-0">
      <div class="flex items-center gap-2">
        <button v-if="selectedProject" @click="clearProject" class="text-slate-400 hover:text-slate-200 transition-colors">
          <UIcon name="i-lucide-arrow-left" class="w-4 h-4" />
        </button>
        <UIcon name="i-lucide-folder" class="w-4 h-4 text-sky-400" />
        <span class="text-sm font-heading font-semibold text-foreground">
          {{ selectedProject ? selectedProject.name : 'Projects' }}
        </span>
      </div>
      <UButton color="gray" variant="ghost" icon="i-lucide-x" size="sm" class="rounded-lg" @click="$emit('close')" />
    </div>

    <div v-if="loading" class="flex-1 flex items-center justify-center">
      <UIcon name="i-lucide-refresh-cw" class="w-6 h-6 text-slate-500 animate-spin" />
    </div>

    <div v-else-if="!selectedProject" class="flex-1 overflow-y-auto p-3 space-y-1">
      <button
        v-for="project in projects"
        :key="project.encodedDir"
        @click="selectProject(project)"
        class="w-full flex items-center gap-3 px-3 py-3 rounded-xl text-left transition-all hover:bg-slate-900 border border-transparent hover:border-border/30 group"
      >
        <div class="w-8 h-8 rounded-lg bg-slate-800 border border-border/30 flex items-center justify-center shrink-0">
          <UIcon name="i-lucide-folder" class="w-4 h-4 text-sky-400" />
        </div>
        <div class="min-w-0 flex-1">
          <div class="text-sm font-medium text-slate-200 group-hover:text-white truncate">{{ project.name }}</div>
          <div class="text-[11px] text-slate-500 font-mono truncate">{{ project.path }}</div>
        </div>
        <UIcon name="i-lucide-chevron-right" class="w-4 h-4 text-slate-600 group-hover:text-slate-400 shrink-0" />
      </button>

      <div v-if="projects.length === 0" class="flex flex-col items-center justify-center py-12 text-slate-500 gap-3">
        <UIcon name="i-lucide-folder-open" class="w-10 h-10 text-slate-600" />
        <p class="text-sm">No projects found</p>
      </div>
    </div>

    <div v-else class="flex-1 overflow-y-auto p-3 space-y-1">
      <button
        v-for="session in sessions"
        :key="session.id"
        @click="handleSessionClick(session)"
        class="w-full flex items-start gap-3 px-3 py-3 rounded-xl text-left transition-all hover:bg-slate-900 border border-transparent hover:border-border/30 group"
      >
        <div class="w-8 h-8 rounded-lg bg-slate-800 border border-border/30 flex items-center justify-center shrink-0 mt-0.5">
          <UIcon name="i-lucide-messages-square" class="w-4 h-4 text-violet-400" />
        </div>
        <div class="min-w-0 flex-1">
          <div class="text-sm text-slate-300 group-hover:text-white leading-snug">{{ truncate(session.display) }}</div>
          <div class="text-[11px] text-slate-600 font-mono mt-1 flex items-center gap-2">
            <span>{{ formatDate(session.timestamp) }}</span>
            <span class="text-slate-700">{{ session.id.slice(0, 8) }}</span>
          </div>
        </div>
      </button>

      <div v-if="sessions.length === 0" class="flex flex-col items-center justify-center py-12 text-slate-500 gap-3">
        <UIcon name="i-lucide-messages-square" class="w-10 h-10 text-slate-600" />
        <p class="text-sm">No sessions found</p>
      </div>
    </div>
  </div>
</template>
