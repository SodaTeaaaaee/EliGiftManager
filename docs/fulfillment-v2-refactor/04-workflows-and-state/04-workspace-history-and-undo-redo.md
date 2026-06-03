# 工作区历史、树状撤销与重做

> 全局撤销/重做的行为边界、scope 规则、分支保留、交互反馈。

## 目标

支持 `Ctrl+Z`/`Ctrl+Shift+Z`，撤销后重新编辑保留旧未来分支，软件关闭后仍可用，历史图按需打开。

## 作用域

全局快捷键统一存在，但撤销/重做的是当前激活的 `HistoryScope`（`wave`/`template`/`product_library`）。不在波次页按 Ctrl+Z 不会撤掉商品页的改动。

## 默认像线性，底层是树

- `Ctrl+Z` → 回到 parent
- `Ctrl+Shift+Z` → 沿首选分支前进
- 撤销后编辑 → 新分支，旧未来保留

## 节点粒度

一个 `HistoryNode` = 一次用户意图（批量添加规则、导入、确认调整、地址绑定）。重算、状态刷新、overview 统计不各自生成 node。

## 只回滚本地

undo/redo 回滚本地工作区 head。外部对象（`SupplierOrder`/`Shipment`/`ChannelSyncJob`）保留，偏离进入 `basis_drift_status` + `review_requirement` 提示。

## Wave 优先

全应用共用 history 基础设施，优先做稳 `wave` scope，再逐步接入 `templates`/`products`。

## 快捷键不劫持文本输入

焦点在 input/textarea/contenteditable 时，Ctrl+Z 优先服务输入控件自身。

## 反馈层

1. **即时 Toast**：每次 undo/redo 后立即出现，说明做了什么
2. **短期回执托盘**：自动消失后短时间内可翻出查看最近动作

## 历史图

高级入口，按需打开（侧边抽屉/独立弹层）。默认只暴露快捷键、toast 和轻量回执。

## 持久化

历史树不能只在前端内存。head、分支、checkpoint、pin 状态持久化。重新打开后 scope 恢复到上次状态。

## 首版最小边界

wave scope 可用、树状分支不丢、持久化、全局快捷键可用、不抢文本输入原生 undo/redo、toast 与回执、basis 偏离与历史节点关联。

延后：复杂历史图可视化、全应用一次性深度接入、高级 merge/cherry-pick。
