import { defineComponent, h, type PropType, type VNodeChild } from 'vue'

type StubMap = Record<string, ReturnType<typeof defineComponent>>

function renderChildren(children: VNodeChild[]) {
  return children.filter((child) => child !== null && child !== undefined && child !== false) as VNodeChild[]
}

function toDisplayValue(value: unknown) {
  return value == null ? '' : String(value)
}

function createPassthroughStub(tag = 'div', name = tag) {
  return defineComponent({
    name,
    inheritAttrs: false,
    setup(_, { attrs, slots }) {
      return () => h(tag, {
        ...attrs,
        'data-stub': attrs['data-stub'] || name
      }, slots.default?.())
    }
  })
}

const UApp = createPassthroughStub('div', 'UApp')
const UContainer = createPassthroughStub('div', 'UContainer')
const USkeleton = createPassthroughStub('div', 'USkeleton')
const UIcon = createPassthroughStub('span', 'UIcon')

const ClientOnly = defineComponent({
  name: 'ClientOnly',
  setup(_, { slots }) {
    return () => slots.default?.() ?? slots.fallback?.()
  }
})

const UCard = defineComponent({
  name: 'UCard',
  inheritAttrs: false,
  setup(_, { attrs, slots }) {
    return () => h('section', { ...attrs, 'data-stub': 'UCard' }, renderChildren([
      slots.header ? h('div', { 'data-slot': 'header' }, slots.header()) : null,
      slots.default?.()
    ]))
  }
})

const UAlert = defineComponent({
  name: 'UAlert',
  inheritAttrs: false,
  props: {
    title: {
      type: String,
      default: ''
    },
    description: {
      type: String,
      default: ''
    },
    color: {
      type: String,
      default: ''
    }
  },
  setup(props, { attrs, slots }) {
    return () => h('div', { ...attrs, role: 'alert', 'data-color': props.color || undefined }, renderChildren([
      props.title ? h('p', { class: 'alert-title' }, props.title) : null,
      slots.default?.(),
      slots.description
        ? h('div', { class: 'alert-description' }, slots.description())
        : props.description
          ? h('p', { class: 'alert-description' }, props.description)
          : null
    ]))
  }
})

const UBadge = defineComponent({
  name: 'UBadge',
  inheritAttrs: false,
  props: {
    color: {
      type: String,
      default: ''
    }
  },
  setup(props, { attrs, slots }) {
    return () => h('span', { ...attrs, 'data-stub': 'UBadge', 'data-color': props.color || undefined }, slots.default?.())
  }
})

const UButton = defineComponent({
  name: 'UButton',
  inheritAttrs: false,
  props: {
    label: {
      type: String,
      default: ''
    },
    to: {
      type: String,
      default: ''
    },
    type: {
      type: String,
      default: 'button'
    },
    disabled: {
      type: Boolean,
      default: false
    },
    loading: {
      type: Boolean,
      default: false
    }
  },
  emits: ['click'],
  setup(props, { attrs, emit, slots }) {
    return () => {
      const content = renderChildren([
        props.label,
        slots.default?.()
      ])

      if (props.to) {
        return h('a', {
          ...attrs,
          href: props.to,
          'aria-disabled': props.disabled ? 'true' : undefined,
          'data-loading': props.loading ? 'true' : undefined,
          onClick: (event: MouseEvent) => {
            if (props.disabled) {
              event.preventDefault()
              return
            }
            emit('click', event)
          }
        }, content)
      }

      return h('button', {
        ...attrs,
        type: props.type,
        disabled: props.disabled,
        'data-loading': props.loading ? 'true' : undefined,
        onClick: (event: MouseEvent) => emit('click', event)
      }, content)
    }
  }
})

const USelect = defineComponent({
  name: 'USelect',
  inheritAttrs: false,
  props: {
    modelValue: {
      type: [String, Number],
      default: ''
    },
    items: {
      type: Array as () => Array<Record<string, unknown>>,
      default: () => []
    },
    valueKey: {
      type: String,
      default: 'value'
    },
    labelKey: {
      type: String,
      default: 'label'
    },
    disabled: {
      type: Boolean,
      default: false
    },
    placeholder: {
      type: String,
      default: ''
    }
  },
  emits: ['update:modelValue'],
  setup(props, { attrs, emit }) {
    return () => h('select', {
      ...attrs,
      disabled: props.disabled,
      value: toDisplayValue(props.modelValue),
      onChange: (event: Event) => {
        emit('update:modelValue', (event.target as HTMLSelectElement).value)
      }
    }, renderChildren([
      props.placeholder
        ? h('option', { value: '' }, props.placeholder)
        : null,
      ...props.items.map((item) => h('option', {
        value: toDisplayValue(item[props.valueKey])
      }, toDisplayValue(item[props.labelKey])))
    ]))
  }
})

const UInput = defineComponent({
  name: 'UInput',
  inheritAttrs: false,
  props: {
    modelValue: {
      type: [String, Number],
      default: ''
    },
    type: {
      type: String,
      default: 'text'
    },
    disabled: {
      type: Boolean,
      default: false
    },
    placeholder: {
      type: String,
      default: ''
    },
    id: {
      type: String,
      default: ''
    },
    autocomplete: {
      type: String,
      default: ''
    }
  },
  emits: ['update:modelValue'],
  setup(props, { attrs, emit }) {
    return () => h('input', {
      ...attrs,
      id: props.id || attrs.id,
      type: props.type,
      value: toDisplayValue(props.modelValue),
      disabled: props.disabled,
      placeholder: props.placeholder,
      autocomplete: props.autocomplete,
      onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value)
    })
  }
})

const UTextarea = defineComponent({
  name: 'UTextarea',
  inheritAttrs: false,
  props: {
    modelValue: {
      type: String,
      default: ''
    },
    rows: {
      type: Number,
      default: 3
    },
    placeholder: {
      type: String,
      default: ''
    }
  },
  emits: ['update:modelValue'],
  setup(props, { attrs, emit }) {
    return () => h('textarea', {
      ...attrs,
      rows: props.rows,
      placeholder: props.placeholder,
      value: props.modelValue,
      onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLTextAreaElement).value)
    }, props.modelValue)
  }
})

const UCheckbox = defineComponent({
  name: 'UCheckbox',
  inheritAttrs: false,
  props: {
    modelValue: {
      type: Boolean,
      default: false
    },
    label: {
      type: String,
      default: ''
    },
    disabled: {
      type: Boolean,
      default: false
    }
  },
  emits: ['update:modelValue'],
  setup(props, { attrs, emit }) {
    return () => h('label', { ...attrs }, [
      h('input', {
        type: 'checkbox',
        checked: props.modelValue,
        disabled: props.disabled,
        onChange: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).checked)
      }),
      props.label
    ])
  }
})

const UFormField = defineComponent({
  name: 'UFormField',
  inheritAttrs: false,
  props: {
    label: {
      type: String,
      default: ''
    },
    name: {
      type: String,
      default: ''
    },
    description: {
      type: String,
      default: ''
    }
  },
  setup(props, { attrs, slots }) {
    return () => h('div', {
      ...attrs,
      'data-stub': 'UFormField',
      'data-field-name': props.name || undefined
    }, renderChildren([
      props.label ? h('label', { class: 'form-field-label' }, props.label) : null,
      slots.default?.(),
      props.description ? h('p', { class: 'form-field-description' }, props.description) : null
    ]))
  }
})

const UForm = defineComponent({
  name: 'UForm',
  inheritAttrs: false,
  props: {
    state: {
      type: Object as () => Record<string, unknown>,
      required: true
    },
    validate: {
      type: Function as PropType<(state: Record<string, unknown>) => Array<unknown>>,
      default: undefined
    }
  },
  emits: ['submit'],
  setup(props, { attrs, emit, slots }) {
    return () => h('form', {
      ...attrs,
      onSubmit: (event: Event) => {
        event.preventDefault()
        const errors = props.validate?.(props.state) ?? []
        if (!errors.length) {
          emit('submit', { data: props.state })
        }
      }
    }, slots.default?.())
  }
})

const UHeader = defineComponent({
  name: 'UHeader',
  inheritAttrs: false,
  props: {
    title: {
      type: String,
      default: ''
    },
    to: {
      type: String,
      default: ''
    }
  },
  setup(props, { attrs, slots }) {
    return () => h('header', { ...attrs, 'data-stub': 'UHeader' }, renderChildren([
      props.title ? h('div', { 'data-slot': 'left' }, [
        h(props.to ? 'a' : 'span', props.to ? { href: props.to } : {}, props.title)
      ]) : null,
      slots.default ? h('div', { 'data-slot': 'center' }, slots.default()) : null,
      slots.right ? h('div', { 'data-slot': 'right' }, slots.right()) : null,
      slots.body ? h('div', { 'data-slot': 'body' }, slots.body()) : null
    ]))
  }
})

const UNavigationMenu = defineComponent({
  name: 'UNavigationMenu',
  inheritAttrs: false,
  props: {
    items: {
      type: Array as () => Array<Record<string, unknown>>,
      default: () => []
    },
    orientation: {
      type: String,
      default: 'horizontal'
    }
  },
  setup(props, { attrs }) {
    return () => h('nav', {
      ...attrs,
      'data-stub': 'UNavigationMenu',
      'data-orientation': props.orientation
    }, props.items.map((item) => h('a', {
      href: toDisplayValue(item.to),
      'data-active': item.active ? 'true' : 'false',
      'data-key': toDisplayValue(item.key)
    }, toDisplayValue(item.label))))
  }
})

const VChart = defineComponent({
  name: 'VChart',
  inheritAttrs: false,
  props: {
    option: {
      type: Object,
      default: undefined
    }
  },
  setup(props, { attrs }) {
    return () => h('div', {
      ...attrs,
      'data-stub': 'VChart',
      'data-option': props.option ? JSON.stringify(props.option) : ''
    })
  }
})

const baseStubs: StubMap = {
  UApp,
  UContainer,
  UCard,
  UAlert,
  UBadge,
  UButton,
  USelect,
  UInput,
  UTextarea,
  UCheckbox,
  UFormField,
  UForm,
  UHeader,
  UNavigationMenu,
  USkeleton,
  UIcon,
  ClientOnly,
  VChart
}

export function createNuxtUiStubs(overrides: StubMap = {}): StubMap {
  return {
    ...baseStubs,
    ...overrides
  }
}
