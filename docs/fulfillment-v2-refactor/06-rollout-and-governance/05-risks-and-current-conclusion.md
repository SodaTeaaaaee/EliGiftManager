# 风险与当前结论

## 主要风险

| # | 风险 | 缓解 |
|---|------|------|
| 1 | 在旧术语上继续硬补 | 已完成 V2 重建 |
| 2 | 过早重命名数据库表 | 已采用 greenfield 直接建新表 |
| 3 | 用单一进度条伪装复杂流程 | 多维状态 + 漏斗指标 |
| 4 | 强迫所有需求走 tag 规则系统 | `policy_driven` + `demand_driven` 双路径 |
| 5 | 为适配零售让会员分配 UX 退化 | 会员分配体验作为独立约束 |
| 6 | 把辅助系统误做成系统外履约总账 | `RoutingDisposition` 表达路由决策，不伪造外部状态 |
| 7 | 缺乏完整历史时本地重算平台权威成就 | `EntitlementAuthority = upstream_platform` |
| 8 | 跨步骤跳转伴随隐式重写 | 非破坏性编辑为核心约束 |
| 9 | 试图一次性设计完美 `Wave Overview` | 假设驱动，先最小可用再迭代 |
| 10 | 把辅助工作区对象误做成不可覆盖历史账本 | 理解为最新已知辅助对象，偏离用提示不用硬锁 |
| 11 | 把语义问题误交给反馈循环 | 语义边界先定，视图排布可反馈 |
| 12 | 让 `Adjustment Review` 长成第二套动态规则引擎 | 动态 selector 严格留在 `Membership Allocation` |
| 13 | 用线性 undo 栈破坏分支保留 | 必须保留树状分支 |
| 14 | 每步完整快照粗暴实现历史 | patch + checkpoint + pin + GC |
| 15 | 全局快捷键劫持文本输入撤销 | 文本输入上下文优先原生 undo |

## 当前结论

主线：`Demand → Entitlement Resolution/Routing → Wave → FulfillmentLine → SupplierOrder → Shipment → ChannelSync/Manual Closure`

最值得保留：`ProductMaster + Product` 双层、`Wave` 业务批次边界、`ReconcileWave` 收敛思路。

最需要停止扩张：单一 `status` 字段、`platform` 语义过载、导出即终点、共享调整层膨胀、线性 undo 栈。
