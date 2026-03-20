# Aether 界面重设计实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 Aether 前端从深色单页面设计重构为浅色主题 + 顶部 Tab 导航（概览/设置/日志）的 Linear 风格卡片式布局。

**Architecture:** 引入 Tab 路由状态管理，拆分三个页面组件，重写 CSS 设计系统，更新所有现有组件样式。保留 hooks、types、backend 接口层不变。

**Tech Stack:** React 18 + TypeScript + Vite + Vanilla CSS

---

## 文件结构

### 新增文件
- `frontend/src/components/TabBar.tsx` — 顶部导航栏（共享）
- `frontend/src/pages/OverviewPage.tsx` — 概览页
- `frontend/src/pages/SettingsPage.tsx` — 设置页
- `frontend/src/pages/LogsPage.tsx` — 日志页

### 修改文件
- `frontend/src/App.tsx` — 引入 Tab 路由，替换单页面布局
- `frontend/src/styles.css` — 全面重写 CSS 设计系统（深色→浅色）
- `frontend/src/components/StatusCard.tsx` — 简化为状态指示器+标题
- `frontend/src/components/PrimaryActionCard.tsx` — 简化为主操作按钮区
- `frontend/src/components/BasicProxyConfigCard.tsx` — 改为单栏居中布局
- `frontend/src/components/QuickActionsCard.tsx` — 改为图标按钮列表
- `frontend/src/components/RecentLogsCard.tsx` — 改为全宽日志查看器
- `frontend/src/components/FirstRunOnboarding.tsx` — 改为浅色主题
- `frontend/src/components/OnboardingReminder.tsx` — 改为浅色主题
- `frontend/src/App.test.tsx` — 更新文本查询以匹配新 UI
- 各组件 `*.test.tsx` — 更新测试中的文本断言

### 不变文件
- `frontend/src/types.ts`
- `frontend/src/backend.ts`
- `frontend/src/hooks/useRuntimeState.ts`
- `frontend/src/hooks/useBasicProxySettings.ts`
- `frontend/src/hooks/runtimeLogBuffer.ts`
- `frontend/src/components/runtimePhaseCopy.ts`
- `frontend/src/main.tsx`

---

## Task 1: 重写 CSS 设计系统

**Files:**
- Modify: `frontend/src/styles.css`

- [ ] **Step 1: 备份现有 styles.css**

```bash
cd frontend
copy src\styles.css src\styles.css.bak
```

- [ ] **Step 2: 重写 CSS 变量和全局样式**

替换 `:root` 和 `body` 样式为浅色主题：
- 背景 `#FAFAFA`，文字 `#111`
- 定义 CSS 变量：`--color-bg`, `--color-card`, `--color-border`, `--color-primary`, `--color-danger`, `--color-warning` 等
- 更新字体为 `"Inter", "Segoe UI", system-ui, sans-serif`

- [ ] **Step 3: 编写卡片、按钮、输入框基础样式**

- `.card` — 白色背景、16px 圆角、1px 边框、微阴影
- `.btn-primary` — 绿色背景 `#10B981`、白字、12px 圆角
- `.btn-secondary` — 灰色边框、灰色文字
- `.input` — 全宽、10px 圆角、`#E5E5E5` 边框
- `.badge` — pill 形状、小号文字

- [ ] **Step 4: 编写导航栏样式**

- `.tab-bar` — 52px 高度、白色背景、底部边框
- `.tab-button` — pill 形状、激活态 `#F0F0F0` 背景
- `.status-dot` — 8px 圆形、根据状态变色

- [ ] **Step 5: 编写页面布局样式**

- `.page-overview` — 左右 60/40 分栏
- `.page-settings` — 单栏居中 max-width 560px
- `.page-logs` — 全宽

- [ ] **Step 6: 编写动画样式**

- `@keyframes pulse` — 状态指示器呼吸动画
- `@keyframes spin` — 加载旋转动画
- 按钮 hover/active 过渡效果

- [ ] **Step 7: 运行构建验证 CSS 无语法错误**

```bash
cd frontend
npm run build
```
Expected: 构建成功，无 CSS 错误

- [ ] **Step 8: 提交**

```bash
git add frontend/src/styles.css
git commit -m "style: rewrite design system to light theme"
```

---

## Task 2: 创建 TabBar 组件

**Files:**
- Create: `frontend/src/components/TabBar.tsx`

- [ ] **Step 1: 编写 TabBar 组件**

Props: `activeTab: string`, `onTabChange: (tab: string) => void`, `statusPhase: string`

渲染：品牌文字 + 三个 Tab 按钮 + 状态圆点

- [ ] **Step 2: 运行构建验证**

```bash
cd frontend
npm run build
```

- [ ] **Step 3: 提交**

```bash
git add frontend/src/components/TabBar.tsx
git commit -m "feat: add TabBar component for tab navigation"
```

---

## Task 3: 创建 OverviewPage 页面组件

**Files:**
- Create: `frontend/src/pages/OverviewPage.tsx`
- Modify: `frontend/src/components/StatusCard.tsx`
- Modify: `frontend/src/components/PrimaryActionCard.tsx`
- Modify: `frontend/src/components/QuickActionsCard.tsx`

- [ ] **Step 1: 重构 StatusCard**

简化为新布局：状态标题 + 圆形状态指示器（带脉冲动画）。移除 hero-card 相关结构，改用 `.card` 基础样式。

- [ ] **Step 2: 重构 PrimaryActionCard**

简化为主操作按钮区域，移除 action-hints 和 action-capabilities 列表。在 StatusCard 底部直接渲染按钮。

- [ ] **Step 3: 重构 QuickActionsCard**

改为图标+文字的紧凑按钮列表样式。

- [ ] **Step 4: 创建 OverviewPage**

组合 StatusCard + PrimaryActionCard（左侧）+ 代理信息卡片 + QuickActionsCard（右侧），使用 60/40 分栏布局。

- [ ] **Step 5: 运行构建验证**

```bash
cd frontend
npm run build
```

- [ ] **Step 6: 提交**

```bash
git add frontend/src/pages/OverviewPage.tsx frontend/src/components/StatusCard.tsx frontend/src/components/PrimaryActionCard.tsx frontend/src/components/QuickActionsCard.tsx
git commit -m "feat: create OverviewPage with redesigned components"
```

---

## Task 4: 创建 SettingsPage 页面组件

**Files:**
- Create: `frontend/src/pages/SettingsPage.tsx`
- Modify: `frontend/src/components/BasicProxyConfigCard.tsx`

- [ ] **Step 1: 重构 BasicProxyConfigCard**

- 移除 config-overview 顶部概览区（信息已在概览页展示）
- 改为纵向排列的表单字段（原来是 3 列 grid）
- 更新 CSS class 名使用新设计系统
- 底部添加高级配置文字链接

- [ ] **Step 2: 创建 SettingsPage**

包装 BasicProxyConfigCard，使用单栏居中布局（max-width 560px）。

- [ ] **Step 3: 运行构建验证**

```bash
cd frontend
npm run build
```

- [ ] **Step 4: 提交**

```bash
git add frontend/src/pages/SettingsPage.tsx frontend/src/components/BasicProxyConfigCard.tsx
git commit -m "feat: create SettingsPage with centered form layout"
```

---

## Task 5: 创建 LogsPage 页面组件

**Files:**
- Create: `frontend/src/pages/LogsPage.tsx`
- Modify: `frontend/src/components/RecentLogsCard.tsx`

- [ ] **Step 1: 重构 RecentLogsCard**

- 将 panel-header 改为简洁标题 + 条数 badge + 按钮
- 更新时间线和日志条目为浅色主题样式
- 保留全部自动滚动逻辑不变

- [ ] **Step 2: 创建 LogsPage**

全宽包装 RecentLogsCard，添加「打开日志目录」按钮到顶栏右侧。

- [ ] **Step 3: 运行构建验证**

```bash
cd frontend
npm run build
```

- [ ] **Step 4: 提交**

```bash
git add frontend/src/pages/LogsPage.tsx frontend/src/components/RecentLogsCard.tsx
git commit -m "feat: create LogsPage with full-width log viewer"
```

---

## Task 6: 重构 App.tsx 引入 Tab 路由

**Files:**
- Modify: `frontend/src/App.tsx`

- [ ] **Step 1: 添加 Tab 状态管理**

使用 `useState<"overview" | "settings" | "logs">("overview")` 管理当前 Tab。

- [ ] **Step 2: 替换布局为 TabBar + 页面内容**

- 渲染 TabBar → 根据 activeTab 渲染对应的 Page 组件
- 移除 dashboard-frame、dashboard-overview、summary-strip、workspace-layout 等旧布局

- [ ] **Step 3: 更新首启向导和提醒条**

在所有 Tab 之上层叠渲染 FirstRunOnboarding overlay（保持原有逻辑不变）。OnboardingReminder 显示在 TabBar 下方。

- [ ] **Step 4: 运行构建验证**

```bash
cd frontend
npm run build
```

- [ ] **Step 5: 提交**

```bash
git add frontend/src/App.tsx
git commit -m "refactor: App.tsx to tab-based navigation layout"
```

---

## Task 7: 更新首启向导浅色主题

**Files:**
- Modify: `frontend/src/components/FirstRunOnboarding.tsx`
- Modify: `frontend/src/components/OnboardingReminder.tsx`

- [ ] **Step 1: 更新 FirstRunOnboarding**

更新 CSS class 引用，使用浅色卡片和按钮样式。保留 welcome/config 两步流程逻辑。

- [ ] **Step 2: 更新 OnboardingReminder**

更新为浅色提醒条样式。

- [ ] **Step 3: 运行构建验证**

```bash
cd frontend
npm run build
```

- [ ] **Step 4: 提交**

```bash
git add frontend/src/components/FirstRunOnboarding.tsx frontend/src/components/OnboardingReminder.tsx
git commit -m "style: update onboarding components to light theme"
```

---

## Task 8: 更新测试适配新 UI

**Files:**
- Modify: `frontend/src/App.test.tsx`
- Modify: 各组件 `*.test.tsx`

- [ ] **Step 1: 更新 App.test.tsx**

更新文本断言：
- "运行概览" → 新 UI 对应文本
- "主操作" → 新 UI 对应文本
- "常用入口" → 新 UI 对应文本
- "基础代理" → 新 Tab 导航按钮文本
- "运行日志" → 新 Tab 导航按钮文本
- "下一步建议" → 移除或更新（新 UI 无 summary-strip）
- 按钮名称（"启动代理"）保持不变

- [ ] **Step 2: 更新各组件测试文件**

逐个更新 StatusCard.test.tsx、PrimaryActionCard.test.tsx 等中的文本断言。

- [ ] **Step 3: 运行全部测试**

```bash
cd frontend
npm run test
```
Expected: 全部通过

- [ ] **Step 4: 提交**

```bash
git add frontend/src
git commit -m "test: update tests for redesigned UI"
```

---

## Task 9: 清理旧代码

**Files:**
- Delete: `frontend/src/styles.css.bak`（如果存在）
- Modify: `frontend/src/styles.css` — 删除不再使用的旧 CSS class

- [ ] **Step 1: 查找并清除无引用的旧 CSS class**

搜索 styles.css 中的 class 名，确认在组件中不再使用后删除。

- [ ] **Step 2: 运行完整验证**

```bash
cd frontend
npm run test
npm run build
```
Expected: 测试全通过，构建成功

- [ ] **Step 3: 提交**

```bash
git add -A
git commit -m "chore: clean up unused CSS from old design"
```

---

## 验证计划

### 自动化测试

```bash
cd frontend
npm run test
```

全部现有测试（加上 Task 8 中更新后的测试）应通过。测试覆盖：
- 控制面板基本渲染
- Preview 模式数据展示
- 启动失败错误显示
- 启动按钮禁用状态
- 停止/运行状态文本
- 保存配置后重启提示
- 首启向导显示/跳过/恢复/保存

### 构建验证

```bash
cd frontend
npm run build
```

构建应成功且无报错。

### 手动验证（需用户配合）

1. 运行 `wails dev` 或 `go run .` 启动应用
2. 检查三个 Tab 是否正常切换
3. 确认浅色主题视觉效果
4. 测试启动/停止代理按钮
5. 测试设置页保存功能
6. 测试日志页自动滚动
7. 测试首启向导流程
