// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: true },
  ssr: false,
  modules: ['@nuxt/ui'],
  css: ['~/assets/css/main.css'],
  runtimeConfig: {
    public: {
      gatewayBase: process.env.NUXT_PUBLIC_GATEWAY_BASE || 'http://127.0.0.1:9000',
      authMode: process.env.NUXT_PUBLIC_AUTH_MODE || 'disabled',
      oidcIssuerUrl: process.env.NUXT_PUBLIC_OIDC_ISSUER_URL || '',
      oidcTauriClientId: process.env.NUXT_PUBLIC_OIDC_TAURI_CLIENT_ID || '',
      oidcScope: process.env.NUXT_PUBLIC_OIDC_SCOPE || 'openid profile email'
    }
  },
  vite: {
    clearScreen: false,
    envPrefix: ['VITE_', 'TAURI_'],
    server: {
      strictPort: true
    }
  },
  ignore: ['**/src-tauri/**']
})
