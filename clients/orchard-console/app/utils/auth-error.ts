export function resolveAuthErrorMessage(value: unknown) {
  switch (String(value || '').trim().toLowerCase()) {
    case 'invalid_request':
      return '登录状态已失效或回调参数缺失，请重新发起登录。'
    case 'auth_unavailable':
      return '登录服务暂时不可用，请稍后重试。'
    case 'access_denied':
      return '身份提供方拒绝了本次登录，请重新尝试或联系管理员。'
    case 'login_failed':
      return '登录未完成，请重新尝试。'
    default:
      return ''
  }
}
