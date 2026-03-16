# Aether 首启向导 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a lightweight first-run onboarding overlay that appears when the user has no real proxy configuration yet, allows skipping, and keeps a visible reminder in the main window until configuration is completed.

**Architecture:** Add one backend onboarding-state binding based on the real config file, reuse the existing `SaveBasicProxySettings()` flow for onboarding completion, and render a React overlay + reminder banner in `App` without introducing a separate window or extra persistent flags.

**Tech Stack:** Go 1.25, existing `internal/config` + Wails bindings in `app.go`, React 18, TypeScript, Vitest, Testing Library.

---

### Task 1: Add backend onboarding-state detection

**Files:**
- Create: `internal/config/onboarding_test.go`
- Modify: `internal/config/config.go`

**Step 1: Write the failing test**

Create `internal/config/onboarding_test.go`:

```go
func TestShouldShowOnboardingWhenConfigMissing(t *testing.T) {
    path := filepath.Join(t.TempDir(), "config.json")

    state, err := DetectOnboardingState(path)
    if err != nil {
        t.Fatalf("DetectOnboardingState error: %v", err)
    }

    if !state.ShouldShowOnboarding {
        t.Fatal("expected onboarding when config is missing")
    }
}

func TestShouldHideOnboardingWhenProxyConfigIsCustomized(t *testing.T) {
    path := filepath.Join(t.TempDir(), "config.json")
    cfg := DefaultConfig()
    cfg.Proxy.Host = "10.0.0.2"
    if err := Save(path, cfg); err != nil {
        t.Fatalf("seed config: %v", err)
    }

    state, err := DetectOnboardingState(path)
    if err != nil {
        t.Fatalf("DetectOnboardingState error: %v", err)
    }

    if state.ShouldShowOnboarding {
        t.Fatal("expected onboarding to stay hidden once proxy config is customized")
    }
}
```

**Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/config -run "TestShouldShowOnboardingWhenConfigMissing|TestShouldHideOnboardingWhenProxyConfigIsCustomized" -v
```

Expected: FAIL because `DetectOnboardingState` and its state type do not exist yet.

**Step 3: Write minimal implementation**

In `internal/config/config.go`, add:

- `type OnboardingState struct`
- `func DetectOnboardingState(path string) (OnboardingState, error)`

Rules:

- `configExists = false` when file is missing
- `shouldShowOnboarding = true` when file missing
- `shouldShowOnboarding = true` when `proxy.host/port/type` still equal defaults
- otherwise `shouldShowOnboarding = false`

**Step 4: Run test to verify it passes**

Run:

```bash
go test ./internal/config -run "TestShouldShowOnboardingWhenConfigMissing|TestShouldHideOnboardingWhenProxyConfigIsCustomized" -v
```

Expected: PASS.

### Task 2: Expose onboarding binding through Wails

**Files:**
- Modify: `app.go`
- Modify: `app_test.go`

**Step 1: Write the failing test**

Add tests to `app_test.go`:

```go
func TestGetOnboardingStateUsesDefaultPathsConfig(t *testing.T) {
    t.Setenv("LOCALAPPDATA", t.TempDir())

    app := &App{controller: gui.NewController(guiTestLauncher{}, unavailableClient{})}
    state, err := app.GetOnboardingState()
    if err != nil {
        t.Fatalf("GetOnboardingState error: %v", err)
    }

    if !state.ShouldShowOnboarding {
        t.Fatal("expected onboarding on first run")
    }
}
```

**Step 2: Run test to verify it fails**

Run:

```bash
go test . -run TestGetOnboardingStateUsesDefaultPathsConfig -v
```

Expected: FAIL because `GetOnboardingState()` does not exist in `App`.

**Step 3: Write minimal implementation**

In `app.go`, add:

- `func (a *App) GetOnboardingState() (config.OnboardingState, error)`

Implementation should call `config.DetectOnboardingState(paths.DefaultPaths().ConfigFile)`.

**Step 4: Run test to verify it passes**

Run:

```bash
go test . -run TestGetOnboardingStateUsesDefaultPathsConfig -v
```

Expected: PASS.

### Task 3: Build onboarding overlay and reminder banner components

**Files:**
- Create: `frontend/src/components/FirstRunOnboarding.tsx`
- Create: `frontend/src/components/FirstRunOnboarding.test.tsx`
- Create: `frontend/src/components/OnboardingReminder.tsx`
- Create: `frontend/src/components/OnboardingReminder.test.tsx`
- Modify: `frontend/src/styles.css`

**Step 1: Write the failing tests**

Create tests for:

- onboarding welcome step renders `开始配置` and `暂时跳过`
- reminder banner renders `继续配置` and `打开配置文件`

**Step 2: Run tests to verify they fail**

Run:

```bash
cd frontend
npm run test -- src/components/FirstRunOnboarding.test.tsx src/components/OnboardingReminder.test.tsx
```

Expected: FAIL because the components do not exist yet.

**Step 3: Write minimal implementation**

Implement:

- `FirstRunOnboarding` with two steps:
  - welcome
  - basic proxy config form
- `OnboardingReminder` as a non-blocking banner shown after skip

Keep the onboarding form limited to:

- `代理地址`
- `代理端口`
- `代理类型`

Actions:

- `开始配置`
- `暂时跳过`
- `返回`
- `保存并进入主界面`

**Step 4: Run tests to verify they pass**

Run:

```bash
cd frontend
npm run test -- src/components/FirstRunOnboarding.test.tsx src/components/OnboardingReminder.test.tsx
```

Expected: PASS.

### Task 4: Wire onboarding state into the app shell

**Files:**
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/App.test.tsx`

**Step 1: Write the failing test**

Extend `frontend/src/App.test.tsx` with scenarios:

- first run shows onboarding overlay
- skip hides overlay and shows reminder banner
- reminder `继续配置` reopens overlay
- onboarding save closes overlay and hides reminder

**Step 2: Run test to verify it fails**

Run:

```bash
cd frontend
npm run test -- src/App.test.tsx
```

Expected: FAIL because the App does not yet fetch onboarding state or render the new components.

**Step 3: Write minimal implementation**

Update `frontend/src/types.ts` with:

- `OnboardingState`
- backend binding `GetOnboardingState`

Update `App.tsx` to:

- fetch onboarding state on init
- show onboarding overlay when needed
- allow `暂时跳过` to dismiss overlay in-session
- show reminder banner while onboarding is still incomplete
- reopen overlay from reminder banner
- reuse existing proxy settings save flow to complete onboarding

**Step 4: Run test to verify it passes**

Run:

```bash
cd frontend
npm run test -- src/App.test.tsx
```

Expected: PASS.

### Task 5: Final verification and docs

**Files:**
- Modify: `README.md`
- Modify: `docs/gui-smoke-test.md`

**Step 1: Update docs**

Document:

- when onboarding appears
- that users may skip
- that reminder stays visible until non-default proxy config exists

**Step 2: Run targeted verification**

Run:

```bash
go test ./internal/config -run "TestShouldShowOnboardingWhenConfigMissing|TestShouldHideOnboardingWhenProxyConfigIsCustomized" -v
go test . -run TestGetOnboardingStateUsesDefaultPathsConfig -v
cd frontend
npm run test -- src/components/FirstRunOnboarding.test.tsx src/components/OnboardingReminder.test.tsx
npm run test -- src/App.test.tsx
cd ..
```

Expected: PASS.

**Step 3: Run broader verification**

Run:

```bash
go test ./...
cd frontend
npm run test
npm run build
cd ..
go run github.com/wailsapp/wails/v2/cmd/wails@v2.11.0 build -platform windows/amd64
```

Expected: all checks pass and a fresh Windows GUI binary is produced.

**Step 4: Manual smoke**

Verify:

- first run shows onboarding overlay
- skip hides overlay and keeps reminder visible
- clicking `继续配置` reopens onboarding
- saving onboarding settings closes overlay
- once config is non-default, onboarding no longer appears on next launch

## Notes for the Implementer

- Do not add a separate onboarding persistence file.
- Keep the first-run decision tied to the real config file only.
- Do not add connection tests in this version.
- Reuse the existing basic proxy save flow whenever possible.
- Keep the overlay simple and visually aligned with the current dashboard.

## Done Criteria

- First-run users automatically see onboarding when config is missing or still default.
- Users may skip onboarding, but the main UI keeps a visible reminder.
- Saving onboarding settings removes the reminder and prevents future onboarding prompts.
- Automated Go, frontend, and Wails build verification all pass.
