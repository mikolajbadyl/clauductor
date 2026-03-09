export const toolColorClasses: Record<string, string> = {
  Read: 'bg-sky-500/10 border-sky-500/30 text-sky-400',
  Write: 'bg-emerald-500/10 border-emerald-500/30 text-emerald-400',
  Edit: 'bg-emerald-500/10 border-emerald-500/30 text-emerald-400',
  Bash: 'bg-violet-500/10 border-violet-500/30 text-violet-400',
  Glob: 'bg-sky-500/10 border-sky-500/30 text-sky-400',
  Grep: 'bg-sky-500/10 border-sky-500/30 text-sky-400',
  MultiTool: 'bg-slate-500/5 border-slate-500/20 text-slate-400',
  Task: 'bg-violet-500/10 border-violet-500/30 text-violet-400',
  TodoWrite: 'bg-violet-500/10 border-violet-500/30 text-violet-400',
  WebSearch: 'bg-amber-500/10 border-amber-500/30 text-amber-400',
  WebFetch: 'bg-amber-500/10 border-amber-500/30 text-amber-400',
}

export const toolIcons: Record<string, string> = {
  Read: 'i-lucide-file-text',
  Write: 'i-lucide-file-plus',
  Edit: 'i-lucide-pen-square',
  Bash: 'i-lucide-terminal',
  Glob: 'i-lucide-search',
  Grep: 'i-lucide-search-check',
  MultiTool: 'i-lucide-wrench',
  Task: 'i-lucide-list-ordered',
  WebSearch: 'i-lucide-globe',
  WebFetch: 'i-lucide-globe',
  prompt: 'i-lucide-message-square',
  response: 'i-lucide-cpu',
}

export interface SearchLink {
  title: string
  url: string
}

export function parseSearchResult(result: string): { query: string; links: SearchLink[]; summary: string } {
  let query = ''
  let links: SearchLink[] = []
  let summary = ''

  const queryMatch = result.match(/Web search results for query:\s*"([^"]+)"/)
  if (queryMatch?.[1]) query = queryMatch[1]

  const linksIdx = result.indexOf('Links:')
  if (linksIdx !== -1) {
    const afterLinks = result.slice(linksIdx + 6).trim()
    const bracketStart = afterLinks.indexOf('[')
    if (bracketStart !== -1) {
      let depth = 0
      let end = -1
      for (let i = bracketStart; i < afterLinks.length; i++) {
        if (afterLinks[i] === '[') depth++
        else if (afterLinks[i] === ']') { depth--; if (depth === 0) { end = i; break } }
      }
      if (end !== -1) {
        const jsonStr = afterLinks.slice(bracketStart, end + 1)
        try { links = JSON.parse(jsonStr) } catch {}
        summary = afterLinks.slice(end + 1).trim()
      }
    }
  }

  return { query, links, summary }
}

export function toolLabel(name: string, input?: Record<string, any>): string {
  if (!input) return name
  switch (name) {
    case 'Read':
    case 'Write':
    case 'Edit':
      return input.file_path?.split('/').pop() || name
    case 'Bash':
      return `$ ${(input.command || '').slice(0, 50)}`
    case 'Glob':
    case 'Grep':
      return input.pattern || name
    case 'WebSearch':
      return input.query ? `Search: ${input.query.slice(0, 40)}` : name
    default:
      return name
  }
}
