<script setup lang="ts">
import { VueFlow, useVueFlow } from '@vue-flow/core'
import type { Node, Edge } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import WorkNode from './WorkNode.vue'
import EditNode from './nodes/EditNode.vue'
import ReadNode from './nodes/ReadNode.vue'
import WriteNode from './nodes/WriteNode.vue'
import WebSearchNode from './nodes/WebSearchNode.vue'
import LayoutSettings from './LayoutSettings.vue'

const props = defineProps<{
  nodes: Node[]
  edges: Edge[]
}>()

const emit = defineEmits<{
  (e: 'nodeClick', nodeId: string): void
}>()

const { onNodeClick, fitView, setCenter, onPaneReady, onMove, getNodes, getIntersectingNodes, project, dimensions } = useVueFlow()
const { layoutConfig, isLayouting } = useWorkGraph()

onNodeClick((evt) => {
  emit('nodeClick', evt.node.id)
})

let fitViewTimer: ReturnType<typeof setTimeout> | null = null

function centerOnLastNode() {
  if (fitViewTimer) clearTimeout(fitViewTimer)
  fitViewTimer = setTimeout(() => {
    const lastNode = props.nodes[props.nodes.length - 1]
    if (lastNode) {
      setCenter(lastNode.position.x + 104, lastNode.position.y, { zoom: 1, duration: 600 })
    }
  }, 50)
}

function fitAfterLayoutChange() {
  if (fitViewTimer) clearTimeout(fitViewTimer)
  fitViewTimer = setTimeout(() => {
    if (props.nodes.length > 0) {
      const lastNode = props.nodes[props.nodes.length - 1]
      if (lastNode) {
        setCenter(lastNode.position.x + 104, lastNode.position.y, { zoom: 0.8, duration: 800 })
      }
    } else {
      fitView({ padding: 0.2, duration: 800 })
    }
  }, 50)
}

watch(() => props.nodes.length, centerOnLastNode)
watch(() => layoutConfig.value, fitAfterLayoutChange, { deep: true })

onUnmounted(() => {
  if (fitViewTimer) clearTimeout(fitViewTimer)
})
</script>

<template>
  <div class="h-full w-full bg-background absolute inset-0 pb-20 overflow-hidden">
    <div class="absolute inset-0 bg-[radial-gradient(ellipse_at_top_right,_var(--tw-gradient-stops))] from-sky-900/20 via-background to-background z-0 pointer-events-none"></div>

    <div v-if="props.nodes.length === 0" class="absolute inset-0 z-10 flex flex-col items-center justify-center gap-4 text-slate-400 dark:text-slate-500">
      <UIcon name="i-lucide-map" class="w-12 h-12 text-slate-300 dark:text-slate-600" />
      <p class="text-sm font-medium">No activity yet</p>
      <p class="text-xs text-slate-400 dark:text-slate-600">Nodes will appear here as Claude works</p>
    </div>

    <VueFlow v-else :nodes="props.nodes" :edges="props.edges" :min-zoom="0.1" :max-zoom="1.5" fit-view-on-init class="z-10 relative">
      <Background pattern-color="rgb(var(--color-border) / 0.5)" :gap="24" :size="1" />
      <template #node-work="nodeProps">
        <WorkNode :id="nodeProps.id" :data="nodeProps.data" />
      </template>
      <template #node-edit="nodeProps">
        <EditNode :id="nodeProps.id" :data="nodeProps.data" />
      </template>
      <template #node-read="nodeProps">
        <ReadNode :id="nodeProps.id" :data="nodeProps.data" />
      </template>
      <template #node-write="nodeProps">
        <WriteNode :id="nodeProps.id" :data="nodeProps.data" />
      </template>
      <template #node-websearch="nodeProps">
        <WebSearchNode :id="nodeProps.id" :data="nodeProps.data" />
      </template>
    </VueFlow>

    <!-- Full Graph Loading Overlay -->
    <Transition name="fade">
      <div v-if="isLayouting" class="absolute inset-0 z-[60] bg-background/50 backdrop-blur-[2px] flex flex-col items-center justify-center pointer-events-auto">
        <div class="bg-card px-6 py-4 rounded-2xl border border-border/40 shadow-xl flex items-center gap-4">
          <UIcon name="i-lucide-refresh-cw" class="w-6 h-6 text-sky-400 animate-spin" />
          <div class="flex flex-col">
            <span class="text-sm font-semibold text-slate-700 dark:text-slate-200">Rebuilding Layout</span>
            <span class="text-xs text-slate-500 dark:text-slate-400">Positioning nodes...</span>
          </div>
        </div>
      </div>
    </Transition>

    <!-- UI Controls Overlay -->
    <div class="absolute top-4 left-1/2 -translate-x-1/2 z-50 flex flex-col items-center pointer-events-none">
      <div class="flex items-center gap-2 pointer-events-auto">
        <LayoutSettings />
      </div>
    </div>
  </div>
</template>

<style>
.vue-flow__edge-path {
  transition: stroke 0.3s ease;
}
.fade-enter-active, .fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from, .fade-leave-to {
  opacity: 0;
}
</style>
