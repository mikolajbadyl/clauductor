<script setup lang="ts">
import type { NodeDetail } from '~/types'

const props = defineProps<{
  node: NodeDetail | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const isOpen = ref(false)

watch(() => props.node, (n) => {
  isOpen.value = !!n
})

watch(isOpen, (val) => {
  if (!val) emit('close')
})

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
  <USlideover v-model="isOpen" :ui="{ width: 'w-full sm:w-[500px] xl:w-[600px]', wrapper: 'z-[100]', overlay: { background: 'bg-background/50 backdrop-blur-sm' } }">
    <div class="flex flex-col h-full bg-background text-foreground text-sm">

      <div class="px-5 py-4 border-b border-border/30 flex items-center justify-between shrink-0">
        <div class="flex items-center gap-3 min-w-0">
          <div
            class="w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0"
            :class="iconColor(node?.color)"
          >
            <UIcon
              :name="toolIcons[node?.type || ''] || 'i-lucide-info'"
              class="w-4 h-4"
              :class="iconTextColor(node?.color)"
            />
          </div>
          <div class="min-w-0">
            <div class="font-heading font-semibold text-foreground truncate">{{ node?.label || 'Details' }}</div>
            <div class="text-xs text-slate-500 font-mono">{{ node?.type }}</div>
          </div>
        </div>
        <UButton color="gray" variant="ghost" icon="i-lucide-x" size="sm" class="rounded-lg" @click="isOpen = false" />
      </div>

      <div class="flex-1 overflow-y-auto p-5 custom-scrollbar">
        <NodeDetailContent v-if="node" :node="node" />
        <div v-else class="flex flex-col items-center justify-center text-slate-600 gap-3 pt-16">
          <UIcon name="i-lucide-inbox" class="w-8 h-8" />
          <p class="text-sm">No details yet</p>
        </div>
      </div>

    </div>
  </USlideover>
</template>
