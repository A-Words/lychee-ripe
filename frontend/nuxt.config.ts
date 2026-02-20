export default defineNuxtConfig({
  compatibilityDate: '2025-05-15',
  devtools: { enabled: true },
  ssr: false,
  modules: ['@nuxt/ui'],
  css: ['~/assets/css/main.css'],
  ui: {
    fonts: true,
  },
  runtimeConfig: {
    public: {
      gatewayBase: 'http://127.0.0.1:9000',
    },
  },
  nitro: {
    output: {
      publicDir: 'dist',
    },
  },
  devServer: {
    host: '127.0.0.1',
    port: 3000,
  },
  vite: {
    clearScreen: false,
    envPrefix: ['VITE_', 'TAURI_'],
    server: {
      strictPort: true,
    },
  },
  ignore: ['**/src-tauri/**'],
})
