export function formatPercent(value: number, digits = 1): string {
  if (!Number.isFinite(value)) {
    return '--'
  }

  return `${(value * 100).toFixed(digits)}%`
}

export function formatDateTime(value: string | null | undefined): string {
  if (!value) {
    return '--'
  }

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }

  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  }).format(date)
}

export function truncateTxHash(value: string | null | undefined, head = 10, tail = 8): string {
  if (!value) {
    return '--'
  }

  if (value.length <= head + tail + 3) {
    return value
  }

  return `${value.slice(0, head)}...${value.slice(-tail)}`
}
