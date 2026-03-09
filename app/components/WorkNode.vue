<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'
import { computed } from 'vue'

const props = defineProps<{
  id: string
  data: {
    label: string
    icon: string
    type: string
    status: 'idle' | 'running' | 'success' | 'error'
    customClass?: string
    color?: 'blue' | 'green' | 'amber' | 'rose' | 'indigo' | 'slate'
  }
}>()

const themeClasses = computed(() => {
  const color = props.data.color || 'blue'
  switch (color) {
    case 'green':
      return {
        border: 'border-emerald-500/30 dark:border-emerald-400/20 hover:border-emerald-400 dark:hover:border-emerald-500',
        borderActive: 'border-emerald-500 bg-emerald-50/50 dark:bg-emerald-950/20',
        borderSuccess: 'border-emerald-500/40 bg-emerald-50/20 dark:bg-emerald-950/10',
        iconBg: 'bg-emerald-500/10 dark:bg-emerald-400/5',
        iconText: 'text-emerald-500 dark:text-emerald-400'
      }
    case 'amber':
      return {
        border: 'border-amber-500/30 dark:border-amber-400/20 hover:border-amber-400 dark:hover:border-amber-500',
        borderActive: 'border-amber-500 bg-amber-50/50 dark:bg-amber-950/20',
        borderSuccess: 'border-amber-500/40 bg-amber-50/20 dark:bg-amber-950/10',
        iconBg: 'bg-amber-500/10 dark:bg-amber-400/5',
        iconText: 'text-amber-500 dark:text-amber-400'
      }
    case 'rose':
      return {
        border: 'border-rose-500/30 dark:border-rose-400/20 hover:border-rose-400 dark:hover:border-rose-500',
        borderActive: 'border-rose-500 bg-rose-50/50 dark:bg-rose-950/20',
        borderSuccess: 'border-rose-500/40 bg-rose-50/20 dark:bg-rose-950/10',
        iconBg: 'bg-rose-500/10 dark:bg-rose-400/5',
        iconText: 'text-rose-500 dark:text-rose-400'
      }
    case 'indigo':
      return {
        border: 'border-indigo-500/30 dark:border-indigo-400/20 hover:border-indigo-400 dark:hover:border-indigo-500',
        borderActive: 'border-indigo-500 bg-indigo-50/50 dark:bg-indigo-950/20',
        borderSuccess: 'border-indigo-500/40 bg-indigo-50/20 dark:bg-indigo-950/10',
        iconBg: 'bg-indigo-500/10 dark:bg-indigo-400/5',
        iconText: 'text-indigo-500 dark:text-indigo-400'
      }
    case 'slate':
      return {
        border: 'border-slate-200 dark:border-border/40 hover:border-slate-400 dark:hover:border-slate-500',
        borderActive: 'border-slate-400 bg-slate-50 dark:bg-slate-900/50',
        borderSuccess: 'border-slate-300 dark:border-slate-700 bg-slate-50/50 dark:bg-slate-900/20',
        iconBg: 'bg-slate-100 dark:bg-slate-800/50',
        iconText: 'text-slate-500 dark:text-slate-400'
      }
    case 'blue':
    default:
      return {
        border: 'border-blue-500/30 dark:border-blue-400/20 hover:border-blue-400 dark:hover:border-blue-500',
        borderActive: 'border-blue-500 bg-blue-50/50 dark:bg-blue-950/20',
        borderSuccess: 'border-blue-500/40 bg-blue-50/20 dark:bg-blue-950/10',
        iconBg: 'bg-blue-500/10 dark:bg-blue-400/5',
        iconText: 'text-blue-500 dark:text-blue-400'
      }
  }
})
</script>

<template>
  <div
    class="group relative rounded-xl border p-2.5 flex flex-col gap-2 w-auto min-w-52 max-w-[300px] sm:max-w-[400px] overflow-hidden cursor-pointer bg-card transition-all duration-300"
    :class="[
      {
        [themeClasses.border]: props.data.status === 'idle',
        [themeClasses.borderActive]: props.data.status === 'running',
        [themeClasses.borderSuccess]: props.data.status === 'success',
        'border-rose-500/40 bg-rose-50/10 dark:bg-rose-950/5': props.data.status === 'error',
      },
      props.data.customClass
    ]"
  >
    <Handle type="target" id="top" :position="Position.Top" class="!bg-slate-600 !w-2 !h-2 !border-0 !-top-1" />
    <Handle type="target" id="left" :position="Position.Left" class="!bg-slate-600 !w-1.5 !h-1.5 !border-0 !-left-1" />
    <Handle type="target" id="right" :position="Position.Right" class="!bg-slate-600 !w-1.5 !h-1.5 !border-0 !-right-1" />

    <!-- Header Row -->
    <div class="flex items-center gap-2.5">
      <div
        class="flex-shrink-0 w-8 h-8 rounded-lg flex items-center justify-center transition-colors duration-300"
        :class="[
          props.data.status === 'error' ? 'bg-rose-500/10' : themeClasses.iconBg
        ]"
      >
        <UIcon :name="props.data.icon" class="w-4 h-4" :class="[
          props.data.status === 'error' ? 'text-rose-500' : themeClasses.iconText
        ]" />
      </div>

      <div class="flex-grow min-w-0 pr-4">
        <div class="font-mono text-xs truncate" :class="{
          'text-slate-400 dark:text-slate-500': props.data.status === 'idle',
          'text-slate-700 dark:text-slate-200': props.data.status !== 'idle',
        }">
          {{ props.data.label }}
        </div>
      </div>

      <div class="flex-shrink-0 absolute right-3.5 top-4">
        <div v-if="props.data.status === 'running'" 
          class="w-2 h-2 rounded-full animate-pulse"
          :class="props.data.color === 'green' ? 'bg-emerald-400' : 'bg-blue-400'"
        />
        <UIcon v-else-if="props.data.status === 'success'" name="i-lucide-check-circle-2" class="w-4 h-4 opacity-80" :class="themeClasses.iconText" />
        <UIcon v-else-if="props.data.status === 'error'" name="i-lucide-x-circle" class="w-4 h-4 text-rose-500/60" />
        <div v-else class="w-2 h-2 rounded-full bg-slate-700 dark:bg-slate-600" />
      </div>
    </div>

    <!-- Payload Slot -->
    <slot></slot>

    <Handle type="source" id="bottom" :position="Position.Bottom" class="!bg-slate-600 !w-2 !h-2 !border-0 !-bottom-1" />
    <Handle type="source" id="left-src" :position="Position.Left" class="!bg-slate-600 !w-1.5 !h-1.5 !border-0 !-left-1" />
    <Handle type="source" id="right-src" :position="Position.Right" class="!bg-slate-600 !w-1.5 !h-1.5 !border-0 !-right-1" />
  </div>
</template>
