---
feature: product-experience-deepening
status: delivered
specs:
  - docs/compose/specs/2026-07-29-product-experience-deepening-design.md
plans:
  - docs/compose/plans/2026-07-29-product-experience-deepening.md
branch: feature/product-experience-deepening
commits: c51fa17
---

# Product Experience Deepening — Final Report

## What Was Built

UniBlack 的核心功能闭环（对象、账号、事件、证据、申诉、处罚、管理）已经完整，但信息以通用 Panel 堆叠呈现，缺乏统一、可扫描、可追溯的产品语言。本轮把三块体验按「共享语言先行 → 三段依次深化」重构，不改变任何后端写协议、认证方式或数据模型：

1. **可信档案体验**：对象详情改为档案式层级（档案抬头 → 账号簇 → 公开事件轨迹），事件详情拆为四层（事件抬头 → 事实描述 → 证据材料 → 申诉与纠错），所有状态、风险、严重程度、证据类型、申诉 outcome 统一为中文标签并保留原始枚举供审计。
2. **治理工作台**：管理端申诉队列从宽表格改为扫描优先的响应式语义列表，提供待办计数、状态筛选、刷新恢复，以及保留原始 `window.prompt` 的逐项裁决流程。
3. **发布引导体验**：`/submit` 改为三段任务流（对象账号 → 事件事实 → 证据与确认），每段有完成状态、分区错误提示与焦点引导；提交失败完整保留用户输入；JSON/FormData 发布协议逐字节不变。

后端仅有一处最小只读补充：`EventRepository.GetByID` 预加载既有 `evidence` 关联（按 `created_at ASC`），不涉及迁移、新表或新写端点。

## Architecture

### 共享展示语言模块

`frontend/lib/trustworthy-profile.ts` 是三个体验共用的纯函数模块（109 行，无 React 依赖）：

- `subjectStatusPresentation` / `eventStatusPresentation` / `appealStatusPresentation` — 返回 `{ label, auditLabel, tone }`，未知枚举确定性回退到 `neutral` 并保留原始值。
- `appealOutcomeLabel(outcome, audit?)` — `upheld/corrected/withdrawn/malicious_submission` 的中文映射。
- `riskPresentation` / `severityPresentation` — 数值钳制（0–5 / 1–5）+ 中文等级标签，满足「状态不能只靠颜色」。
- `evidenceTypeLabel` / `evidenceAction` / `accountDisplayValue` — 证据类型映射、仅对真实 URL 生成外部链接、账号值取 username 优先。

### 后端只读关联

`backend/internal/models/models.go` 的 `Event` 增加 `Evidence []Evidence`（`foreignKey:EventID`）；`backend/internal/repository/event.go` 的 `GetByID` 使用 `Preload("Evidence", ...).Order("created_at ASC")`。`ListBySubject` 不做预加载——对象时间线只需要事件摘要。集成测试位于 `backend/internal/integration/governance_test.go`。

### 页面边界

- `frontend/app/subjects/[id]/page.tsx` — 档案抬头、账号簇（Account 优先、Identifier 标记「兼容账号记录」、首项标「主标识」）、左侧关系线事件时间线。
- `frontend/app/events/[id]/page.tsx` — 四层语义结构；申诉区用显式 `Feedback { tone, message }` 取代 `message.includes('失败')` 推断。
- `frontend/app/admin/appeals/page.tsx` — 待处理计数（非 pending 筛选时用 `page_size=1` 探针）、AbortController 防陈旧响应、语义 `<ul>` 列表、单项 prompt 裁决。
- `frontend/app/submit/page.tsx` + `frontend/app/submit/publishing.ts` — 三段分区、`validatePublishDraft` 单一校验源、`buildPublishPayload`/`listPublishFiles` 保持协议。

### 设计决策

- **选择分段深化而非三线并行**：共享状态语言和组件边界需要先稳定，并行只用于只读调研与独立评审。
- **`publishing.ts` 保持页面级而非全局**：只有发布页一个消费者；出现第二个消费者前不提取全局模块。
- **客户端校验规则从后端常量镜像**（200 KiB 文本、20 MiB 文件、扩展名白名单、`AccountDedupKey`、`ResolveDisplayName`），`buildPublishPayload` 不预置 `display_name`，服务器保持默认名解析的唯一真源。
- **不做批量裁决**：危险动作继续逐项确认，符合 S3/S6 的明确边界。

## Usage

无新配置、无新环境变量、无迁移命令。行为变化全部在前端交互层：

- 访客访问 `/subjects/:id` 与 `/events/:id` 可看到档案式层级与中文状态/风险/严重程度标签；证据仅在有真实 URL 时渲染外链（`target="_blank" rel="noreferrer"`），无 URL 的文件只展示元数据。
- 审核员进入 `/admin/appeals` 可看到「事件申诉工作台」：`待处理 N 项` 计数、状态筛选（`待处理（pending）`/`已通过（approved）`/`已驳回（rejected）`/`全部状态`）、`刷新队列`，每条申诉显示事件 ID 链接、理由全文、已有裁决与四个单项动作（维持/更正/撤回/恶意提交）。
- 登录用户进入 `/submit` 依次完成三段；每段右侧 Badge 显示 `已完成`/`待补充`；提交前客户端校验把焦点引向首个问题分区；API 失败保留全部输入（含 `File[]`）并把错误映射到对应分区。

## Verification

- **前端单元测试**：`npm run test:run` → **75/75 通过**（8 个文件），含 `trustworthy-profile.test.ts`（8）、`publishing.test.ts`（42）、`demo-captcha.test.tsx`（11）、`appeal-presentations.test.ts`（4）。
- **Playwright E2E**：跨 desktop（1280）/tablet（768）/mobile（375）三档视口，关键路径 82 个用例通过；每个场景断言 `expectNoHorizontalOverflow`。
- **前端门禁**：`npm run lint`（0 warnings）、`npm run typecheck`、`npm run build` 全部通过。
- **后端**：`go vet`、`go build ./cmd/server`、`go test ./internal/integration`（6/6）通过。
- **独立只读审查**：对照规格逐条核查 7 项边界（无伪造证据元数据、无 token/fetch 越界、Case 兼容保留、Evidence 仅详情预加载、prompt 单裁决保留、JSON/multipart 形状不变、S1–S12 均有实现或测试证据），结论 **pass**。
- **验证限制（2026-10-02 标记校正）**：历史 E2E 记录包含失败，不能据串行通过就认定其与实现无关。`page.route()` 跨 worker 竞态是当时的归因假说，不是已复现结论；`ECONNREFUSED` 说明该环境后端未就绪。完整并行矩阵、trace 和夹具隔离仍需补验，不豁免发布门禁。

## Journey Log

- [待复核归因] 当时怀疑 fully-parallel 下存在 mock/夹具竞态；`--workers=1` 只是诊断变量，不证明 `page.route()` 跨 worker 污染。应保留并行 trace、共享后端与 mock 生命周期证据后再定因。
- [lesson] Next.js 会注入 `__next-route-announcer__` 的 `role="alert"`，Playwright 中 `getByRole('alert')` 需要按文本过滤，避免严格模式多匹配。
- [pivot] 事件详情页的证据展示发现公开 Event 响应本不含 evidence，先补最小只读 GORM 预加载（无迁移）再实现前端，避免长期空态。
- [lesson] Vitest 配置未启用 RTL 自动 cleanup，组件测试需局部 `afterEach(cleanup)`，否则残留挂载导致 `getByRole` 歧义。
- [lesson] 后端错误为英文关键词、UI 文案为中文，`mapPublishErrorToSection` 需先匹配 evidence/file/verification 等证据类关键词再匹配 facts，避免 `"invalid evidence event index"` 被误分类。

## Source Materials

| File | Role | Notes |
|------|------|-------|
| `docs/compose/specs/2026-07-29-product-experience-deepening-design.md` | 设计规格 | 三段体验与验收标准，S1–S12 |
| `docs/compose/plans/2026-07-29-product-experience-deepening.md` | 实施计划 | 6 个任务的 TDD 步骤与验证命令 |
| `docs/frontend-gap-analysis.md` | 进度台账 | 已按实际验证结果更新状态行 |
