# 前端重设计计划（Frontend Redesign Plan）

> 状态：**已定稿待实施**（2026-07-06）
> 依据：全量代码摸底（10 个子系统地形图）+ 信息架构/操作员模拟双重交叉审 + 产品负责人 12 项决策确认。
> 范围：前端整体重写（骨架 + 皮肤），配套四类后端接口扩展。后端 4 层架构（domain/app/infra/controller）不变。

---

## 0. 设计宪章（产品负责人已确认的决策）

以下决策为本计划的前提，实施中不得擅自偏离；变更需回到产品负责人重新确认。

| # | 决策项 | 结论 |
|---|--------|------|
| 1 | 目标用户 | **将来产品化给其他创作者使用**。界面必须自解释、低门槛、术语友好；人性化优先于极限效率 |
| 2 | 核心痛点排序 | ① 导航与页面结构混乱 ② 界面密度/视觉/术语问题 |
| 3 | 重设计范围 | **骨架 + 皮肤全部重做**。页面可合并、拆分、消失 |
| 4 | 后端配合 | 允许新增接口；本次纳入四类：任务中心聚合、生命周期补全、批量操作、服务端过滤/分页 |
| 5 | 界面语言 | **中英双语一步到位**（完整 i18n 覆盖，零硬编码文案） |
| 6 | 视觉基调 | 介于现代生产力工具风与温暖创作者风之间；**必须预留换肤空间**——后续将有插画师定制皮肤（届时带二次元取向，现阶段不做） |
| 7 | 工作流引导模式 | **任务中心式**：首页聚合"今天该处理什么"，待办直达操作面；波次内自由导航 + 清晰进度总览 |
| 8 | 最痛页面 | 波次工作区（分配/审查）、Profile/模板配置 |
| 9 | 组件库策略 | **混合**：自建 design token 设计系统 + 自写壳/导航/卡片/状态组件；表格等重型数据组件保留 Naive UI（用 token 桥接其主题） |
| 10 | 接入配置定位 | **引导式接入向导 + 可视化字段映射**（平台预设 → 业务面 → 样本 CSV → 拖拽映射带实时预览）；专家模式（直接编辑 JSON）保留为高级入口 |
| 11 | 后端配合范围 | 四类全部纳入（见第 5 节） |
| 12 | 落地节奏 | **并行重写、整体切换**：新前端在独立目录从零构建，完成后一次性切换；旧前端期间保持可用 |

---

## 1. 现状诊断（重设计要解决的问题清单）

### 1.1 结构性问题（信息架构层）

1. **任务中心有名无实**：Dashboard 自称"行动中心"但只读；5 个行动磁贴中 4 个跳转到无过滤的 `/waves`（`DashboardPage.vue:118,127,137,145`），波次列表也没有对应的筛选维度——分诊入口全部断头。"漂移审查"计数是自认的启发式替代（`DashboardPage.vue:63-67`，DTO 无漂移信号）。侧边栏无任何工作量徽标。
2. **四维状态组合筛选无处可用**：文档钦定履约行有 4 个独立状态维度（allocation/address/supplier/channelSync）且运营需按组合筛选（如"地址就绪但未提交工厂"），但唯一展示全维度的 AdjustmentReviewPage 只有 5 张互斥筛选卡 + 关键词搜索，无 addressState 列、无 AND 组合、无导出。
3. **"这个人到底发了没"无界面可答**：CustomerDetailPage 的波次/需求历史是两块永久占位符（`CustomerDetailPage.vue:249-288`，"Cross-reference query is not yet exposed in the bridge"）。跨波次的按人履约真相——产品的核心承诺——需要在多个波次的调整页里人脑 join。
4. **修复类操作把人踢出波次工作区**：就绪页修地址 → `/customers/:id`（只读，还得回列表页找行开抽屉）；改路由 → `/demand-inbox?docId=N`。每个阻塞项一次完整上下文切换，无返回路径。地址采集是持续数周的最长阶段，每条地址约 10-12 次点击。
5. **锁不锁、导不导自相矛盾**：波次侧边栏对 idle 步骤画锁图标但点击照样跳转（`WaveWorkspaceSidebar.vue:59-64` vs `:364-370`）；唯一硬门（就绪页 Proceed 禁用）无任何解释；后端 `ValidateStepAccess`（`controller_wave.go:523-542`）实现了 5 个步骤守卫但前端从未调用——三套半成品的门禁并存。
6. **"导出"步骤导不出文件**：`exportSupplierOrder` 只建 DB 草稿行（`use_cases.go:485-568`）；真正的工厂文件由**最后一步**渠道同步的 executor 写到 `data/exports/`（`document_export_executor.go:103`），且输出路径从不显示在 UI。运营找"给工厂的文件"必然迷路。
7. **术语系统性漂移**：生命周期以原始枚举串直出（`awaiting_manual_closure`）；收件箱/拉单用无图例的单字母标签 `r/w/d/x`；"Profile"一词同时指集成配置和客户档案；约一半操作文案硬编码英文、错误横幅硬编码中文——locale 切换形同虚设。i18n 为自制实现，无插值/复数，缺 key 静默返回 key 路径。

### 1.2 工作流断点（操作员场景走查结论）

以"场景 A：月度会员波"与"场景 D：混合波"逐步走查现有界面的结论：

| 断点 | 现状 |
|------|------|
| **需求导入（阶段 1）走不通** | 收件箱"Import Document"是单行手填占位表单（`DemandInboxPage.vue:467,743-745`）；`PickCSVFile`/`ImportDemandFromCSV` 后端管线存在但无任何活页面调用。导入 300 人名单没有可行路径 |
| 发货回传 CSV 要求内部 DB ID | 导入向导要求 `supplierOrderLineId`/`fulfillmentLineId` 列——工厂回传文件不可能含我方 DB ID；导入错误被收集但**从不渲染**，部分成功时向导重置、错误丢失（`WaveShipmentStep.vue:229-238`） |
| 混合波第二张供应商订单发不了货 | 手工发货表单只绑 `order[0]`（`WaveShipmentStep.vue:337`），其余订单的行在选择器中根本不出现，无警告 |
| 部分发货无已发/剩余 | 行选择器只显示 submittedQuantity；跨多次发货的超发无任何防护；第二批发货需在体外记账 |
| 波次盲命名、不可改名、不可显式关闭 | 一键创建 `Wave <epoch-ms>`（`WavesPage.vue:88-96`）；无 UpdateWave/CloseWave；"关闭"仅是投影计算，运营永远没有完成时刻 |
| 补发（reissue）缺失 | 文档决策 #15 明确允许 reissue，UI 只有 add/reduce/remove/replace（`AdjustmentReviewPage.vue:39-44`）；补发只能伪造上游需求单，污染需求真相层 |
| 工厂回执无处记录 | SupplierOrder 的 status/externalOrderNo/acceptedQuantity 全部 write-never——"已提交工厂/工厂已接单"只能记在聊天记录里 |
| 撤销不可发现、边界不可见 | 仅 Ctrl+Z/Y，无可见按钮；每次撤销强制重挂载整个步骤路由丢失 tab/选中/滚动（`WaveWorkspaceLayout.vue:184`）；导出之后的命令 patch 为空、Ctrl+Z 静默无效，边界从未告知用户 |
| 数据常态性过期 | Dashboard 仅 onMounted 加载、无刷新；波次列表返回不重载；同步任务无轮询 |

### 1.3 现存缺陷（已亲手验证）

1. **CustomerManagementPage 缺 import**：`getCustomerProfile` 在 6 处调用（`:800,865,880,988,1003,1038`）但 import 块（`:513-527`）未导入——客户抽屉打开即抛错，vue-tsc 应无法通过。
2. **ProfileDetailPage 枚举漂移**：`referenceStrategy` 提供 `internal_only`/`external_order_required`/`external_eligibility_context`（`ProfileDetailPage.vue:116-120`），后端白名单为 `member_level`/`order_level`/`order_line_level`（`profile_usecase.go:72-76`）——**没有一个选项能保存成功**。`recipientInputMode`、`entitlementAuthorityMode`、`documentType` 同样漂移。讽刺的是死代码 ProfileListPanel 里的选项集才是对的。
3. 其他确认项：规则删除确认框文案是 "Edit Rule?"（`MembershipAllocationPage.vue:210`）；reconcile 有失败仍弹成功 toast（`:314-316`）；membership-allocation 全部操作无 catch、错误静默；`routingStats` 拉取后从不渲染（`DemandMappingPage.vue:38,148-153`）。

### 1.4 死代码清单（新前端一律不迁移）

| 文件 | 行数 | 备注 |
|------|------|------|
| `pages/demand-intake/DemandIntakePage.vue` | 428 | 曾持有唯一的列表级"推送到波次"UI（该能力值得在新收件箱复活） |
| `pages/profile/ProfileListPanel.vue` | 457 | 持有**正确的**枚举选项集与更完整的表单校验（迁移其语义，不迁移代码） |
| `pages/profile/ProfileMergePage.vue` | 117 | merge 已迁至客户页 |
| `pages/template/TemplateManagementPage.vue` | 225 | 内含跳转到已不存在路由的按钮 |
| `pages/template/TemplateCsvImportPage.vue` | 129 | "CSV 导入"实为 JSON 粘贴 |
| `pages/template/ProfileTemplateBindingPage.vue` | 265 | 持有 live 页面丢掉的 isDefault 开关 |
| `pages/address/AddressManagementPage.vue` | 227 | 混淆 IntegrationProfile/CustomerProfile ID 空间 |
| `shared/lib/table/*` + `shared/ui/table/*` | ~700 | 自适应表格子系统，零消费者；**其中 CJK 感知排序（拼音/假名/谚文）质量高，值得在新 DataGrid 中复活** |
| 全局 ContextMenu 体系 | ~200 | 全局劫持右键但零注册者，用户失去原生复制/粘贴菜单 |
| 桥接层 10 个无人调用的 wrapper | — | `getWaveOverview`、`listWavesPaginated`、`validateStepAccess` 等——多数应在新前端**启用**而非删除 |

### 1.5 可复用的隐藏资产

后端已实现但前端未用/低用的能力，新前端应作为地基直接采用：

- `WaveWorkspaceSnapshotDTO`：stepStates（状态+计数）、guidance codes、suggestedNextStep/nextStepReason/blockingIssues、basisSummary——**任务中心式引导的现成引擎**。
- `ValidateStepAccess`：步骤门禁的唯一真相源。
- 波次级持久化历史树（HistoryScope/Node/Checkpoint/Pin + GC）。
- `ImportDemandFromCSV` 模板管线、`PickCSVFile`/`PickZIPFile` 原生对话框。
- 双模部分成功导入（reject_all / skip_invalid，决策 #36）。
- 文档钦定但未实现的 UX 规范：三问分流（#14）、无伪百分比的漏斗进度、Overview 六桶分组、撤销 toast+回执托盘（#25）、漂移双轴投影为单一摘要态。

---

## 2. 新信息架构

### 2.1 顶层导航（6 项 + 设置）

```
┌──────────────────────────────────────────────┐
│ 任务中心   Home        今天该处理什么（默认页）│
│ 波次       Waves       波次列表 → 波次工作区   │
│ 收件箱     Inbox       需求接入与路由分诊      │
│ 客户       Customers   档案·身份·地址·履约历史 │
│ 商品       Products    商品主档 + 波次备货     │
│ 接入       Integrations 平台接入向导与配置     │
│ ──────────                                    │
│ 设置       Settings    外观·语言·操作员·数据   │
└──────────────────────────────────────────────┘
```

原则：

- 侧边栏项带**实时工作量徽标**（由任务中心聚合接口驱动），导航本身就是状态板。
- 波次工作区内部导航（见 3.3）是第二层壳，与顶层导航视觉区分。
- 全部路由捨弃 legacy 重定向与 `:demandKind?` 参数路由——新树从零定义，无历史包袱。
- 错误边界只替换内容区，导航永远可用（修复现状 App.vue 整壳被替换的问题）。

### 2.2 术语系统

**双语术语表是一等公民资产**：新建 `shared/i18n/glossary.ts`，为每个领域枚举值提供 zh/en 显示文案 + 一句话解释（tooltip 用）。所有状态**只能**通过共享 `<StatusBadge>` / `<StatusDot>` 组件渲染，组件内部查术语表——从机制上杜绝裸枚举直出。

种子文案（待产品负责人终审后进入 glossary）：

| 枚举 | 值 → 中文（英文沿用 Title Case 化的枚举语义） |
|------|------|
| LifecycleStage | draft 草稿 · allocating 分配中 · address_blocked 等待地址 · ready_to_submit 可提交工厂 · submitted_to_supplier 已提交工厂 · partially_shipped 部分发货 · shipped 已发货 · syncing_back 回填中 · awaiting_manual_closure 待人工收尾 · closed 已关闭 |
| RoutingDisposition | pending_intake 待分诊 · accepted 已接收 · deferred 暂缓 · excluded_manual 手动排除 · excluded_duplicate 重复排除 · excluded_revoked 已撤销 |
| RecipientInputState | not_required 无需采集 · waiting_for_input 等待填写 · partially_collected 部分填写 · ready 已就绪 · waived 已豁免 · expired 已过期 |
| AddressState | missing 缺地址 · ready 地址就绪 · invalid 地址无效 |
| SupplierState | not_submitted 未提交 · submitted 已提交工厂 · accepted 工厂已接单 · producing 生产中 · partially_shipped 部分发货 · shipped 已发货 · canceled 已取消 |
| ChannelSyncState | not_required 无需回填 · unsupported 平台不支持 · pending 待回填 · synced 已回填 · manual_confirmed 人工确认 · skipped 已跳过 · failed 回填失败 |
| 漂移摘要（双轴投影单值） | in_sync 无漂移 · drifted-none 有漂移 · drifted-recommended 建议复查 · drifted-required 必须复查 |

其他术语裁定：

- "Profile" 歧义终结：集成配置一律称**接入配置（Integration）**，客户档案一律称**客户（Customer）**。UI 层不再出现裸词 "Profile"。
- `r/w/d/x` 单字母标签废除，改为 StatusBadge 分段条 + 悬浮图例。
- "渠道同步"在仅有本地文件 executor 期间如实命名为**"生成回填文件"**；真实 API 连接器上线后再改称"平台回填"。
- i18n 键与 V1 残留词（会员/发货记录/模板）全部清除，与 `02-ubiquitous-language.md` 对齐。

### 2.3 旧 → 新页面映射

| 旧（28 页） | 新 | 处置 |
|---|---|---|
| DashboardPage | **任务中心** | 重写：只读计数板 → 可行动待办流 |
| WavesPage | **波次列表** | 重写：+搜索/阶段筛选/命名创建/改名/关闭 |
| WaveWorkspaceLayout + 9 步骤页 | **波次工作区**（总览 + 履约网格 + 3 组步骤） | 重写，结构重排（见 3.3） |
| MembershipAllocationPage | 工作区·分配（规则） | 重写：+全量 dry-run 预览、失败明细、约束化选择器 |
| DemandMappingPage | 工作区·分配（订单映射） | 重写：与规则分配并列于同一"分配"区 |
| AdjustmentReviewPage | **履约明细网格**（工作区中枢） | 重写升级：四维组合筛选 + 批量调整 + 补发 |
| WaveReadinessStep | 工作区·就绪（网格的保存视图 + 行内修复） | 重写 |
| WaveExportStep / WaveShipmentStep / WaveChannelSyncStep | 工作区·执行（工厂订单 → 发货回传 → 回填收尾） | 重写（见 3.3.4） |
| WaveHistoryPage + WaveHistoryPanel + UndoRedoTray | **单一历史抽屉** + 可见撤销按钮 | 合并（文档 #25 钦定形态） |
| DemandInboxPage | **收件箱** | 重写：真实文件导入 + 推送到波次 |
| ProfileManagementPage / ProfileDetailPage | **接入**（向导 + 配置卡） | 重写（见 3.4） |
| CustomerManagementPage + CustomerDetailPage | **客户**（列表 + 统一详情） | 合并重写：详情页成为唯一编辑面 + 履约历史 |
| ProductManagementPage | **商品** | 重写：+搜索/筛选/批量备货到波次 |
| SettingsPage | **设置** | 扩展：+操作员、数据目录、皮肤 |
| 7 个死页面 | — | 不迁移（语义资产按 1.4 注记吸收） |

---

## 3. 核心界面设计

### 3.1 任务中心（默认页）

**回答一个问题：「现在该做什么」。**

- **待办流**（主体）：由新聚合接口（5.1）驱动的行动卡片，按紧急度排序。每张卡 = 一个波次的一个阻塞桶：`「三月会员波」12 条缺地址`、`「四月零售波」回填失败 3 条`、`收件箱 5 单待分诊`。**点击直达对应波次工作区的预过滤网格视图**——深链带完整筛选参数，杜绝断头。
- **进行中的波次**（次栏）：每波一张卡：名称、生命周期阶段徽标、漏斗微缩图（见 3.3.1）、最近活动时间。点击进入工作区。
- 空状态即引导：无波次时展示"创建第一个波次"三步引导（接入 → 导入 → 建波）——产品化的新手着陆点。
- 数据每次进入页面刷新 + 手动刷新按钮 + 窗口重获焦点自动刷新。

### 3.2 波次列表

- 创建波次 = 对话框：名称（必填，预填建议 `2026-07 会员波` 式样）、波次类型、备注。杜绝 `Wave <epoch-ms>`。
- 行内改名/备注/关闭（新后端接口）；按生命周期阶段 + 关键词过滤；行点击进入工作区。
- 显式**关闭波次**动作：全部收尾时一键关闭；有残留项时"强制关闭"需填写说明（审计留痕），给运营一个明确的完成时刻。

### 3.3 波次工作区（重设计核心）

结构：**总览 + 履约明细网格（中枢）+ 三组流程区**，自由导航 + 咨询式门禁。

```
┌─ 波次头部：名称(可改) · 阶段徽标 · 漂移摘要(单值) · 撤销/重做按钮 · 历史抽屉 ─┐
│                                                                              │
│  总览 Overview        ← 漏斗 + 建议下一步 + 六桶 + 三问分流                    │
│  ── 准备 ──                                                                  │
│  需求接入 Intake       ← 拉单/退单，含批量                                    │
│  分配 Allocation      ← 规则（会员）｜订单映射（零售）二合一页签               │
│  ── 审查 ──                                                                  │
│  履约明细 Lines        ← ★ 四维网格：筛选/批量/调整/补发/行内修地址            │
│  就绪检查 Readiness    ← 网格的"阻塞项"保存视图 + 前进硬门                     │
│  ── 执行 ──                                                                  │
│  工厂订单 Factory      ← 生成文件·标记提交·记录接单                            │
│  发货回传 Shipments    ← 回传导入·部分发货·多订单                              │
│  回填收尾 Closure      ← 生成回填文件·人工收尾·关闭波次                        │
└──────────────────────────────────────────────────────────────────────────────┘
```

#### 3.3.1 总览

严格按文档实现（`02-status-and-progress-model.md`）：

- **漏斗条而非百分比**：总行数 → 地址就绪 → 已提交工厂 → 已获物流 → 已回填 → 人工收尾 ／ 回填失败。每段可点击 → 网格预过滤视图。
- **建议下一步**为唯一主 CTA（后端 `suggestedNextStep` 驱动），配 reason 的人话句子（guidance code → glossary 完整句，修复现状拿栏目名当理由的问题）。
- **六桶分组卡**（可推进 / 需回分配 / 需回映射 / 建议调整 / 等待输入·地址·资格 / 未接收），每桶直达网格。
- 漂移双轴投影为**单一摘要徽标**（4 档警示强度），点击展开 BasisDrift 详情抽屉，抽屉内每个信号**可下钻到受影响的行**。
- 三问分流嵌入总览（"我要改的是上游事实 / 默认逻辑 / 本波例外？"），三个入口分别带上下文跳转。

#### 3.3.2 履约明细网格（本次重设计最高价值单体）

一张网格回答运营的全部日常问题：

- **列**：参与者 · 商品 · 数量 · 来源（会员权益/零售订单/人工补发，行级溯源）· 分配态 · 地址态 · 工厂态 · 回填态 · 复查要求 · 物流单号。
- **组合筛选栏**：每个状态维度一个多选下拉，维度间 AND——"地址就绪 ∧ 工厂未提交"两次点击可达。常用组合提供**预置保存视图**（阻塞项/可提交/回填失败/已调整）。筛选状态写入 URL query，深链可分享、任务中心可直达。
- **批量操作**：多选 → 批量调整（加赠/减赠/替换/移除/**补发**）、批量绑定默认地址、批量导出 CSV（运营对账逃生门）。
- **行详情侧板**（非弹窗）：该行全维度状态时间线、来源链（哪条需求/哪条规则/哪次调整）、发货记录、调整历史；调整表单在侧板内完成——reasonCode 改为受控枚举 + 备注，operatorId 改为操作员选择器。
- **行内修地址**：地址态异常的行直接在侧板内编辑/选择地址（调用地址簿），不再弹出工作区。
- 服务端筛选 + 分页（5.4），大波次不降级。

#### 3.3.3 门禁与撤销

- **自由导航 + 咨询式门禁**（文档 #11 钦定）：侧边栏用状态点而非假锁；进入"尚不该做"的步骤时顶部显示非阻塞提示条（由 `ValidateStepAccess` 驱动——它成为门禁唯一真相源），说明缺什么、一键跳去补。唯一硬门保留在就绪页 → 工厂订单，禁用按钮必须配原因明细。
- **撤销体系**（文档 #25）：头部可见撤销/重做按钮 + Ctrl+Z/Y；成功后 toast + 回执托盘；**数据就地刷新，不再重挂载路由**（保 tab/选中/滚动）；**撤销边界可视化**——工厂订单生成起的操作标注"不可撤销，可用补偿操作"，界面如实告知。历史图收敛为单一抽屉（时间线 + 检查点/固定），全页历史路由取消；GC 统一带确认。

#### 3.3.4 执行链（工厂 → 发货 → 收尾）

- **工厂订单**：生成订单 + **当场生成可下载的工厂文件**（文件生成从渠道同步 executor 前移；显示输出路径 + "打开所在文件夹"）。文件内**嵌入行 ID/批次号**用于回传对账。新增"标记已提交"（填工厂单号/时间）与"记录接单"（接受数量）动作，打通 draft → submitted → accepted（5.2）。多订单并列卡片，各自独立操作。
- **发货回传**：CSV 导入匹配键改为**回传对账键**（嵌入的行 ID 或 批次号+行号 组合），不再要求工厂提供 DB ID；导入结果页**逐行渲染错误**（skip_invalid 模式必须列出被跳过的行）；行选择器显示**已发/剩余**数量并阻止超发；手工发货表单支持选择任意供应商订单。
- **回填收尾**：更名"生成回填文件"（如实）；任务表显示输出文件路径；运行中任务自动轮询 + 手动刷新；人工收尾决策表单保留（reason + 操作员）；末尾是**关闭波次**卡片——运营的完成时刻。

### 3.4 接入（原 Profile/模板，最痛区之二）

**双层结构：向导创建 + 配置卡管理。**

- **接入向导**（新数据源的唯一推荐路径）：
  1. 选来源渠道——**平台预设卡片**（Patreon / Bilibili / Gumroad / YouTube / 自定义），预设自带典型字段映射与能力开关；
  2. 选业务面（会员赠礼 / 商店订单 / …），向导用人话解释两类需求的区别;
  3. 上传**样本 CSV** → 解析表头 → **可视化字段映射**：左侧我方语义字段（带解释），右侧下拉/拖拽绑定源列，**实时预览前 5 行映射结果**，错误即时标红；
  4. 确认能力项（是否部分发货/是否回填/收件信息采集方式）——全部人话开关，枚举值不露出；
  5. 生成 IntegrationProfile + DocumentTemplate + 绑定，一步完成。
- **配置卡管理**：已有接入按渠道分组卡片（多组可同开，废除手风琴互斥）；卡片进入详情可改能力项、重跑映射向导、管理模板绑定（**含解绑/换默认**——修复现状 append-only）。
- **专家模式**：详情页折叠区保留原始 JSON（mappingRules/extraData）直编 + 校验；connectorKey 改为对 `listConnectorCapabilities()` 的选择器。
- 枚举选项**一律来自代码生成的契约**（5.5），从根上终结 ProfileDetailPage 式的选项漂移。

### 3.5 收件箱

- **真实文件导入**：主按钮"导入需求文件"→ 选接入配置 → `PickCSVFile`/`PickZIPFile` 原生对话框 → 后端模板管线解析 → **导入预览**（识别行数/异常行）→ 选择 reject_all / skip_invalid → 逐行结果报告。手工录入保留为次按钮。
- **推送到波次**：分诊完成的单据可直接"发送到波次"（批量，复活死页面 DemandIntakePage 的正确语义），与波次内拉取双向可达。
- 主从布局保留，行编辑改为宽侧板（修复 3/8 窄列塞 7 列表格的现状）；离开有未保存更改时守卫提示。

### 3.6 客户

- **列表 + 统一详情**：详情页成为唯一编辑面（吸收现状抽屉里的全部 CRUD），列表行点击直达详情；侧板快速预览保留只读。
- **跨波次履约历史**（新查询 5.4）：详情页核心区——该客户在所有波次的履约行时间线：波次 · 商品 · 数量 · 四维状态 · 物流单号。**"这个人到底发了没"三秒可答。**
- **合并安全化**：合并前**预览对话框**（双方身份/地址全文对照、冲突高亮），执行后结果回执；按文档 #37 合并应可撤销（后端补齐）。
- 地址表单区域级联选择 + 常用校验；平台筛选改为来自客户身份实际平台。

### 3.7 商品 / 设置

- **商品**：搜索/筛选/归档视图；**批量备货到波次**（多选 → 选波次 → 显示该波已有快照、去重提示）；同时在波次工作区·分配页提供反向入口"从主档挑选商品"。
- **设置**：外观（主题/皮肤/密度）、语言、**操作员名单**（轻量本地名单，供调整/收尾表单选择，替代自由文本 operatorId）、数据目录展示与打开、自动合并开关（配"这会做什么"说明文案）。

---

## 4. 设计系统与皮肤架构

### 4.1 Design Tokens（皮肤的地基）

三层 CSS 自定义属性：

```
primitives（原始值） →  semantic（语义角色）        →  component（组件级）
--color-blue-500        --color-bg-surface             --card-bg
--radius-3              --color-text-primary           --statusbadge-radius
--space-4               --color-accent / --color-danger --datagrid-row-height
                        --status-{ready|blocked|…}-fg/bg
```

- 状态色是语义层一等成员：每个状态维度的每个值都有 token（StatusBadge 只消费 token）。
- 排版/间距/圆角/阴影/动效时长全部 token 化；提供**密度模式**（舒适/紧凑）。
- 亮/暗主题 = 两套 semantic 层映射；`data-theme` 属性驱动，Naive UI 经由 token → `themeOverrides` 生成器桥接（单文件适配器 `shared/theme/naive-bridge.ts`）。

### 4.2 皮肤包契约（面向未来插画师定制）

皮肤 = 一个静态资源包，**不改代码**即可挂载：

```
skins/<name>/
  tokens.css        # 覆盖 primitives/semantic 变量（必需）
  assets/           # 可选装饰资产
    empty-*.svg     # 空状态插画（按场景命名槽位）
    hero.png        # 任务中心页头装饰区
    mascot.png      # 角落吉祥物槽位
  manifest.json     # 名称、作者、亮暗支持声明
```

- 组件层预留**装饰槽位**：EmptyState 的插画 slot、任务中心页头背景区、侧边栏底部角落——默认皮肤留白或用中性几何图形，未来二次元皮肤在同一槽位替换。
- 设置页皮肤选择器读取 skins 目录清单；默认皮肤即"介于生产力工具与温暖创作者之间"的基调：中性偏暖底色 + 单一强调色 + 充足留白 + 柔和圆角，插画仅在空状态出现。

### 4.3 组件策略（混合）

自建（`shared/ui/`，全部 token 驱动）：

| 组件 | 说明 |
|------|------|
| AppShell / SideNav / WorkspaceNav | 双层壳，徽标、折叠、错误边界仅覆盖内容区 |
| PageHeader / SectionCard / StatCard | 页面骨架件 |
| **StatusBadge / StatusDot / StatusLegend** | 唯一合法的状态渲染出口，查 glossary |
| **FilterBar** | 多维组合筛选 + URL 同步 + 保存视图 |
| **DataGrid** | NDataTable 的强约定封装：统一分页/加载/空态/行点击/多选/列配置；集成复活的 CJK 感知排序 |
| **FunnelBar** | 漏斗进度条（可点击分段） |
| GuidanceCard / CalloutBar | 建议下一步、咨询式门禁提示 |
| WizardFrame / **FieldMappingEditor** | 接入向导框架与映射编辑器（含实时预览） |
| EmptyState | 带插画槽位与行动按钮 |
| FeedbackSystem | 统一 toast/回执托盘/错误横幅——全应用唯一反馈路径 |
| DetailDrawer / SidePanel | 行详情侧板规范 |

保留 Naive UI：NDataTable（DataGrid 内核）、表单件（Input/Select/DatePicker/Cascader）、NModal/NDrawer 内核、虚拟滚动。**页面代码禁止直接引 Naive 布局/反馈类组件**——必须经过自建层（ESLint 规则强制）。

### 4.4 i18n 体系

- 采用 **vue-i18n**（替换自制实现）：ICU 插值/复数、缺 key 开发期告警。
- zh-CN 与 en-US 同步完整；**硬编码文案零容忍**（lint 规则扫描模板字面量）。
- glossary（2.2）与 messages 分层：glossary 管领域词，messages 管界面语句。
- 桌面原生对话框标题等 Go 侧文案同步走后端 locale 参数（小改）。

### 4.5 桌面体验细节

- 恢复原生右键菜单（移除全局劫持）；表格行的自定义右键菜单按需局部注册。
- 全局快捷键：Ctrl+Z/Y（波次内）、Ctrl+F 聚焦当前网格筛选、F5/Ctrl+R 刷新当前数据。
- 路由懒加载配顶部进度条；桥接层不可用时显示**明确的"后端未连接"全局横幅**（终结 silent-empty）。
- 缩放持久化走后端 `SaveZoom`（完成现存 TODO），启动时恢复。

---

## 5. 后端配合清单（四类，均已获批）

> 约束沿用现有 4 层架构与 controller 模式；新增 bound method 后需 `wails generate module`。

### 5.1 任务中心聚合

| 方法 | 说明 |
|------|------|
| `GetActionCenterSummary()` | 一次调用返回：各波次阻塞桶（缺地址/等输入/映射阻塞/回填失败/待人工收尾/漂移需复查）计数 + waveId + 预过滤参数；收件箱待分诊计数；导航徽标计数。**漂移信号来自真实 basis 数据**，替换现状启发式 |

### 5.2 生命周期补全

| 方法 | 说明 |
|------|------|
| `UpdateWave(id, {name, notes, levelTags})` | 波次改名/备注（DTO 字段已存在，纯 write 路径缺失） |
| `CloseWave(id, {note, force})` | 显式关闭；force 时要求说明，审计留痕 |
| `UnassignDemandFromWave(waveId, docId)` | 需求退回收件箱（分配开始前） |
| `MarkSupplierOrderSubmitted(orderId, {externalOrderNo, submittedAt})` | draft → submitted |
| `RecordSupplierOrderAcceptance(orderId, lines[])` | 记录工厂接单数量，submitted → accepted |
| `GenerateSupplierOrderFile(orderId)` → path | 文件生成前移到工厂步骤；文件嵌入行 ID/批次号供回传对账 |
| `UpdateShipment / VoidShipment` | 修正与作废（补偿路径，配合撤销边界的如实呈现） |
| 领域枚举扩展：`AdjustmentKind` 增加 `reissue` | 文档 #15 钦定；同时定义 reasonCode 受控枚举表 |
| `PreviewMergeProfiles(sourceId, targetId)` | 合并预览（冲突明细）；合并按文档 #37 补撤销路径 |

### 5.3 批量操作

| 方法 | 说明 |
|------|------|
| `BatchAssignDemandToWave(waveId, docIds[])` | 逐条结果返回（部分成功语义），替换前端串行循环 |
| `BatchBindAddressToLines(entries[])` + `BindDefaultAddressesForWave(waveId)` | 批量绑定 / 一键为缺地址行绑默认地址 |
| `BatchRecordAdjustments(entries[])` | 批量调整，逐条结果 |
| `SnapshotProductsForWave` 已支持多 id | 前端补多选 UI 即可；返回值增加"已存在跳过"明细 |

### 5.4 服务端过滤 / 分页（含查询修复）

| 方法 | 说明 |
|------|------|
| `ListWaveFulfillmentRows` 重载：filter DTO（四维状态多选 + reviewRequirement + drift + keyword）+ 分页 | 履约网格的地基；顺带修 N+1 |
| `ListDemandInboxRows` 加分页 + 服务端 profile 筛选 | 修 N+1（`controller_demand.go:246-303`） |
| `GetCustomerFulfillmentHistory(customerProfileId)` | 跨波次履约行（波次/商品/状态/物流），填补 CustomerDetail 占位符 |
| `ListWavesPaginated` 补类型化 DTO 并真正实现 SortBy | 或废弃，波次量小可暂缓 |

### 5.5 契约治理

- **枚举单一真相源**：新增小型 codegen（Go `domain/enums.go` → 生成 TS union + glossary key 清单），终结手工同步漂移（CLAUDE.md 约定 #10 已被证明失效）。
- `GetDefaultTemplateForProfile` 无模板时返回明确的 null 语义（而非零值 DTO）。
- `selectorPayload`（json.RawMessage → 错误的 number[]）的 codegen 缺陷：维持手写实体类型的 workaround，在桥接层集中注释与封装。

---

## 6. 工程方案（并行重写）

### 6.1 目录与切换

> 已于 2026-07-13 执行。

```
frontend/          # 旧前端，重写期间保持可用、原则上功能冻结
frontend-next/     # 新前端（独立 deno.json / vite / 自己的 wailsjs 生成副本）
```

- 开发期：临时分支上把 `wails.json` 的 frontend 路径指向 `frontend-next` 联调；日常 `deno task dev` 独立跑（Wails 桥不可用时显示"后端未连接"横幅 + 可选 mock 层）。
- **切换日**：`frontend` → `frontend-legacy`（保留一个版本周期后删除），`frontend-next` → `frontend`；`main.go` 的 embed 路径无需变更；更新 CLAUDE.md 与 docs/PROJECT-STRUCTURE.md。
- 旧前端冻结例外（见 8.3 热修清单）：只修阻断使用的缺陷，不加功能。

### 6.2 技术栈

Vue 3 `<script setup>` + TypeScript + Vite + **vue-i18n** + Pinia + Naive UI（重型件）+ 自建 token 设计系统。目录沿用 feature-slice 精神：`app/ pages/ widgets/(工作区级复合件) shared/{ui,api,i18n,theme,model,lib}`。桥接约定不变：唯一入口 `shared/api/bridge.ts`（对应现 `shared/lib/wails/app.ts`，迁移时逐函数校对签名）。

### 6.3 质量基线

- vue-tsc 全绿是提交门槛（现状旧树 typecheck 是破的，新树从第一天守住）。
- ESLint 自定义规则：禁止硬编码用户可见文案、禁止绕过 StatusBadge 渲染状态、禁止页面直引 wailsjs 与 Naive 布局件。
- 组件展示页 `/design-lab`（dev-only 路由）：全部自建组件 + 两套主题 + 密度模式的活样本，皮肤开发的联调场。

---

## 7. 实施阶段

> 并行重写内部仍分阶段推进，每阶段有验收标准；后端接口（P-B）与前端阶段并行。

| 阶段 | 内容 | 验收标准 |
|------|------|----------|
| **P0 地基** | tokens + 主题引擎 + 皮肤包加载器；vue-i18n + glossary；壳与双层导航；FeedbackSystem/StatusBadge/DataGrid/FilterBar/EmptyState；桥接层迁移 | `/design-lab` 展示全组件双主题；壳跑通真实路由；亮暗/中英切换即时生效 |
| **P-B 后端批次** | 5.1–5.5 全部接口（与 P0 起并行，按 P2/P3/P5 依赖排序交付） | `go test ./...` 全绿；每接口有用例 |
| **P1 任务中心 + 波次列表** | 3.1、3.2 全部 | 任务中心每张卡可深链直达预过滤视图；创建/改名/关闭波次可用 |
| **P2 工作区骨架 + 总览** | 波次壳、漏斗、六桶、建议下一步、三问分流、咨询式门禁（ValidateStepAccess 接入）、撤销按钮/托盘/就地刷新 | 总览零裸枚举；撤销不丢 UI 状态；门禁提示准确 |
| **P3 履约网格 + 调整 + 就绪** | 3.3.2 网格全功能（组合筛选/保存视图/批量/补发/行内修地址/侧板） | 验收问句 D-2："地址就绪且未提交工厂"两次点击可达并可批量操作 |
| **P4 需求接入线** | 接入向导 + 可视化映射（3.4）、收件箱 + 真实 CSV 导入（3.5）、分配页重写 | 验收场景 A 第 1-2 步：从真实 Patreon 样本 CSV 到分配完成，全程无手工逐行录入 |
| **P5 执行线** | 工厂订单（文件生成/提交/接单）、发货回传（对账键导入/已发剩余/多订单）、回填收尾 + 关闭波次 | 验收场景 A 第 6-9 步走通；导入错误逐行可见；文件路径可打开 |
| **P6 主数据线** | 客户（统一详情 + 履约历史 + 合并预览）、商品（批量备货）、设置（操作员/皮肤/数据目录） | 验收问句 D-1："这个人发了没"在客户详情 3 秒可答 |
| **P7 收尾与切换** | en-US 全量审校、场景 A/D 端到端走查、性能样本（大波次分页）、示例皮肤包、切换日操作、文档更新 | 场景 A/D 全流程点击数对比旧版下降 ≥50%；切换后 `wails build` 产物正常 |

**端到端验收场景**（源自操作员模拟审视，作为回归清单永久保留）：

- **场景 A（月度会员波）**：导入 Patreon 名单 CSV → 规则分配 → 调整数人 → 两周地址追踪（期间并行开第二个波）→ 生成工厂文件 → 导入回传单 → 生成回填文件 → 关闭。
- **场景 D（混合波）**：会员赠礼 + 零售订单 + 3 条人工补发同波；必须三秒内回答"X 发了没""哪些行地址就绪但未提交工厂"。

---

## 8. 风险与开放事项

### 8.1 风险

| 风险 | 缓解 |
|------|------|
| Naive UI 主题深度受限，未来皮肤撞墙 | token 桥接单点隔离；重型件如撞墙可按件替换（DataGrid 已是封装层，内核可换） |
| 新增 bound method 成本高（每个 controller 手工重建 ~10 repos，全库重复 ~15 次） | 接口按批次集中交付；可顺带抽一个 controller 级工厂函数（限重构范围，不动架构） |
| 并行重写期间旧前端继续演化导致双份工作 | 旧树功能冻结（8.3 热修除外）为纪律 |
| Wails codegen 对 json.RawMessage 的缺陷 | 维持手写实体 + 桥接层集中封装 |
| en-US 术语质量 | glossary 单点审校；P7 前完成负责人终审 |
| 数据规模未知 | 服务端分页/筛选兜底（已纳入 5.4） |

### 8.2 留给产品负责人的开放事项

1. **真实样本文件收集**：Patreon/Bilibili/Gumroad 导出样例、工厂回传单样例——接入向导预设与回传对账键设计的输入（P4/P5 前需要）。
2. **术语表终审**：2.2 种子表的中文文案 + en-US 对应文案。
3. **日常组合筛选清单**：运营实际最常用的状态组合（决定网格预置保存视图，P3 前确认即可）。
4. 皮肤包与插画师的协作细节（P7 后另立计划）。

### 8.3 旧前端并存期热修清单（建议，均为最小 diff）

1. `CustomerManagementPage.vue:527` 补 `getCustomerProfile` import——客户抽屉当前打开即错。
2. `ProfileDetailPage.vue:104-140` 选项集替换为后端白名单值（参照死代码 `ProfileListPanel.vue:229-245` 的正确集合）——当前部分保存必被拒。
3. （可选）`MembershipAllocationPage.vue:210` 删除确认文案、`:314-316` 失败时不弹成功 toast。

---

## 附录 A：本计划引用的关键锚点

- 路由与 legacy 残留：`frontend/src/app/router/index.ts:29-115,169-197`
- 引导引擎（未被使用的后端能力）：`controller_wave.go:523-542`、`internal/app/dto/workspace.go`、`dto/wave_overview.go`
- 文件导出真实位置：`internal/app/use_cases.go:485-568`、`internal/app/document_export_executor.go:103-121`
- 状态模型canon：`docs/fulfillment-v2-refactor/04-workflows-and-state/02-status-and-progress-model.md`
- 待实现的文档钦定 UX：`docs/fulfillment-v2-refactor/06-rollout-and-governance/06-open-decisions.md`（#11、#13、#25、#26、#38、#39）
- 术语canon：`docs/fulfillment-v2-refactor/01-boundaries-and-language/02-ubiquitous-language.md`
