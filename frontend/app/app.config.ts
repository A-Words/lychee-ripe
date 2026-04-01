export default defineAppConfig({
  ui: {
    colors: {
      primary: 'emerald',
      secondary: 'sky',
      neutral: 'slate'
    },
    header: {
      slots: {
        root: 'bg-default/90 border-b border-default backdrop-blur h-(--ui-header-height) sticky top-0 z-50',
        title: 'text-highlighted font-semibold text-base sm:text-lg',
        center: 'hidden md:flex min-w-0',
        toggle: 'md:hidden',
        body: 'bg-default p-4 sm:p-6 overflow-y-auto'
      }
    },
    navigationMenu: {
      slots: {
        label: 'text-highlighted',
        link: 'text-default',
        childLink: 'text-default'
      }
    }
  }
})
