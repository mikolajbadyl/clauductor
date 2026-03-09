<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { parseSearchResult, type SearchLink } from '~/utils/tools'

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

const { nodeDetails } = useWorkGraph()

const isHovering = ref(false)
const hoverProgress = ref(0)
const isRevealed = ref(false)
let intervalTimer: ReturnType<typeof setInterval> | null = null

const parsedLinks = ref<SearchLink[]>([])

watch(
  () => nodeDetails.value[props.id]?.result,
  (result) => {
    if (!result) { parsedLinks.value = []; return }
    const parsed = parseSearchResult(result)
    parsedLinks.value = parsed.links
  },
  { immediate: true }
)

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
}

const previewLinks = computed(() => parsedLinks.value.slice(0, 5))

const hasResult = computed(() => !!nodeDetails.value[props.id]?.result)

function hostname(url: string): string {
  try { return new URL(url).hostname.replace('www.', '') } catch { return url }
}
</script>

<template>
  <WorkNode
    :id="props.id"
    :data="{
      ...props.data,
      color: 'amber',
      customClass: 'min-w-56 max-w-[500px] w-auto'
    }"
    @mouseenter="onMouseEnter"
    @mouseleave="onMouseLeave"
  >
    <div v-if="props.data.args?.query" class="text-xs text-amber-500/80 dark:text-amber-400/80 font-mono mb-1 truncate px-1">
      {{ props.data.args.query }}
    </div>

    <div
      v-if="isHovering && !isRevealed"
      class="absolute bottom-0 left-0 h-1 bg-amber-500/50 transition-all duration-75 ease-linear"
      :style="{ width: `${hoverProgress}%` }"
    />

    <Transition name="expand">
      <div
        v-if="isRevealed && previewLinks.length > 0"
        class="text-[10px] sm:text-[11px] overflow-hidden rounded-md bg-[#1a1a2e] dark:bg-[#121212] border border-slate-200 dark:border-border/20 p-2 mt-1 space-y-1"
      >
        <div
          v-for="(link, i) in previewLinks"
          :key="i"
          class="flex items-start gap-1.5 text-slate-300"
        >
          <UIcon name="i-lucide-link" class="w-3 h-3 text-amber-400/60 mt-0.5 shrink-0" />
          <div class="min-w-0">
            <div class="truncate text-slate-200 leading-tight">{{ link.title }}</div>
            <div class="truncate text-amber-500/60 text-[9px]">{{ hostname(link.url) }}</div>
          </div>
        </div>
        <div v-if="parsedLinks.length > 5" class="text-[9px] text-slate-500 pl-4">
          +{{ parsedLinks.length - 5 }} more
        </div>
      </div>
    </Transition>

    <div v-if="isRevealed && !hasResult" class="text-[10px] text-slate-500 italic px-1 mt-1">
      Searching...
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
</style>
