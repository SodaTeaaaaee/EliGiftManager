# 重命名 Demand 与 FulfillmentLine 概念

领域模型停用 Demand 作为输入事实总称，并将 FulfillmentLine 的目标概念改名为来源级履约结果。代码迁移不做向后兼容层：目标命名直接表达 InputDocument/InputFact/InputFactLine、FulfillmentResult 以及 SupplierOrderLine 之间的边界。
