# 操作记录保存模板版本快照

每次输入解析、工厂订单导出和渠道回写都保存当时使用的模板 ID 和版本号快照（`InputDocument.TemplateID/TemplateVersion`、`SupplierOrder.TemplateID/TemplateVersion`、`ChannelWritebackItem.TemplateID/TemplateVersion`、`ExecutionQuantityLink.ConfigVersion`）。模板更新不重新解释历史记录；如果用户要用新模板重新解析或重新生成，系统创建新的结果版本并保留旧版本。

模板编辑是原地覆盖：同一行改写名称、备注、`MappingJSON`、`LayoutJSON`，每次成功更新把 `Version` 单调加一。旧版本的内容不保留，历史快照只保留「ID + 版本号」这对标识，用来区分哪些操作早于哪次修改。平台、文档类型和方向在创建后固定，不能通过更新改变。

删除模板允许，包括被历史快照引用的模板。快照保留悬空的 ID 和版本号，不级联删除任何输入文档、工厂订单或回写项。内置模板不能更新也不能删除，它们的版本随软件升级在启动时刷新。
