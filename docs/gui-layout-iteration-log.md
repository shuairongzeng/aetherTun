# Aether GUI Layout Iteration Log

## Baseline

- `round-00-baseline.png`
- 说明：改版前基线，用于对照后续所有轮次。

## Round 1

- 目标：建立“运行概览 / 主操作 / 基础代理 / 常用入口 / 运行日志”的稳定信息骨架。
- 自测：`cmd /c "cd frontend && npm run test -- --run src/App.test.tsx src/components/StatusCard.test.tsx src/components/PrimaryActionCard.test.tsx src/components/QuickActionsCard.test.tsx"`
- 截图：`docs/screenshots/gui-iterations/round-01.png`

## Round 2

- 目标：提升字号、留白和卡片高度，缓解首屏拥挤感。
- 自测：同 Round 1 聚焦测试。
- 截图：`docs/screenshots/gui-iterations/round-02.png`

## Round 3

- 目标：把日志区从右下角释放成全宽区域，重新拉回页面重心。
- 自测：`cmd /c "cd frontend && npm run test -- --run src/App.test.tsx src/components/StatusCard.test.tsx src/components/PrimaryActionCard.test.tsx src/components/QuickActionsCard.test.tsx src/components/RecentLogsCard.test.tsx"`
- 截图：`docs/screenshots/gui-iterations/round-03.png`

## Round 4

- 目标：增加“当前阶段 / 下一步建议 / 日志行为”摘要条，降低新手决策成本。
- 自测：同 Round 3 聚焦测试。
- 截图：`docs/screenshots/gui-iterations/round-04.png`

## Round 5

- 目标：为基础代理卡增加当前上游概览和显式说明，让配置更可扫读。
- 自测：`cmd /c "cd frontend && npm run test -- --run src/components/BasicProxyConfigCard.test.tsx src/components/RecentLogsCard.test.tsx"`
- 截图：`docs/screenshots/gui-iterations/round-05.png`

## Round 6

- 目标：重排配置卡底部操作区，让“保存配置”更突出、次操作更退后。
- 自测：`cmd /c "cd frontend && npm run test -- --run src/App.test.tsx src/components/BasicProxyConfigCard.test.tsx"`
- 截图：`docs/screenshots/gui-iterations/round-06.png`

## Round 7

- 目标：把日志列表强化为时间线式扫描结构，提高时间、来源和级别的辨识度。
- 自测：`cmd /c "cd frontend && npm run test -- --run src/components/RecentLogsCard.test.tsx src/App.test.tsx"`
- 截图：`docs/screenshots/gui-iterations/round-07.png`

## Round 8

- 目标：把首启向导升级成“主内容 + 侧边指导”，同时增强跳过后的提醒条。
- 自测：`cmd /c "cd frontend && npm run test -- --run src/components/FirstRunOnboarding.test.tsx src/components/OnboardingReminder.test.tsx"`
- 截图：`docs/screenshots/gui-iterations/round-08.png`
- 备注：截图使用 `onboarding` 预览场景。

## Round 9

- 目标：为主操作卡增加能力提示条，让用户更清楚 UAC、托盘和日志同步行为。
- 自测：`cmd /c "cd frontend && npm run test -- --run src/components/PrimaryActionCard.test.tsx src/App.test.tsx"`
- 截图：`docs/screenshots/gui-iterations/round-09.png`

## Round 10

- 目标：做最终视觉收口，统一背景层次、卡片材质、阴影和对比度。
- 自测：`cmd /c "cd frontend && npm run test -- --run"`
- 截图：`docs/screenshots/gui-iterations/round-10.png`

## Screenshot Workflow

- 构建：`cmd /c "cd frontend && npm run build"`
- 截图：`node scripts/capture-preview-screenshot.mjs <output> <running|onboarding>`

## Final Evidence

- 全量前端测试：通过
- 前端生产构建：通过
- 已保留 10 轮截图和 1 张基线图，便于人工 review
