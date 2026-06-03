# Profile、模板与 Service 的分层

## 三层分工

| 层 | 职责 | 模式 |
|----|------|------|
| `IntegrationProfile` | 来源业务面定义、需求类型、策略、能力、闭环策略、连接器绑定 | Strategy Selection + Capability Declaration |
| `DocumentTemplate` | 文档字段映射、列顺序、CSV/Excel/JSON 结构 | Document Schema / Mapping |
| Service/Handler | 真实业务逻辑、导入/导出/回填/重试/异常 | Adapter + Process Logic |

## Profile 是模板的上层

同一业务面需要多个模板（需求导入、工厂导出、发货回传、物流回填）。一个 Profile 绑定多个 Template。

V2 首版直接升级为 profile-centric 入口，不长期并行旧模板入口。

## 模板与连接器分离

- 模板：字段映射、列顺序、CSV/Excel 结构
- 连接器：平台真实交互、API/CSV/手工上传差异、失败处理

Profile 声明"支持什么"，不承担"具体怎么做"。

## Profile 与模板编辑接入统一历史

全应用共用 history 基础设施，优先做稳 `wave` scope。Profile 编辑有自己的 `HistoryScope(scope_type = template)`。

## Profile 版本演进策略

- **已关闭波次**：不受 profile 变更影响
- **活跃波次**：使用创建时绑定的 profile，直到用户显式刷新（刷新作为 `HistoryNode`，可撤销）
- **首版不引入版本号**：通过 `DemandDocument.integration_profile_id` + `raw_payload` 保留解释依据
