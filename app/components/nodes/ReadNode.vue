<script setup lang="ts">
import { ref, watch, computed } from 'vue'

const props = defineProps<{
  id: string
  data: {
    label: string
    icon: string
    type: string
    status: 'idle' | 'running' | 'success' | 'error'
    args?: Record<string, any>
  }
}>()

const { highlight } = useSyntaxHighlighter()
const { nodeDetails } = useWorkGraph()
const highlightedHtml = ref('')

const isHovering = ref(false)
const hoverProgress = ref(0)
const isRevealed = ref(false)
let intervalTimer: ReturnType<typeof setInterval> | null = null

function onMouseEnter() {
  isHovering.value = true
  hoverProgress.value = 0

  if (!isRevealed.value) {
    const duration = 1000
    const interval = 50
    const steps = duration / interval
    let step = 0

    intervalTimer = setInterval(() => {
      step++
      hoverProgress.value = (step / steps) * 100
      if (step >= steps) {
        if (intervalTimer) clearInterval(intervalTimer)
        isRevealed.value = true
      }
    }, interval)
  }
}

function onMouseLeave() {
  isHovering.value = false
  if (intervalTimer) clearInterval(intervalTimer)
  hoverProgress.value = 0
  isRevealed.value = false
  highlightedHtml.value = ''
}

const resultText = computed(() => {
  return nodeDetails.value[props.id]?.result || ''
})

const codeSnippet = computed(() => {
  const text = resultText.value
  if (!text) return ''
  const lines = text.split('\n')
  if (lines.length > 8) {
    return lines.slice(0, 8).join('\n') + '\n...'
  }
  return text
})

watch([codeSnippet, isRevealed], async ([snippet, revealed]) => {
  if (snippet && revealed) {
    highlightedHtml.value = await highlight(snippet, props.data.args?.file_path || '')
  }
})
</script>

<template>
  <WorkNode
    :id="props.id"
    :data="{
      ...props.data,
      color: 'indigo',
      customClass: 'min-w-52 max-w-[800px] w-auto'
    }"
    @mouseenter="onMouseEnter"
    @mouseleave="onMouseLeave"
  >
    <div v-if="props.data.args?.file_path" class="text-xs text-indigo-500/80 dark:text-indigo-400/80 font-mono mb-1 truncate px-1">
      {{ props.data.args.file_path.split('/').pop() }}
    </div>

    <!-- Reveal Progress Bar -->
    <div
      v-if="isHovering && !isRevealed"
      class="absolute bottom-0 left-0 h-1 bg-indigo-500/50 transition-all duration-75 ease-linear"
      :style="{ width: `${hoverProgress}%` }"
    />

    <!-- Shiki Code Preview -->
    <Transition name="expand">
      <div
        v-if="isRevealed && highlightedHtml"
        class="text-[10px] sm:text-[11px] overflow-hidden rounded-md bg-[#1a1a2e] dark:bg-[#121212] border border-slate-200 dark:border-border/20 p-2 shine-effect shiki-container mt-1"
        v-html="highlightedHtml"
      />
    </Transition>

    <!-- Fallback if no result yet -->
    <div v-if="isRevealed && !resultText" class="text-[10px] text-slate-500 italic px-1 mt-1">
      Reading file...
    </div>
  </WorkNode>
</template>

<style>
.expand-enter-active,
.expand-leave-active {
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  overflow: hidden;
}

.expand-enter-from,
.expand-leave-to {
  max-height: 0;
  opacity: 0;
  padding-top: 0;
  padding-bottom: 0;
  margin-top: 0;
  border-width: 0;
}

.expand-enter-to,
.expand-leave-from {
  max-height: 300px;
  opacity: 1;
}
.shiki-container pre {
  margin: 0 !important;
  background-color: transparent !important;
}
.shiki-container code {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  line-height: 1.4;
}
</style>
