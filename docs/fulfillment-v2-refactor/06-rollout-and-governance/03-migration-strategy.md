# Greenfield 重建与切换策略

## 总原则

采用 greenfield 重建。重点不是让旧代码演进，而是保留必要文档与考古资料，让新版本从第一天按目标语义落地。

## 保留边界

**保留**：docs/、备份分支、仓库历史、中性工程壳子与构建配置。

**不保留**：旧业务 controller/service/model、旧状态模型、旧模板入口、旧页面流程、带旧业务倾向的代码。

工程壳子判断标准：不依赖旧业务命名、不固化旧页面流程、不要求沿用旧 controller/service/module 切分。

## 切换方式

同仓库内重建新版本，不长期并排维护两套实现。

## 数据与 Schema

直接建立目标 Schema，不为旧表补过渡字段，不为旧数据设计长期兼容层。如未来出现生产包袱，再单独设计迁移方案。

## 工作区历史起点

`HistoryScope/HistoryNode/Checkpoint/Pin` 从 V2 新路径开始产生，旧版本历史不回填，旧对象 basis 链不补造。

## 第一批重建目标

端到端纵切面：`Demand Intake → Initial Allocation → Wave Overview → SupplierOrder Export`。

不是只搭目录/只做页面/只建表，而是一条可跑通的最小主干链路（有输入、有领域处理、有用户可见输出）。
