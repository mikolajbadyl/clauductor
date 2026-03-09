<script setup lang="ts">
import { backendUrl } from '~/utils/api'

const route = useRoute()

const emit = defineEmits<{
  (e: 'navigate'): void
}>()

const navItems = [
  { path: '/', label: 'Chat', icon: 'i-lucide-messages-square' },
  { path: '/settings', label: 'Settings', icon: 'i-lucide-settings' },
  { path: '/updates', label: 'Updates', icon: 'i-lucide-megaphone' },
]

const version = ref('')

async function fetchVersion() {
  try {
    const res = await fetch(backendUrl('/api/version'))
    if (res.ok) {
      const data = await res.json()
      version.value = data.version || 'unknown'
    }
  } catch (e) {
    version.value = '?'
  }
}

onMounted(() => {
  fetchVersion()
})

function navigate(path: string) {
  navigateTo(path)
  emit('navigate')
}
</script>

<template>
  <div class="w-64 h-full bg-white dark:bg-slate-950 border-r border-slate-200 dark:border-border/30 flex flex-col shrink-0">
    <!-- Header -->
    <div class="h-14 border-b border-slate-200 dark:border-border/20 flex items-center px-5">
      <h1 class="font-heading font-bold text-slate-900 dark:text-slate-100 text-lg tracking-tight">Clauductor</h1>
    </div>

    <div class="flex-1 overflow-y-auto p-3 space-y-1">
      <button
        v-for="item in navItems"
        :key="item.path"
        @click="navigate(item.path)"
        class="w-full flex items-center gap-3 px-3 py-2.5 rounded-xl transition-all group"
        :class="route.path === item.path
          ? 'bg-sky-500/10 text-sky-500 dark:text-sky-400 border border-sky-500/20'
          : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-900 border border-transparent'"
      >
        <UIcon :name="item.icon" class="w-5 h-5" :class="route.path === item.path ? 'text-sky-500 dark:text-sky-400' : 'text-slate-400 dark:text-slate-500 group-hover:text-slate-600 dark:group-hover:text-slate-300'" />
        <span class="text-sm font-medium">{{ item.label }}</span>
      </button>
    </div>

    <!-- Footer / Version -->
    <div class="p-4 border-t border-slate-200 dark:border-border/20">
      <div class="flex items-center gap-2 px-2 py-1">
        <UIcon name="i-lucide-info" class="w-3.5 h-3.5 text-slate-400 dark:text-slate-500 shrink-0" />
        <span class="text-[10px] font-mono text-slate-400 dark:text-slate-500 truncate">{{ version || '...' }}</span>
      </div>
    </div>
  </div>
</template>
