<script setup lang="ts">
const { layoutConfig } = useWorkGraph()

const options = [
  { value: 'snake', label: 'Snake', icon: 'i-lucide-corner-down-right' },
  { value: 'radial', label: 'Radial', icon: 'i-lucide-globe' },
  { value: 'linear', label: 'Linear', icon: 'i-lucide-arrow-left-right' },
] as const

const localConfig = ref({ ...layoutConfig.value })

const limitOptions = [
  { value: 50, label: 'Last 50' },
  { value: 100, label: 'Last 100' },
  { value: 500, label: 'Last 500' },
  { value: 'all', label: 'All Nodes' }
]

watch(layoutConfig, (newVal) => {
  localConfig.value = { ...newVal }
}, { deep: true })

const hasChanges = computed(() => {
  return localConfig.value.mode !== layoutConfig.value.mode ||
         localConfig.value.snakeColumns !== layoutConfig.value.snakeColumns ||
         localConfig.value.radialSpacing !== layoutConfig.value.radialSpacing ||
         localConfig.value.nodeLimit !== layoutConfig.value.nodeLimit
})

function applyChanges() {
  layoutConfig.value.mode = localConfig.value.mode
  layoutConfig.value.snakeColumns = localConfig.value.snakeColumns
  layoutConfig.value.radialSpacing = localConfig.value.radialSpacing
  layoutConfig.value.nodeLimit = localConfig.value.nodeLimit
}
</script>

<template>
  <div class="flex flex-wrap items-center justify-center gap-x-6 gap-y-3 bg-card/90 backdrop-blur-md px-4 py-3 sm:py-2 rounded-xl border border-border/40 shadow-sm pointer-events-auto h-auto min-h-[46px] w-[90vw] sm:w-auto transition-all">
    <!-- Layout Selector -->
    <div class="flex items-center gap-3">
      <span class="hidden sm:inline text-[10px] font-semibold text-slate-400 uppercase tracking-wider">Layout</span>
      <div class="flex bg-slate-100 dark:bg-slate-900/50 p-0.5 rounded-lg border border-slate-200 dark:border-border/20">
        <button
          v-for="opt in options"
          :key="opt.value"
          @click="localConfig.mode = opt.value"
          class="flex items-center gap-1.5 px-3 py-1.5 sm:py-1 rounded-md text-xs font-medium transition-colors"
          :class="localConfig.mode === opt.value ? 'bg-sky-500/10 text-sky-500 dark:text-sky-400' : 'text-slate-500 hover:text-slate-700 dark:hover:text-slate-300 hover:bg-slate-200/50 dark:hover:bg-slate-800/50'"
        >
          <UIcon :name="opt.icon" class="w-3.5 h-3.5" />
          <span class="hidden sm:inline">{{ opt.label }}</span>
        </button>
      </div>
    </div>

    <!-- Snake Columns Setting -->
    <div v-if="localConfig.mode === 'snake'" class="flex items-center gap-3">
      <span class="text-[10px] font-semibold text-slate-400 uppercase tracking-wider">Cols</span>
      <div class="w-20 sm:w-24 flex items-center">
        <!-- Prevent URange from immediately updating parent by using v-model locally -->
        <URange v-model="localConfig.snakeColumns" :min="2" :max="30" :step="1" color="sky" size="sm" />
      </div>
      <span class="text-xs font-mono text-sky-400 w-4 text-left sm:text-right">{{ localConfig.snakeColumns }}</span>
    </div>

    <!-- Radial Spacing Setting -->
    <div v-if="localConfig.mode === 'radial'" class="flex items-center gap-3">
      <span class="text-[10px] font-semibold text-slate-400 uppercase tracking-wider">Spacing</span>
      <div class="w-20 sm:w-24 flex items-center">
        <URange v-model="localConfig.radialSpacing" :min="50" :max="1000" :step="10" color="sky" size="sm" />
      </div>
      <span class="text-xs font-mono text-sky-400 w-6 text-left sm:text-right">{{ localConfig.radialSpacing }}</span>
    </div>

    <!-- Node Limit Selector -->
    <div class="flex items-center gap-3">
      <span class="text-[10px] font-semibold text-slate-400 uppercase tracking-wider">Show</span>
      <USelectMenu 
        v-model="localConfig.nodeLimit" 
        :options="limitOptions" 
        value-attribute="value"
        option-attribute="label"
        class="w-24 sm:w-28"
        size="xs"
      />
    </div>

    <!-- Apply Button -->
    <div class="flex items-center transition-opacity" :class="hasChanges ? 'opacity-100' : 'opacity-40'">
      <UButton 
        @click="applyChanges"
        :disabled="!hasChanges"
        color="sky" 
        variant="soft" 
        size="xs" 
        icon="i-lucide-check"
        label="Apply"
        class="rounded-lg shadow-sm font-semibold w-full sm:w-auto justify-center"
      />
    </div>
  </div>
</template>
