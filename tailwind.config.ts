import type { Config } from 'tailwindcss'
import { fontFamily } from 'tailwindcss/defaultTheme'

export default {
    darkMode: 'class',
    content: [
        './app/components/**/*.{vue,js,ts}',
        './app/layouts/**/*.vue',
        './app/pages/**/*.vue',
        './app/app.vue',
    ],
    theme: {
        container: {
            center: true,
            padding: '2rem',
            screens: {
                '2xl': '1400px',
            },
        },
        extend: {
            colors: {
                border: 'rgb(var(--border) / <alpha-value>)',
                input: 'rgb(var(--input) / <alpha-value>)',
                ring: 'rgb(var(--ring) / <alpha-value>)',
                background: 'rgb(var(--background) / <alpha-value>)',
                foreground: 'rgb(var(--foreground) / <alpha-value>)',
                card: 'rgb(var(--card) / <alpha-value>)',
                'card-foreground': 'rgb(var(--card-foreground) / <alpha-value>)',
            },
            borderRadius: {
                xl: 'calc(var(--radius) + 4px)',
                lg: 'var(--radius)',
                md: 'calc(var(--radius) - 2px)',
                sm: 'calc(var(--radius) - 4px)',
            },
            fontFamily: {
                sans: ['Inter', ...fontFamily.sans],
                heading: ['Plus Jakarta Sans', ...fontFamily.sans],
                mono: ['Space Mono', ...fontFamily.mono],
            },
        }
    },
    plugins: [],
} satisfies Config
