<script setup lang="ts">
import type { NodeDetail } from '~/types'
import { parseSearchResult } from '~/utils/tools'

const props = defineProps<{
  node: NodeDetail
}>()

const { highlight } = useSyntaxHighlighter()

const highlightedInputs = ref<Record<string, string>>({})
const highlightedOutput = ref('')
const copiedKey = ref<string | null>(null)
const contentViewMode = ref<'raw' | 'rendered'>('raw')

const isWriteNode = computed(() => props.node.type === 'Write')
const isMarkdownFile = computed(() => /\.(md|mdx|markdown)$/i.test(props.node.input?.file_path || ''))
const canRenderContent = computed(() => isWriteNode.value && isMarkdownFile.value && props.node.input?.content)

watch(() => props.node, async (n) => {
  highlightedInputs.value = {}
  highlightedOutput.value = ''

  const fp = n.input?.file_path || ''
  contentViewMode.value = (n.type === 'Write' && /\.(md|mdx|markdown)$/i.test(fp)) ? 'rendered' : 'raw'

  if (n.input) {
    const filename = n.input.file_path || n.input.filepath || ''
    for (const [key, val] of Object.entries(n.input)) {
      if (typeof val === 'string' && (val.includes('\n') || key.includes('code') || key.includes('string'))) {
        highlightedInputs.value[key] = await highlight(val, filename)
      }
    }
  }

  if (n.result) {
    const text = n.result.trim()
    if (text.includes('\n') || text.length > 150) {
      const filename = n.input?.file_path || n.input?.filepath || ''
      const isJson = text.startsWith('{') || text.startsWith('[')
      const lang = filename || (isJson ? 'test.json' : 'test.sh')
      highlightedOutput.value = await highlight(text, lang)
    }
  }
}, { immediate: true })

const isEditNode = computed(() => {
  const t = props.node.type?.toLowerCase() || ''
  return t.includes('edit') || t.includes('replace')
})

const hasDiff = computed(() => {
  return isEditNode.value && props.node.input?.old_string && props.node.input?.new_string
})

const diffLines = computed(() => {
  if (!hasDiff.value) return []
  const oldLines = (props.node.input!.old_string as string).split('\n')
  const newLines = (props.node.input!.new_string as string).split('\n')
  const lines: { type: 'remove' | 'add'; text: string }[] = []
  for (const l of oldLines) lines.push({ type: 'remove', text: l })
  for (const l of newLines) lines.push({ type: 'add', text: l })
  return lines
})

const inputEntries = computed(() => {
  if (!props.node.input) return []
  if (hasDiff.value) {
    return Object.entries(props.node.input).filter(([key]) => key !== 'old_string' && key !== 'new_string')
  }
  return Object.entries(props.node.input)
})

const isWebSearch = computed(() => props.node.type === 'WebSearch')

const searchResult = computed(() => {
  if (!isWebSearch.value || !props.node.result) return null
  return parseSearchResult(props.node.result)
})

function copyText(text: string, key: string) {
  navigator.clipboard.writeText(text)
  copiedKey.value = key
  setTimeout(() => { copiedKey.value = null }, 2000)
}

function getRawText(val: any): string {
  return typeof val === 'string' ? val : JSON.stringify(val, null, 2)
}

function hostname(url: string): string {
  try { return new URL(url).hostname.replace('www.', '') } catch { return url }
}

function paramColor(color?: string) {
  switch (color) {
    case 'green': return 'text-emerald-400/80'
    case 'indigo': return 'text-indigo-400/80'
    case 'amber': return 'text-amber-400/80'
    default: return 'text-sky-400/80'
  }
}
</script>

<template>
  <div class="space-y-4">

    <!-- Diff section for Edit nodes -->
    <div v-if="hasDiff">
      <h3 class="text-[11px] uppercase tracking-wider text-slate-500 mb-2 font-semibold">Changes</h3>
      <div class="text-[11px] sm:text-[12px] bg-[#1a1a2e] dark:bg-[#121212] rounded-xl p-2.5 sm:p-3 overflow-x-auto border border-slate-200 dark:border-border/20 font-mono leading-relaxed custom-scrollbar max-h-[40vh] relative group/diff">
        <button
          @click="copyText(diffLines.map(l => (l.type === 'remove' ? '- ' : '+ ') + l.text).join('\n'), 'diff')"
          class="absolute top-2 right-2 w-6 h-6 sm:w-7 sm:h-7 rounded-md flex items-center justify-center bg-slate-700/60 hover:bg-slate-600/80 text-slate-300 transition-all opacity-0 group-hover/diff:opacity-100"
        >
          <UIcon :name="copiedKey === 'diff' ? 'i-lucide-check' : 'i-lucide-clipboard-copy'" class="w-3 h-3 sm:w-3.5 sm:h-3.5" />
        </button>
        <div
          v-for="(line, i) in diffLines"
          :key="i"
          class="whitespace-pre-wrap break-all"
          :class="line.type === 'remove' ? 'text-rose-400/90 bg-rose-500/10' : 'text-emerald-400/90 bg-emerald-500/10'"
        >{{ line.type === 'remove' ? '- ' : '+ ' }}{{ line.text }}</div>
      </div>
    </div>

    <!-- Parameters -->
    <div v-if="inputEntries.length > 0">
      <h3 class="text-[11px] uppercase tracking-wider text-slate-500 mb-2 font-semibold">Parameters</h3>
      <div class="space-y-3">
        <div v-for="[key, val] in inputEntries" :key="key">
          <div class="flex items-center justify-between mb-1">
            <div class="text-[11px] sm:text-xs font-mono" :class="paramColor(node.color)">{{ key }}</div>
            <button
              v-if="canRenderContent && key === 'content'"
              @click="contentViewMode = contentViewMode === 'raw' ? 'rendered' : 'raw'"
              class="w-6 h-6 rounded-md flex items-center justify-center transition-colors"
              :class="contentViewMode === 'rendered'
                ? 'bg-sky-500/15 text-sky-400 hover:bg-sky-500/25'
                : 'bg-slate-100 dark:bg-slate-800 text-slate-500 hover:text-slate-700 dark:hover:text-slate-300'"
              :title="contentViewMode === 'raw' ? 'Show rendered' : 'Show raw'"
            >
              <UIcon :name="contentViewMode === 'raw' ? 'i-lucide-eye' : 'i-lucide-code'" class="w-3.5 h-3.5" />
            </button>
          </div>

          <!-- Rendered markdown view -->
          <div v-if="canRenderContent && key === 'content' && contentViewMode === 'rendered'" class="relative group/code">
            <button
              @click="copyText(getRawText(val), `input-${key}`)"
              class="absolute top-2 right-2 z-10 w-6 h-6 sm:w-7 sm:h-7 rounded-md flex items-center justify-center bg-slate-200/60 dark:bg-slate-700/60 hover:bg-slate-300/80 dark:hover:bg-slate-600/80 text-slate-500 dark:text-slate-300 transition-all opacity-0 group-hover/code:opacity-100"
            >
              <UIcon :name="copiedKey === `input-${key}` ? 'i-lucide-check' : 'i-lucide-clipboard-copy'" class="w-3 h-3 sm:w-3.5 sm:h-3.5" />
            </button>
            <div
              class="text-sm leading-relaxed bg-slate-50 dark:bg-card rounded-xl p-4 overflow-y-auto border border-slate-200 dark:border-border/20 custom-scrollbar max-h-[50vh]
                prose dark:prose-invert prose-sm max-w-none
                prose-p:my-2 prose-headings:my-3 prose-li:my-1
                prose-pre:bg-slate-100 dark:prose-pre:bg-slate-900/80 prose-pre:border prose-pre:border-slate-200 dark:prose-pre:border-border/20 prose-pre:rounded-xl
                prose-code:text-sky-600 dark:prose-code:text-sky-300 prose-code:font-mono prose-code:text-[12px]
                prose-a:text-sky-400 prose-a:no-underline hover:prose-a:underline
                prose-strong:text-foreground"
              v-html="renderMarkdown(String(val))"
            />
          </div>

          <!-- Raw code view -->
          <template v-else>
            <div v-if="highlightedInputs[key]" class="relative group/code">
              <button
                @click="copyText(getRawText(val), `input-${key}`)"
                class="absolute top-2 right-2 z-10 w-6 h-6 sm:w-7 sm:h-7 rounded-md flex items-center justify-center bg-slate-700/60 hover:bg-slate-600/80 text-slate-300 transition-all opacity-0 group-hover/code:opacity-100"
              >
                <UIcon :name="copiedKey === `input-${key}` ? 'i-lucide-check' : 'i-lucide-clipboard-copy'" class="w-3 h-3 sm:w-3.5 sm:h-3.5" />
              </button>
              <div
                class="text-[11px] sm:text-[12px] bg-[#1a1a2e] dark:bg-[#121212] rounded-xl p-2.5 sm:p-3 overflow-x-auto border border-slate-200 dark:border-border/20 shiki-container custom-scrollbar max-h-[50vh]"
                v-html="highlightedInputs[key]"
              />
            </div>
            <div v-else class="relative group/code">
              <button
                v-if="typeof val === 'string' && val.length > 30"
                @click="copyText(getRawText(val), `input-${key}`)"
                class="absolute top-2 right-2 z-10 w-6 h-6 sm:w-7 sm:h-7 rounded-md flex items-center justify-center bg-slate-200/60 dark:bg-slate-700/60 hover:bg-slate-300/80 dark:hover:bg-slate-600/80 text-slate-500 dark:text-slate-300 transition-all opacity-0 group-hover/code:opacity-100"
              >
                <UIcon :name="copiedKey === `input-${key}` ? 'i-lucide-check' : 'i-lucide-clipboard-copy'" class="w-3 h-3 sm:w-3.5 sm:h-3.5" />
              </button>
              <pre class="text-[12px] sm:text-[13px] text-slate-700 dark:text-slate-300 bg-slate-50 dark:bg-card rounded-xl p-2.5 sm:p-3 overflow-x-auto whitespace-pre-wrap break-words border border-slate-200 dark:border-border/20 font-mono leading-relaxed custom-scrollbar">{{ getRawText(val) }}</pre>
            </div>
          </template>
        </div>
      </div>
    </div>

    <!-- WebSearch results -->
    <div v-if="searchResult && searchResult.links.length > 0">
      <h3 class="text-[11px] uppercase tracking-wider text-slate-500 mb-2 font-semibold">
        Sources
        <span class="text-amber-400/60 ml-1 normal-case">({{ searchResult.links.length }})</span>
      </h3>
      <div class="space-y-1">
        <a
          v-for="(link, i) in searchResult.links"
          :key="i"
          :href="link.url"
          target="_blank"
          rel="noopener"
          class="group/link flex items-start gap-2.5 px-2.5 py-2 rounded-xl border border-slate-200 dark:border-border/30 hover:border-amber-500/40 hover:bg-amber-500/5 transition-all"
        >
          <UIcon name="i-lucide-globe" class="w-3.5 h-3.5 text-amber-400/60 group-hover/link:text-amber-400 mt-0.5 shrink-0" />
          <div class="min-w-0 flex-1">
            <div class="text-[12px] sm:text-[13px] text-slate-700 dark:text-slate-200 group-hover/link:text-amber-300 leading-snug line-clamp-2">{{ link.title }}</div>
            <div class="text-[10px] sm:text-[11px] text-slate-500 font-mono mt-0.5 truncate">{{ hostname(link.url) }}</div>
          </div>
          <UIcon name="i-lucide-external-link" class="w-3.5 h-3.5 text-slate-500 group-hover/link:text-amber-400 mt-0.5 shrink-0 opacity-0 group-hover/link:opacity-100 transition-opacity" />
        </a>
      </div>
    </div>

    <div v-if="searchResult && searchResult.summary">
      <h3 class="text-[11px] uppercase tracking-wider text-slate-500 mb-2 font-semibold">Summary</h3>
      <div
        class="text-sm leading-relaxed text-slate-700 dark:text-slate-200 prose dark:prose-invert prose-sm max-w-none
          prose-p:my-2 prose-headings:my-3
          prose-strong:text-foreground"
        v-html="renderMarkdown(searchResult.summary)"
      />
    </div>

    <!-- Generic output (non-WebSearch) -->
    <div v-if="node.result && !searchResult">
      <h3 class="text-[11px] uppercase tracking-wider text-slate-500 mb-2 font-semibold">Output</h3>
      <div v-if="highlightedOutput" class="relative group/code">
        <button
          @click="copyText(node.result!, 'output')"
          class="absolute top-2 right-2 z-10 w-6 h-6 sm:w-7 sm:h-7 rounded-md flex items-center justify-center bg-slate-700/60 hover:bg-slate-600/80 text-slate-300 transition-all opacity-0 group-hover/code:opacity-100"
        >
          <UIcon :name="copiedKey === 'output' ? 'i-lucide-check' : 'i-lucide-clipboard-copy'" class="w-3 h-3 sm:w-3.5 sm:h-3.5" />
        </button>
        <div
          class="text-[11px] sm:text-[12px] bg-[#1a1a2e] dark:bg-[#121212] rounded-xl p-2.5 sm:p-3 overflow-x-auto border border-slate-200 dark:border-border/20 shiki-container custom-scrollbar max-h-[50vh]"
          v-html="highlightedOutput"
        />
      </div>
      <div v-else class="relative group/code">
        <button
          v-if="node.result.length > 30"
          @click="copyText(node.result!, 'output')"
          class="absolute top-2 right-2 z-10 w-6 h-6 sm:w-7 sm:h-7 rounded-md flex items-center justify-center bg-slate-200/60 dark:bg-slate-700/60 hover:bg-slate-300/80 dark:hover:bg-slate-600/80 text-slate-500 dark:text-slate-300 transition-all opacity-0 group-hover/code:opacity-100"
        >
          <UIcon :name="copiedKey === 'output' ? 'i-lucide-check' : 'i-lucide-clipboard-copy'" class="w-3 h-3 sm:w-3.5 sm:h-3.5" />
        </button>
        <pre class="text-[12px] sm:text-[13px] text-slate-700 dark:text-slate-300 bg-slate-50 dark:bg-card rounded-xl p-2.5 sm:p-3 overflow-x-auto whitespace-pre-wrap break-words border border-slate-200 dark:border-border/20 font-mono leading-relaxed custom-scrollbar">{{ node.result }}</pre>
      </div>
    </div>

    <div v-if="inputEntries.length === 0 && !node.result && !hasDiff" class="flex flex-col items-center justify-center text-slate-600 gap-2 pt-8">
      <UIcon name="i-lucide-inbox" class="w-6 h-6 sm:w-8 sm:h-8" />
      <p class="text-xs sm:text-sm">No details yet</p>
    </div>
  </div>
</template>
