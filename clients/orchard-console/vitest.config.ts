import { fileURLToPath } from 'node:url'
import { defineVitestProject } from '@nuxt/test-utils/config'
import { defineConfig } from 'vitest/config'

const alias = {
  '~': fileURLToPath(new URL('./app', import.meta.url)),
  '@': fileURLToPath(new URL('./app', import.meta.url))
}

const nuxtProject = await defineVitestProject({
  resolve: {
    alias
  },
  test: {
    name: 'nuxt',
    include: ['tests/nuxt/**/*.spec.ts'],
    environment: 'nuxt',
    environmentOptions: {
      nuxt: {
        rootDir: fileURLToPath(new URL('./', import.meta.url)),
        domEnvironment: 'happy-dom'
      }
    }
  }
})

export default defineConfig({
  resolve: {
    alias
  },
  test: {
    projects: [
      {
        resolve: {
          alias
        },
        test: {
          name: 'unit',
          environment: 'node',
          include: ['tests/unit/**/*.spec.ts']
        }
      },
      nuxtProject
    ]
  }
})
