export default defineAppConfig({
    ui: {
        primary: 'sky',
        gray: 'slate',
        button: {
            default: {
                size: 'md',
            },
            rounded: 'rounded-full',
            font: 'font-semibold tracking-tight',
            variant: {
                solid: 'shadow-md shadow-primary-500/20 hover:shadow-primary-500/30 transition-all',
                ghost: 'hover:bg-slate-100 dark:hover:bg-gray-800/50 transition-colors',
                soft: 'bg-primary-500/10 text-primary-400 hover:bg-primary-500/20',
            }
        },
        input: {
            default: {
                size: 'md',
            },
            rounded: 'rounded-xl',
            color: {
                white: {
                    outline: 'shadow-sm bg-white dark:bg-gray-900/50 text-gray-900 dark:text-gray-100 ring-1 ring-inset ring-gray-200 dark:ring-gray-800 focus:ring-2 focus:ring-primary-500'
                }
            }
        },
        slideover: {
            overlay: {
                background: 'bg-gray-500/20 dark:bg-gray-950/40 backdrop-blur-sm'
            },
            background: 'bg-white/95 dark:bg-gray-900/95 backdrop-blur-2xl supports-[backdrop-filter]:bg-white/80 dark:supports-[backdrop-filter]:bg-gray-900/80',
        },
        badge: {
            rounded: 'rounded-full'
        }
    }
})
