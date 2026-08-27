# 对齐引擎是纯函数深模块，模板锚定语义字典 v3

外部格式的全部知识收进 `internal/app/alignment` 一个纯函数包：`ReadRows` 读 csv/xlsx/xls 成记录，`Parse` 按映射配置产出语义行，`Render` 把语义行写回平台文件。接口止步于字节和配置，不触碰 Store、Workspace 或持久化——外部世界怎样表达一个订单，只有这一个包需要知道（locality）。四个消费方共享同一引擎：文件导入、模板测试（PreviewTemplate 用真实样例数据跑同一 Parse）、工厂订单与渠道回写的输出渲染、发货回传导入。引擎修一处，四方受益（leverage）。因为纯函数，测试面就是样例文件字节对语义行的断言，不需要数据库或 Wails 运行时。

模板配置 v3 锚定封闭语义字典：映射目标只能是字典语义键（`source.document_no`、`product.alias_id` 等），不是任意列名或数据库列；布局侧同理按语义键声明列序和表头。字段处理是七个命名转换器（trim、strip_quotes、parseDate、mapEnum、normalizePhone、joinAddress、splitSkuQuantity），注册表是唯一真值，用户不写脚本。其中 splitSkuQuantity 是行展开能力而非值改写，独占 `SplitSkuQuantity` 字段承载，混进 Transforms 链是配置错误。格式范围 csv（BOM + 标准引用）、xlsx（excelise）、xls 只读（extrame/xls）：xls 支持只为吃下工厂侧遗留导出，代价是接受一个不维护的依赖，输出侧不产 xls。

配置版本随执行快照落库：InputDocument、SupplierOrder、ChannelWritebackItem 各自保存使用时的模板 ID 和版本，ExecutionQuantityLink 保存输出模板版本。配置更新只影响之后的操作，历史记录继续用原版本解释。
