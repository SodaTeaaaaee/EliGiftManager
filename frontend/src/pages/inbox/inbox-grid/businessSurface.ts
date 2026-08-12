/** 收件箱业务面三态：全部 / 会员权益 / 零售订单。纯函数模块，供 InboxPage 与波内导入页共用。 */
export type BusinessSurface = 'all' | 'membership_entitlement' | 'retail_order'

export function surfaceFromKinds(kinds: readonly string[]): BusinessSurface {
  if (kinds.length === 1 && (kinds[0] === 'membership_entitlement' || kinds[0] === 'retail_order')) {
    return kinds[0]
  }
  return 'all'
}

export function kindsFromSurface(surface: BusinessSurface): string[] {
  return surface === 'all' ? [] : [surface]
}
