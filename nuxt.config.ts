// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: true },
  devServer: {
    host: '0.0.0.0',
    allowedHosts: ['mbserver'],
  },
  ssr: false,
  modules: ['@nuxt/ui', '@nuxtjs/tailwindcss'],
  colorMode: {
    preference: 'system',
    fallback: 'dark',
  },
  css: ['~/assets/css/main.css', '@vue-flow/core/dist/style.css', '@vue-flow/core/dist/theme-default.css'],
  app: {
    head: {
      title: 'Clauductor',
      meta: [
        { name: 'viewport', content: 'width=device-width, initial-scale=1, maximum-scale=1, viewport-fit=cover, interactive-widget=resizes-content' },
        { name: 'theme-color', content: '#f8fafc', media: '(prefers-color-scheme: light)' },
        { name: 'theme-color', content: '#0a0e1a', media: '(prefers-color-scheme: dark)' },
        { name: 'apple-mobile-web-app-capable', content: 'yes' },
        { name: 'apple-mobile-web-app-status-bar-style', content: 'black-translucent' },
      ],
      link: [
        { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
        { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' },
        { rel: 'stylesheet', href: 'https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=Plus+Jakarta+Sans:wght@500;600;700&family=Space+Mono:wght@400;700&display=swap' }
      ]
    }
  }
})
