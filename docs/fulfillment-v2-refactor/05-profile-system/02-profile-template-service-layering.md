# Profile、模板与 Service 的分层

本文件定义 profile、模板、service 三层分工，以及为什么 profile 应作为模板系统的上层。

### 9.3 Profile、模板、Service 的三层分工

V2 推荐明确三层结构：

1. `IntegrationProfile`

- 定义来源业务面
- 定义需求类型
- 定义义务触发方式
- 定义权益判定权威
- 定义领取/表单/协商输入模式
- 定义能力边界
- 定义闭环策略
- 选择应绑定的模板集合

2. `DocumentTemplate`

- 定义具体文档字段映射
- 定义列顺序
- 定义 CSV / Excel / JSON 结构

3. `Service / Handler`

- 执行真实业务逻辑
- 处理导入、导出、回填、重试、异常分支

这三层的关系应当是：

- Profile 决定“怎么理解这个来源”
- Template 决定“怎么读写这个来源的文档”
- Service 决定“实际怎么执行这套流程”

### 9.4 为什么 Profile 是模板系统的上层

同一个来源业务面往往不只需要一个模板。

例如：

- 一个 `bilibili.creator_commerce` profile 可能需要：
  - 订单导入模板
  - 工厂导出模板
  - 工厂发货回传模板
  - 物流回填模板

这意味着：

- 模板是文档级对象
- Profile 是来源业务面级对象

因此更合理的结构是：

- 一个 Profile 绑定多个 Template
- 而不是让一个 Template 自己承担全部业务职责

这里还要补一个当前阶段的产品决策：

- V2 首版不计划继续保留“旧模板入口”和“新 profile 入口”长期并行
- 更推荐直接把入口升级成 profile-centric 的配置方式
- 模板仍然存在，但它退回到 profile 下面的文档结构子层

### 9.5 模板与连接器分离

后续架构里必须明确：

- 模板：
  - 负责字段映射
  - 负责列顺序
  - 负责 CSV/Excel 结构解释
- 连接器：
  - 负责平台能力
  - 负责导入导出方式
  - 负责 API / CSV / 手工上传差异
