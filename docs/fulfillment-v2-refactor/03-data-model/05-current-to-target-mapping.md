# 当前模型到目标模型的映射

本文件用于帮助实施时识别哪些结构可以保留、哪些结构需要泛化、哪些旧术语需要停用。

## 6. 当前结构到目标结构的映射

### 6.1 当前实体与目标实体映射

| 当前实体 | 目标语义 | 处理策略 |
| --- | --- | --- |
| `Member` | `CustomerProfile` 的过渡实现 | 短期保留表名，先扩展语义 |
| `MemberNickname` | `CustomerProfile` 的昵称历史 | 保留，后续可并入 profile/identity 辅助表 |
| `MemberAddress` | `CustomerAddress` | 保留并增强 |
| `Wave` | `Wave` | 保留并扩充生命周期语义 |
| `WaveMember` | `WaveParticipantSnapshot` | 泛化，不再只承载会员 |
| `ProductMaster` | `ProductMaster` | 直接保留 |
| `Product` | `Wave Product Snapshot` | 直接保留 |
| `ProductTag` | 分配规则层 | 会员权益波次继续使用；零售订单波次可弱化 |
| `DispatchRecord` | `FulfillmentLine` | 演进，不再继续承担全部外部状态 |
| `TemplateConfig` | 模板配置层 | 保留，但需升级模板类型和能力模型 |

### 6.2 关键保留原则

以下设计原则必须保持：

1. 历史波次必须是快照，不受全局实体后续变动污染
2. 全局商品主档和波次商品快照必须继续分层
3. 履约真相必须有单一归宿
4. 工厂执行和物流回填不能只靠导入导出瞬时脚本

---

