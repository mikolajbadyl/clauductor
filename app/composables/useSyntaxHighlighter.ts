import { createHighlighter } from 'shiki'

let shikiPromise: Promise<any> | null = null

export function useSyntaxHighlighter() {
    function getSharedHighlighter() {
        if (!shikiPromise) {
            shikiPromise = createHighlighter({
                themes: ['tokyo-night'],
                langs: ['text', 'typescript', 'javascript', 'python', 'go', 'json', 'html', 'css', 'vue', 'bash', 'markdown']
            })
        }
        return shikiPromise
    }

    async function highlight(code: string, filepath: string = '') {
        if (!code) return ''

        let lang = 'text'
        const path = filepath.toLowerCase()
        if (path.endsWith('.ts') || path.endsWith('.vue')) lang = 'typescript'
        else if (path.endsWith('.js')) lang = 'javascript'
        else if (path.endsWith('.py')) lang = 'python'
        else if (path.endsWith('.go')) lang = 'go'
        else if (path.endsWith('.json')) lang = 'json'
        else if (path.endsWith('.html')) lang = 'html'
        else if (path.endsWith('.css')) lang = 'css'
        else if (path.endsWith('.sh')) lang = 'bash'
        else if (path.endsWith('.md')) lang = 'markdown'

        try {
            const highlighter = await getSharedHighlighter()
            return highlighter.codeToHtml(code, {
                lang,
                theme: 'tokyo-night'
            })
        } catch (e) {
            console.warn('Shiki highlight failed, falling back to plaintext', e)
            return `<pre><code>${code.replace(/</g, '&lt;').replace(/>/g, '&gt;')}</code></pre>`
        }
    }

    return { highlight }
}
