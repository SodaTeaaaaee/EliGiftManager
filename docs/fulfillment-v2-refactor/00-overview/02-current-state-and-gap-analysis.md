# 当前现状与缺口分析

> 旧系统为什么不适合继续增量修补。这里的"当前系统"指 greenfield 重建前的历史基线。

## 旧核心模型

`Member`、`MemberNickname`、`MemberAddress`、`Wave`、`WaveMember`、`ProductMaster`、`Product`、`ProductTag`、`DispatchRecord`、`TemplateConfig`。

已验证的健康分层：`ProductMaster + Product`（全局主档+波次快照）、`Member + WaveMember`（全局实体+波次快照）。V2 泛化而非推翻。

## 旧工作流

`导入会员 → WaveMember → 导入商品 → ProductTag → ReconcileWave → 绑定地址 → 导出 CSV`

旧 tag 系统本质：商品中心的规则编辑器 + 会员权益分配系统 + 用户级例外覆盖。预览页已隐含"规则生成层→人工调整层→履约真相层"三层。

## 旧状态模型

`DispatchRecord.Status`：`pending` / `pending_address` / `exported`。`Wave.Status` 由聚合推导。

无法表达：已提交未发货、部分发货、已回传未回填、回填失败。

## 7 个结构性缺口

1. **缺少上游需求单抽象**：会员无订单号 vs 零售有订单号，无统一结构承接
2. **缺少供应商执行单抽象**：无持久化"发给哪个工厂、哪次导出、外部订单号"
3. **缺少发货/物流抽象**：快递单号无处稳定落库
4. **缺少来源渠道回填任务抽象**：物流拿到 ≠ 回填完成
5. **`platform` 语义过载**：会员来源/商品工厂/来源渠道/承运商混用一个字段
6. **`DispatchRecord` 承担过多**：同时承担分配结果、地址状态、工厂导出状态
7. **缺少权益判定/输入采集/路由层**：无法区分"权益是否成立"、"输入是否收齐"、"本系统是否接手"

这些缺口已在 V2 中全部解决，见 `internal/domain/models.go`。
