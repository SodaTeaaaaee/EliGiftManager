# 平台维度与 Profile 定位

> 平台相关讨论：应该分的维度、`IntegrationProfile` 的定位。

## 平台维度

V2 区分 5 个平台维度（见 `02-ubiquitous-language.md` § 4.2）：

- `identity_platform` — 用户身份来源
- `source_channel` — 平台供应商
- `source_surface` — 业务面
- `supplier_platform` — 工厂/供应商
- `carrier_code` — 物流承运商

## 分平台的必要性

- **边界层**：非常必要（`source_channel` + `source_surface` 必须拆开）
- **核心履约层**：不应让平台差异无限渗透

### 波次内共性很强

`FulfillmentLine` 不应按平台拆成多套模型，`Wave` 也不因来源不同裂变。

### 波次前后两端差异很大

差异在：需求导入、外部编号、身份识别、规则推导、物流回填方式。封装在需求导入层、`IntegrationProfile`/连接器层、回填层。

### 真正该分的维度

1. `source_channel` — 哪个平台供应商
2. `source_surface` — 哪个业务面
3. `demand_kind` — 产生什么语义的需求
4. `identity_strategy` — 如何识别履约对象
5. `strategy fields + capability flags + connector binding` — 默认策略、正交能力、连接器

## `IntegrationProfile` 定位

某来源渠道供应商的某业务面的统一配置入口。`profile_key` 格式：`<source_channel>.<source_surface>`。

回答的问题：
- 这个来源是哪个业务面
- 归类为哪种需求语义
- 义务由资格还是订单成立
- 如何识别履约对象
- 权益判定权威来源
- 是否支持物流回填
- 绑定哪些文档模板与连接器

比模板更高一层：模板回答"CSV 长什么样"，Profile 回答"这个来源到底是什么"。

### Profile 结构收敛

- strategy 字段表达主流程语义
- capability flag 表达不与 strategy 重复的正交能力
- connector binding 表达外部交互实现入口
- 不再同时保留通用 capabilities blob + 重复的 supports_* 布尔值 + strategy 字段
