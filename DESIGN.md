# UniBlack Frontend Design System

## 1. Objective

UniBlack 前端优先服务三件事：核验账号、理解公开案件、处理治理工作。设计传达可信、克制和可追溯，而不是营销站点的热闹感。

成功标准：

- 访客在首页第一屏即可发起账号核验。
- 登录、角色、站点配置变化立即反映在导航和入口。
- 公开端、名单档案、管理端属于同一产品，但采用适合任务的信息密度。
- 375px 至桌面宽度可用，键盘操作完整，老旧设备不承担非必要动画成本。

## 2. Product Context

UniBlack 是可复用的社区云黑系统，不是固定品牌的单一站点。部署者可配置 `site.name`、描述、Logo 和主题色；“UniBlack”只作为配置不可用时的安全回退。

主要使用者：访客核验对象；社区成员举报和申诉；moderator 审核；admin 管理用户、名单和系统策略。

## 3. Visual Foundations

### Visual modes

- **公开端，轻量 SaaS**：浅冷灰画布、清晰搜索入口、宽松内容节奏；不用渐变 Hero 和装饰统计卡。
- **名单与对象，可信档案**：细分隔线、标识符簇、风险状态、案件时间线；关系优先于卡片数量。
- **管理端，数据控制台**：固定侧栏、紧凑工具栏、表格优先、明确状态与批量操作。

### Color tokens

- `--background: #F6F8FA`
- `--surface: #FFFFFF`
- `--foreground: #172033`
- `--muted: #667085`
- `--border: #DDE3EA`
- `--primary: <site.theme_color, fallback #2563EB>`
- `--danger: #B42318`、`--warning: #B54708`、`--success: #067647`

主题色派生 hover、浅色背景和 focus ring。若管理员配置色不满足 WCAG AA，原色只用于装饰，文字和操作使用安全回退色。

### Typography

- 正文与 UI：`Inter, "Noto Sans SC", "Microsoft YaHei", system-ui, sans-serif`。
- 数据、ID、时间：`"SFMono-Regular", Consolas, "Liberation Mono", monospace`。
- 不依赖字体 CDN；字号阶梯为 12 / 14 / 16 / 20 / 24 / 32 / 40px。
- 正文默认 16px/1.6，后台表格 14px/1.5。

### Shape and spacing

- 4px 基础间距；常用 8 / 12 / 16 / 24 / 32px。
- 表单圆角 8px，面板 10px，状态徽章全圆角。
- 阴影只用于浮层、抽屉和确需强调层级的面板。
- 公开页最大 1200px，详情最大 960px，认证表单最大 440px。

## 4. Accessibility

- 正文对比度至少 4.5:1，大字至少 3:1。
- 控件提供 hover、focus-visible、active、disabled 状态。
- 点击目标至少 44×44px；label 与控件显式关联。
- 状态不能只靠颜色表达；导航提供 `aria-current`。
- 动效受 `prefers-reduced-motion` 控制。

## 5. Voice & Tone

- 使用清晰、克制、可操作的中文，不使用营销套话。
- 操作名称保持一致，例如“保存更改”对应“更改已保存”。
- 空状态解释下一步；错误说明原因及恢复方式。
- 风险信息避免煽动性表达，明确社区记录、公开状态和申诉入口。

## 6. Implementation Practices

- Tailwind 只通过 npm/PostCSS 构建，不允许 CDN。
- CSS variables 保存运行时品牌 token，Tailwind utilities 负责布局和状态。
- `app/` 负责路由与组合，`components/shell/` 负责布局，`components/ui/` 负责基础组件，`lib/` 负责 API、认证、权限、设置和导航。
- 内部导航统一使用 Next.js `Link` 或 router；外部链接才使用原生 `<a>`。
- 核心路由由前端注册表声明；后端配置只控制品牌和功能开关，不下发任意 URL。
- 页面必须设计 loading、empty、error、unauthorized 和 normal 状态。
- 不引入全局状态库；React Context 足够承载 Auth 和 Settings。

## 7. Anti-Patterns

- 禁止渐变 Hero、悬浮统计卡、同质功能卡网格和装饰 emoji。
- 禁止所有内容使用相同白色圆角阴影卡片。
- 禁止用菜单隐藏代替后端 RBAC。
- 禁止页面自行读取 token、拼 Authorization header 或处理 401。
- 禁止设置加载期间闪现错误业务状态。
- 禁止为动态页面引入通用 CMS、远程组件或任意后端菜单。
- 禁止无目的入场动画、视差和持续运动背景。

## 8. Decision-Making

新增页面依次确定：页面场景、主导航必要性、登录态/角色/功能开关、移动端呈现与五类状态、可复用组件。只有出现第二个真实使用者时才抽象业务组件。

## 9. Workflow

1. 在 `lib/navigation.ts` 注册路由元数据。
2. 使用既有 Shell 和 token 构建页面。
3. 先写状态/权限测试，再实现最小行为。
4. 运行 lint、typecheck、build 和 Playwright。
5. 在 375px、768px、1280px 验收 reduced motion 与键盘焦点。
6. 更新 `docs/frontend-gap-analysis.md` 的状态与证据。

## Decision Trace

```json
[
  {
    "decision": "公开端轻量 SaaS、后台数据控制台、名单可信档案",
    "reason": "用户明确指定三类方向，三类任务的信息密度和信任诉求也不同",
    "alternatives": ["全站轻量 SaaS", "全站数据控制台"],
    "tradeoff": "需要维护三种布局密度，但共享同一 token 和组件状态"
  },
  {
    "decision": "前端路由注册表与后端配置的混合动态模式",
    "reason": "满足扩展和功能开关，同时避免任意 URL 带来的权限风险",
    "alternatives": ["布局硬编码", "后端完全下发菜单"],
    "tradeoff": "新增代码页面仍需部署前端版本"
  },
  {
    "decision": "保留 localStorage JWT 协议，仅统一封装 AuthProvider",
    "reason": "本轮不扩展为后端认证协议重构",
    "alternatives": ["迁移 HttpOnly Cookie", "继续页面直接读 token"],
    "tradeoff": "本阶段不消除 localStorage token 的 XSS 暴露面"
  },
  {
    "decision": "不引入 Zustand 或 TanStack Query",
    "reason": "共享状态仅有登录态和低频站点设置，Context 与统一 API client 足够",
    "alternatives": ["Zustand", "TanStack Query"],
    "tradeoff": "复杂缓存能力延后到确有需求时"
  },
  {
    "decision": "主题色以 CSS 变量运行时注入",
    "reason": "site.theme_color 可由控制台热更新，无法只在构建期固化",
    "alternatives": ["固定蓝色", "每主题生成 bundle"],
    "tradeoff": "需要对比度校验和安全回退"
  },
  {
    "decision": "动效限制为 120–200ms 状态过渡",
    "reason": "满足现代丝滑且兼顾老旧设备的要求",
    "alternatives": ["完全无动效", "页面入场和视差"],
    "tradeoff": "视觉表现更克制"
  }
]
```

## Anti-Slop Self-Check

Clean：规划排除了渐变 Hero、同质卡片网格、装饰 emoji、悬浮统计卡和过量主按钮。公开首页的特征来自核验主任务，名单页来自档案关系。
