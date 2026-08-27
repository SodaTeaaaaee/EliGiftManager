# 写用例在 app 层包事务，Store 即 UnitOfWork

事务边界在用例方法体上：`domain.Store` 本身就是 UnitOfWork，`WithTx(ctx, fn)` 把一个事务绑定的 Store 交给回调，回调内的每一步读写都显式走这个 tx-Store，成功一起提交，失败一起回滚。用例方法体保持单层薄壳——开事务、调私有步骤函数、带出结果——不在 controller 层包事务：controller 只做参数转发和空值兜底，看不到也不该看到 Store；事务里哪些步骤必须原子是业务规则，属于 app 层。

嵌套靠两条机制防住。其一，recompute 等共享步骤接收显式 store 参数并由调用方传入 tx-Store，公开入口 `RecomputeEntitlements` 只是自带 WithTx 的便捷包装，用例内部从不调它。其二，单连接池（`SetMaxOpenConns(1)`）是天然护栏：在 WithTx 回调里误用外层 Store 会死等那条被事务占用的连接，测试基建同样单连接，hang 会在测试里当场暴露而不是溜进生产。`withStore` 把 Workspace 浅拷贝到 tx-Store 上，让事务内的复合步骤仍然复用用例代码。

审计字段由 FromDomain 映射拷贝保护：infra 的 `*FromDomain` 逐字段拷贝 CreatedAt/UpdatedAt，Save 全字段更新时不会用调用方结构体里的零值时间戳冲掉库里的创建时间；调用方从零构造编辑表单的场景（如规则编辑）由用例先回读旧记录回填再更新。
