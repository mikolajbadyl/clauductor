<script setup lang="ts">
import type { NodeDetail } from '~/types'

const props = defineProps<{
  node: NodeDetail | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

function iconColor(color?: string) {
  switch (color) {
    case 'green': return 'bg-emerald-500/10'
    case 'indigo': return 'bg-indigo-500/10'
    case 'amber': return 'bg-amber-500/10'
    default: return 'bg-sky-500/10'
  }
}

function iconTextColor(color?: string) {
  switch (color) {
    case 'green': return 'text-emerald-400'
    case 'indigo': return 'text-indigo-400'
    case 'amber': return 'text-amber-400'
    default: return 'text-sky-400'
  }
}
</script>

<template>
  <Transition name="sheet">
    <div
      v-if="node"
      class="absolute bottom-0 left-0 right-0 z-50 flex flex-col max-h-[55vh] bg-background border-t border-border/40 rounded-t-2xl shadow-2xl"
    >
      <div class="flex justify-center pt-2 pb-1 shrink-0">
        <div class="w-10 h-1 rounded-full bg-slate-300 dark:bg-slate-600" />
      </div>

      <div class="px-4 pb-3 flex items-center justify-between shrink-0">
        <div class="flex items-center gap-2.5 min-w-0">
          <div
            class="w-7 h-7 rounded-lg flex items-center justify-center flex-shrink-0"
            :class="iconColor(node.color)"
          >
            <UIcon
              :name="toolIcons[node.type] || 'i-lucide-info'"
              class="w-3.5 h-3.5"
              :class="iconTextColor(node.color)"
            />
          </div>
          <div class="min-w-0">
            <div class="font-heading font-semibold text-sm text-foreground truncate">{{ node.label || 'Details' }}</div>
            <div class="text-[10px] text-slate-500 font-mono">{{ node.type }}</div>
          </div>
        </div>
        <button
          @click="emit('close')"
          class="w-7 h-7 rounded-lg flex items-center justify-center text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
        >
          <UIcon name="i-lucide-x" class="w-4 h-4" />
        </button>
      </div>

      <div class="flex-1 overflow-y-auto px-4 pb-4 custom-scrollbar">
        <NodeDetailContent :node="node" />
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.sheet-enter-active,
.sheet-leave-active {
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}
.sheet-enter-from,
.sheet-leave-to {
  transform: translateY(100%);
}
</style>
