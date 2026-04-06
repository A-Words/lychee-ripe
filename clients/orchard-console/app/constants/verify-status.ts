import type { VerifyStatus } from '~/types/trace'

type VerifyColor = 'primary' | 'success' | 'warning' | 'error'

export interface VerifyStatusMeta {
  label: string
  description: string
  color: VerifyColor
}

export const VERIFY_STATUS_META: Record<VerifyStatus, VerifyStatusMeta> = {
  recorded: {
    label: '数据库存证',
    description: '批次已入库并可在系统内查询',
    color: 'primary'
  },
  pass: {
    label: '验证通过',
    description: '链上摘要与库内摘要一致',
    color: 'success'
  },
  pending: {
    label: '待上链',
    description: '批次已保存，等待链上锚定',
    color: 'warning'
  },
  fail: {
    label: '验证失败',
    description: '摘要不一致或链上校验失败',
    color: 'error'
  }
}
