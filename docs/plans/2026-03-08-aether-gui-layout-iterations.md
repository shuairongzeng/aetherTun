# Aether GUI Layout Iterations Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Iteratively improve the Aether GUI layout over 10 self-reviewed rounds, with automated checks and screenshots after every round.

**Architecture:** Keep the existing Wails + React data flow intact, but reorganize the frontend into a clearer dashboard layout and add a browser preview mode for repeatable screenshots. Use component tests to drive structural changes and a lightweight screenshot workflow to capture each iteration.

**Tech Stack:** React 18, TypeScript, Vite, Vitest, Testing Library, Wails frontend bindings, PowerShell screenshot automation.

---

## Preflight

Before starting the iteration rounds:

- create a dedicated screenshot directory under `docs/screenshots/gui-iterations/`;
- add a browser preview scenario so the page can render stable mock data without Wails;
- define an iteration log file to record round goals, test commands, and screenshot paths.

### Task 1: Add browser preview scaffolding

**Files:**
- Create: `frontend/src/preview/mockBackend.ts`
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/hooks/useRuntimeState.ts`
- Modify: `frontend/src/hooks/useBasicProxySettings.ts`
- Test: `frontend/src/App.test.tsx`

**Step 1: Write the failing test**

Add a test that proves the app can render stable preview data when running without Wails bindings but with a preview scenario.

**Step 2: Run test to verify it fails**

Run:

```bash
cd frontend
npm run test -- --run src/App.test.tsx
```

Expected: FAIL because preview mode does not exist yet.

**Step 3: Write minimal implementation**

Implement:

- a query-param based preview mode;
- a mock backend for at least `running` and `onboarding` scenarios;
- no behavior change for real Wails runtime.

**Step 4: Run test to verify it passes**

Run:

```bash
cd frontend
npm run test -- --run src/App.test.tsx
```

Expected: PASS.

### Task 2: Iteration rounds 1-4 — establish layout hierarchy

**Files:**
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/components/StatusCard.tsx`
- Modify: `frontend/src/components/PrimaryActionCard.tsx`
- Modify: `frontend/src/components/QuickActionsCard.tsx`
- Modify: `frontend/src/styles.css`
- Test: `frontend/src/components/StatusCard.test.tsx`
- Test: `frontend/src/components/PrimaryActionCard.test.tsx`
- Test: `frontend/src/App.test.tsx`

**Step 1: Write the failing tests**

Add tests for:

- a top-level dashboard overview section;
- stronger status summary content;
- a primary action card that exposes next-step guidance;
- quick actions rendered as descriptive items instead of plain buttons.

**Step 2: Run tests to verify they fail**

Run:

```bash
cd frontend
npm run test -- --run src/components/StatusCard.test.tsx src/components/PrimaryActionCard.test.tsx src/App.test.tsx
```

Expected: FAIL because the new structure and copy do not exist yet.

**Step 3: Write minimal implementation**

Round-by-round within this task:

- Round 1: add screenshot workflow and capture baseline;
- Round 2: restructure the main shell into overview + content columns;
- Round 3: strengthen the status card with key metrics and a clearer hero band;
- Round 4: improve the primary action card and quick actions hierarchy.

**Step 4: Run tests after each round**

Run the focused frontend tests after every round.

**Step 5: Capture screenshot after each round**

Save one screenshot per round under `docs/screenshots/gui-iterations/round-0X.png`.

### Task 3: Iteration rounds 5-7 — improve work surfaces

**Files:**
- Modify: `frontend/src/components/BasicProxyConfigCard.tsx`
- Modify: `frontend/src/components/RecentLogsCard.tsx`
- Modify: `frontend/src/styles.css`
- Test: `frontend/src/components/BasicProxyConfigCard.test.tsx`
- Test: `frontend/src/components/RecentLogsCard.test.tsx`

**Step 1: Write the failing tests**

Add tests for:

- richer config card helper text / summary surface;
- log panel header metadata and improved empty-state guidance;
- preserved hidden-scrollbar behavior.

**Step 2: Run tests to verify they fail**

Run:

```bash
cd frontend
npm run test -- --run src/components/BasicProxyConfigCard.test.tsx src/components/RecentLogsCard.test.tsx
```

Expected: FAIL.

**Step 3: Write minimal implementation**

Round-by-round within this task:

- Round 5: improve quick-access configuration summary and field grouping;
- Round 6: add stronger action/status affordances inside the config card;
- Round 7: redesign the logs panel header, legend, and scrolling feel.

**Step 4: Run tests after each round**

Run focused tests after every round.

**Step 5: Capture screenshot after each round**

Save one screenshot per round.

### Task 4: Iteration rounds 8-10 — onboarding and polish

**Files:**
- Modify: `frontend/src/components/FirstRunOnboarding.tsx`
- Modify: `frontend/src/components/OnboardingReminder.tsx`
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/styles.css`
- Test: `frontend/src/components/FirstRunOnboarding.test.tsx`
- Test: `frontend/src/components/OnboardingReminder.test.tsx`
- Test: `frontend/src/App.test.tsx`

**Step 1: Write the failing tests**

Add tests for:

- a more structured onboarding layout;
- clearer reminder CTA area;
- final dashboard section labels / support text.

**Step 2: Run tests to verify they fail**

Run:

```bash
cd frontend
npm run test -- --run src/components/FirstRunOnboarding.test.tsx src/components/OnboardingReminder.test.tsx src/App.test.tsx
```

Expected: FAIL.

**Step 3: Write minimal implementation**

Round-by-round within this task:

- Round 8: redesign the onboarding overlay for stronger first-use guidance;
- Round 9: improve responsiveness, spacing, contrast, and action consistency;
- Round 10: polish microcopy, section framing, and final visual balance.

**Step 4: Run tests after each round**

Run focused tests after every round.

**Step 5: Capture screenshot after each round**

Save one screenshot per round.

### Task 5: Final verification and evidence log

**Files:**
- Create: `docs/gui-layout-iteration-log.md`
- Modify: `docs/gui-smoke-test.md`

**Step 1: Run the automated suite**

Run:

```bash
cd frontend
npm run test -- --run
npm run build
cd ..
go test ./...
```

Expected: PASS.

**Step 2: Record the 10-round evidence**

Document:

- each round’s goal;
- what changed;
- what test command was run;
- which screenshot was captured.

**Step 3: Optional app rebuild**

If frontend changes are stable, rebuild the Wails binary:

```bash
go run github.com/wailsapp/wails/v2/cmd/wails@v2.11.0 build -platform windows/amd64
```

Expected: PASS.
