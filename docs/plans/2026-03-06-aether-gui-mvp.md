# Aether GUI MVP Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build an installable Windows GUI for Aether that lets novice users start/stop the proxy, minimize to tray, inspect current status, and open config/logs without touching the existing networking core logic more than necessary.

**Architecture:** Extract the current CLI startup flow into a reusable `runtime.Manager`, run it inside a separately elevated `aether-core` process, expose a loopback-only control API, and place a `Wails v2.11.0 + React/TypeScript` GUI in front of it. Keep the legacy CLI as a separate command for debugging, and add a tray controller plus installer-friendly app paths for end-user distribution.

**Tech Stack:** Go 1.25, existing `internal/config|dns|routing|tun`, `github.com/wailsapp/wails/v2@v2.11.0`, React, TypeScript, Vite, `github.com/getlantern/systray@v1.2.2`, Windows UAC / ShellExecute, NSIS via Wails build pipeline.

---

## Preflight

Before task work begins, create an isolated worktree and install the desktop toolchain:

```bash
git worktree add ..\\aether-gui-mvp -b feat/aether-gui-mvp
go install github.com/wailsapp/wails/v2/cmd/wails@v2.11.0
cd ..\\aether-gui-mvp
wails doctor
```

Expected:

- A clean feature worktree exists.
- `wails doctor` reports Windows prerequisites as ready.

### Task 1: Extract runtime lifecycle orchestration

**Files:**
- Create: `internal/runtime/manager.go`
- Create: `internal/runtime/types.go`
- Create: `internal/runtime/manager_test.go`
- Modify: `main.go`

**Step 1: Write the failing test**

Create `internal/runtime/manager_test.go` with fake components and a lifecycle test:

```go
func TestManagerTransitionsThroughStartAndStop(t *testing.T) {
    fake := newFakeFactory()
    manager := NewManager(fake)

    require.Equal(t, PhaseStopped, manager.Status().Phase)
    require.NoError(t, manager.Start(context.Background()))
    require.Equal(t, PhaseRunning, manager.Status().Phase)

    require.NoError(t, manager.Stop(context.Background()))
    require.Equal(t, PhaseStopped, manager.Status().Phase)
    require.Equal(t, []string{"router", "tun", "dns"}, fake.started)
    require.Equal(t, []string{"dns", "tun", "router"}, fake.stopped)
}
```

**Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/runtime -run TestManagerTransitionsThroughStartAndStop -v
```

Expected: FAIL because `NewManager`, `PhaseStopped`, and fake lifecycle interfaces do not exist yet.

**Step 3: Write minimal implementation**

Implement:

- `RuntimePhase` enum (`stopped`, `starting`, `running`, `stopping`, `error`);
- `RuntimeStatus` struct;
- `Manager` that accepts injected component factories;
- `Start()` ordering: router → tun → dns;
- `Stop()` ordering: dns → tun → router;
- `main.go` rewritten as thin CLI wrapper around `runtime.Manager`.

**Step 4: Run test to verify it passes**

Run:

```bash
go test ./internal/runtime -run TestManagerTransitionsThroughStartAndStop -v
```

Expected: PASS.

**Step 5: Commit**

```bash
git add main.go internal/runtime
git commit -m "refactor: extract runtime lifecycle manager"
```

### Task 2: Build the production runtime adapter around existing core packages

**Files:**
- Create: `internal/runtime/live_factory.go`
- Create: `internal/runtime/live_factory_test.go`
- Modify: `internal/config/config.go`
- Modify: `internal/dns/server.go`
- Modify: `internal/tun/engine.go`

**Step 1: Write the failing test**

Create a test for config bootstrap and factory wiring:

```go
func TestLiveFactoryCreatesDefaultConfigWhenMissing(t *testing.T) {
    tempDir := t.TempDir()
    configPath := filepath.Join(tempDir, "config.json")

    factory := NewLiveFactory(configPath)
    cfg, err := factory.LoadConfig()

    require.NoError(t, err)
    require.FileExists(t, configPath)
    require.Equal(t, "127.0.0.1", cfg.Proxy.Host)
}
```

**Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/runtime -run TestLiveFactoryCreatesDefaultConfigWhenMissing -v
```

Expected: FAIL because `NewLiveFactory` and automatic bootstrap behavior do not exist.

**Step 3: Write minimal implementation**

Implement:

- `NewLiveFactory(configPath string)` that writes `config.DefaultConfig()` when config is missing;
- light wrappers that construct `routing.Engine`, `dns.Server`, `tun.Engine`;
- non-invasive hooks in `internal/dns/server.go` and `internal/tun/engine.go` so runtime can pass logger/status callbacks later without changing protocol logic;
- preserve the existing current behavior for CLI mode.

**Step 4: Run test to verify it passes**

Run:

```bash
go test ./internal/runtime -run TestLiveFactoryCreatesDefaultConfigWhenMissing -v
```

Expected: PASS.

**Step 5: Commit**

```bash
git add internal/runtime internal/config/config.go internal/dns/server.go internal/tun/engine.go
git commit -m "feat: add live runtime factory and config bootstrap"
```

### Task 3: Add structured status and recent-log storage

**Files:**
- Create: `internal/logs/store.go`
- Create: `internal/logs/store_test.go`
- Modify: `internal/runtime/manager.go`
- Modify: `internal/runtime/types.go`

**Step 1: Write the failing test**

Add a store capacity test:

```go
func TestStoreKeepsRecentEntriesInOrder(t *testing.T) {
    store := NewStore(3)
    store.Append(Entry{Message: "1"})
    store.Append(Entry{Message: "2"})
    store.Append(Entry{Message: "3"})
    store.Append(Entry{Message: "4"})

    entries := store.Recent(3)
    require.Len(t, entries, 3)
    require.Equal(t, []string{"2", "3", "4"}, []string{
        entries[0].Message, entries[1].Message, entries[2].Message,
    })
}
```

**Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/logs -run TestStoreKeepsRecentEntriesInOrder -v
```

Expected: FAIL because `Store` and `Entry` do not exist.

**Step 3: Write minimal implementation**

Implement:

- bounded in-memory ring buffer for recent logs;
- optional file writer for persistent logs in `%LocalAppData%\Aether\logs`;
- runtime manager hooks for phase transitions and key events (`starting`, `running`, `stop requested`, `error`);
- `LastErrorCode` / `LastErrorText` population in `RuntimeStatus`.

**Step 4: Run test to verify it passes**

Run:

```bash
go test ./internal/logs -run TestStoreKeepsRecentEntriesInOrder -v
```

Expected: PASS.

**Step 5: Commit**

```bash
git add internal/logs internal/runtime
git commit -m "feat: add runtime status and recent log store"
```

### Task 4: Expose the core control API

**Files:**
- Create: `internal/control/types.go`
- Create: `internal/control/server.go`
- Create: `internal/control/server_test.go`
- Create: `internal/control/client.go`
- Modify: `internal/runtime/manager.go`

**Step 1: Write the failing test**

Create an HTTP server test:

```go
func TestServerReturnsStatusAndStopsManager(t *testing.T) {
    manager := newFakeManager(PhaseRunning)
    logs := logs.NewStore(10)
    srv := NewServer(manager, logs, "token-123")

    ts := httptest.NewServer(srv.Handler())
    defer ts.Close()

    req, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/status", nil)
    req.Header.Set("Authorization", "Bearer token-123")
    resp, err := http.DefaultClient.Do(req)

    require.NoError(t, err)
    require.Equal(t, http.StatusOK, resp.StatusCode)
}
```

**Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/control -run TestServerReturnsStatusAndStopsManager -v
```

Expected: FAIL because the control server does not exist yet.

**Step 3: Write minimal implementation**

Implement:

- `GET /v1/status`
- `GET /v1/meta`
- `GET /v1/logs/recent?limit=50`
- `POST /v1/stop`
- bearer token validation;
- client helpers used by GUI backend.

Bind only to `127.0.0.1`.

**Step 4: Run test to verify it passes**

Run:

```bash
go test ./internal/control -run TestServerReturnsStatusAndStopsManager -v
```

Expected: PASS.

**Step 5: Commit**

```bash
git add internal/control internal/runtime/manager.go
git commit -m "feat: add localhost control api for elevated core"
```

### Task 5: Add Windows paths and elevation launcher

**Files:**
- Create: `internal/paths/paths.go`
- Create: `internal/paths/paths_test.go`
- Create: `internal/launcher/elevate_windows.go`
- Create: `internal/launcher/elevate_windows_test.go`
- Modify: `internal/runtime/live_factory.go`

**Step 1: Write the failing test**

Add a path resolution test:

```go
func TestDefaultPathsLiveUnderLocalAppData(t *testing.T) {
    t.Setenv("LOCALAPPDATA", `C:\Users\Test\AppData\Local`)

    paths := DefaultPaths()

    require.Equal(t, `C:\Users\Test\AppData\Local\Aether\config.json`, paths.ConfigFile)
    require.Equal(t, `C:\Users\Test\AppData\Local\Aether\logs`, paths.LogDir)
}
```

**Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/paths -run TestDefaultPathsLiveUnderLocalAppData -v
```

Expected: FAIL because `DefaultPaths()` does not exist.

**Step 3: Write minimal implementation**

Implement:

- `DefaultPaths()` for config, logs, runtime dir;
- `EnsureAppDirs()` helper;
- `LaunchElevatedCore()` using `ShellExecute("runas", ...)`;
- tests for commandline construction and paths;
- note: actual UAC prompt is verified manually, not via unit test.

**Step 4: Run test to verify it passes**

Run:

```bash
go test ./internal/paths -run TestDefaultPathsLiveUnderLocalAppData -v
go test ./internal/launcher -v
```

Expected: PASS for path tests and commandline builder tests.

**Step 5: Commit**

```bash
git add internal/paths internal/launcher internal/runtime/live_factory.go
git commit -m "feat: add windows app paths and elevated core launcher"
```

### Task 6: Split entrypoints into GUI, core, and legacy CLI

**Files:**
- Create: `cmd/aether-cli/main.go`
- Create: `cmd/aether-core/main.go`
- Create: `cmd/aether-core/main_test.go`
- Modify: `main.go`
- Modify: `go.mod`

**Step 1: Write the failing test**

Add a flag parsing test for core mode:

```go
func TestParseCoreFlags(t *testing.T) {
    cfg := parseFlags([]string{
        "--config", "x.json",
        "--control-port", "43129",
        "--token", "abc",
    })

    require.Equal(t, "x.json", cfg.ConfigPath)
    require.Equal(t, 43129, cfg.ControlPort)
    require.Equal(t, "abc", cfg.Token)
}
```

**Step 2: Run test to verify it fails**

Run:

```bash
go test ./cmd/aether-core -run TestParseCoreFlags -v
```

Expected: FAIL because the new command entrypoints do not exist yet.

**Step 3: Write minimal implementation**

Implement:

- move the current CLI behavior into `cmd/aether-cli/main.go`;
- add `cmd/aether-core/main.go` that creates `runtime.Manager`, `logs.Store`, `control.Server`;
- make root `main.go` the future Wails GUI entry placeholder (it can fail fast or show TODO until Task 7 is done);
- update `go.mod` with new GUI/tray dependencies only after `cmd` split is clean.

**Step 4: Run test to verify it passes**

Run:

```bash
go test ./cmd/aether-core -run TestParseCoreFlags -v
go build ./cmd/aether-core
go build ./cmd/aether-cli
```

Expected: PASS for tests and both binaries compile.

**Step 5: Commit**

```bash
git add main.go cmd go.mod go.sum
git commit -m "refactor: split gui core and cli entrypoints"
```

### Task 7: Bootstrap the Wails shell and Go GUI backend

**Files:**
- Create: `app.go`
- Create: `wails.json`
- Create: `frontend/package.json`
- Create: `frontend/tsconfig.json`
- Create: `frontend/vite.config.ts`
- Create: `frontend/src/main.tsx`
- Create: `frontend/src/App.tsx`
- Create: `frontend/src/App.test.tsx`
- Modify: `main.go`

**Step 1: Write the failing test**

Create a minimal frontend render test:

```tsx
import { render, screen } from "@testing-library/react";
import App from "./App";

it("shows the default stopped state", () => {
  render(<App />);
  expect(screen.getByText("未运行")).toBeInTheDocument();
});
```

**Step 2: Run test to verify it fails**

Run:

```bash
cd frontend
npm install
npm run test -- --run src/App.test.tsx
```

Expected: FAIL because the Wails frontend and `App` component do not exist yet.

**Step 3: Write minimal implementation**

Implement:

- `main.go` Wails bootstrap using `wails.Run`;
- `app.go` backend binding exposing placeholder methods (`GetStatus`, `StartCore`, `StopCore`, `GetRecentLogs`);
- `wails.json` with Windows title, asset dir, and NSIS target;
- `frontend` React + TypeScript + Vite skeleton;
- initial page with status pill, primary action button, quick action placeholders.

Set Wails window options to hide instead of closing when requested by the backend/tray controller.

**Step 4: Run test to verify it passes**

Run:

```bash
cd frontend
npm run test -- --run src/App.test.tsx
cd ..
go build .
```

Expected: PASS for the frontend test and the root GUI binary compiles.

**Step 5: Commit**

```bash
git add main.go app.go wails.json frontend
git commit -m "feat: bootstrap wails gui shell"
```

### Task 8: Implement GUI-to-core control flow

**Files:**
- Create: `internal/gui/controller.go`
- Create: `internal/gui/controller_test.go`
- Modify: `app.go`
- Modify: `internal/control/client.go`
- Modify: `frontend/src/App.tsx`

**Step 1: Write the failing test**

Create a controller test with fake launcher and fake client:

```go
func TestControllerStartsCoreWhenNotRunning(t *testing.T) {
    launcher := newFakeLauncher()
    client := newFakeClient(PhaseStopped)
    controller := NewController(launcher, client)

    err := controller.StartCore(context.Background())

    require.NoError(t, err)
    require.True(t, launcher.called)
}
```

**Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/gui -run TestControllerStartsCoreWhenNotRunning -v
```

Expected: FAIL because the GUI controller layer does not exist yet.

**Step 3: Write minimal implementation**

Implement:

- `internal/gui.Controller` orchestrating `FindRunningCore()`, `LaunchElevatedCore()`, and `control.Client`;
- backend methods used by Wails bindings;
- polling or timer-based refresh every 1-2 seconds;
- frontend wiring so the main button actually triggers `StartCore` / `StopCore`;
- disabled button states for `starting` and `stopping`.

**Step 4: Run test to verify it passes**

Run:

```bash
go test ./internal/gui -run TestControllerStartsCoreWhenNotRunning -v
cd frontend
npm run test -- --run src/App.test.tsx
cd ..
```

Expected: PASS.

**Step 5: Commit**

```bash
git add internal/gui app.go frontend/src/App.tsx internal/control/client.go
git commit -m "feat: wire gui actions to elevated core control flow"
```

### Task 9: Add tray behavior and close-to-tray UX

**Files:**
- Create: `internal/tray/controller_windows.go`
- Create: `internal/tray/menu_test.go`
- Modify: `main.go`
- Modify: `app.go`
- Modify: `frontend/src/App.tsx`

**Step 1: Write the failing test**

Add a menu mapping test:

```go
func TestMenuModelReflectsRunningState(t *testing.T) {
    model := BuildMenuModel(runtime.RuntimeStatus{Phase: runtime.PhaseRunning})

    require.Contains(t, model.Items[0].Title, "停止代理")
}
```

**Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/tray -run TestMenuModelReflectsRunningState -v
```

Expected: FAIL because the tray controller and menu model do not exist.

**Step 3: Write minimal implementation**

Implement:

- `systray`-backed tray controller for Windows;
- first menu item toggles between `启动代理` and `停止代理`;
- `打开 Aether` shows/restores window;
- `查看日志` opens log directory;
- intercept window close and hide to tray instead;
- one-time “已最小化到系统托盘” hint.

**Step 4: Run test to verify it passes**

Run:

```bash
go test ./internal/tray -run TestMenuModelReflectsRunningState -v
go build .
```

Expected: PASS for unit tests and GUI still compiles.

**Step 5: Commit**

```bash
git add internal/tray main.go app.go frontend/src/App.tsx
git commit -m "feat: add tray controls and close-to-tray behavior"
```

### Task 10: Finish the status page, quick actions, and log panel

**Files:**
- Create: `frontend/src/components/StatusCard.tsx`
- Create: `frontend/src/components/PrimaryActionCard.tsx`
- Create: `frontend/src/components/QuickActionsCard.tsx`
- Create: `frontend/src/components/RecentLogsCard.tsx`
- Create: `frontend/src/hooks/useRuntimeState.ts`
- Create: `frontend/src/types.ts`
- Create: `frontend/src/components/StatusCard.test.tsx`
- Modify: `frontend/src/App.tsx`

**Step 1: Write the failing test**

Create a component test:

```tsx
it("renders the running phase and proxy endpoint", () => {
  render(<StatusCard status={{
    phase: "running",
    proxyEndpoint: "socks5://127.0.0.1:10808",
    tunAdapterName: "Aether-TUN"
  }} />);

  expect(screen.getByText("运行中")).toBeInTheDocument();
  expect(screen.getByText("socks5://127.0.0.1:10808")).toBeInTheDocument();
});
```

**Step 2: Run test to verify it fails**

Run:

```bash
cd frontend
npm run test -- --run src/components/StatusCard.test.tsx
```

Expected: FAIL because the new component tree does not exist.

**Step 3: Write minimal implementation**

Implement:

- status card with visual states: `未运行 / 启动中 / 运行中 / 停止中 / 启动失败`;
- primary action card with a single large CTA;
- quick actions: `打开配置文件` / `查看日志` / `开机自启`;
- recent logs card with last 20 entries;
- polling hook that refreshes status and logs from Wails backend.

**Step 4: Run test to verify it passes**

Run:

```bash
cd frontend
npm run test -- --run src/components/StatusCard.test.tsx
npm run build
cd ..
```

Expected: PASS and the frontend build succeeds.

**Step 5: Commit**

```bash
git add frontend/src
git commit -m "feat: build aether gui mvp status dashboard"
```

### Task 11: Add installer-facing polish and docs

**Files:**
- Modify: `README.md`
- Modify: `wails.json`
- Create: `docs/plans/2026-03-06-aether-gui-mvp-design.md`
- Create: `docs/gui-smoke-test.md`

**Step 1: Write the failing check**

Define the smoke checklist in `docs/gui-smoke-test.md` before changing docs:

```markdown
- [ ] GUI launches without admin
- [ ] Clicking start triggers UAC
- [ ] Running state appears in main window and tray
- [ ] Close hides to tray
- [ ] Stop from tray works
- [ ] Installer creates shortcuts
```

**Step 2: Run validation to confirm current repo does not satisfy it**

Run:

```bash
go build .
```

Expected: the binary may compile, but the checklist is clearly not yet satisfied; this is the baseline before doc updates.

**Step 3: Write minimal implementation**

Implement:

- README sections for GUI build, CLI build, config/log paths, and installer usage;
- `wails.json` product metadata, icon, NSIS target, and output naming;
- smoke test checklist for manual QA and release validation.

**Step 4: Run validation**

Run:

```bash
wails build -nsis
```

Expected: installer is produced successfully, or any missing asset error is explicit and fixable.

**Step 5: Commit**

```bash
git add README.md wails.json docs/gui-smoke-test.md docs/plans/2026-03-06-aether-gui-mvp-design.md
git commit -m "docs: document gui mvp build and smoke test flow"
```

### Task 12: Final verification pass

**Files:**
- Modify: `docs/gui-smoke-test.md`
- Modify: `README.md`

**Step 1: Run the automated suite**

Run:

```bash
go test ./...
cd frontend
npm run test
npm run build
cd ..
go build ./cmd/aether-core
go build ./cmd/aether-cli
go build .
```

Expected: all automated checks pass.

**Step 2: Run the Windows manual smoke**

Verify:

- launch GUI as standard user;
- click start and confirm UAC appears;
- verify `运行中` state is shown;
- close window and confirm tray persistence;
- stop from tray and verify state returns to `未运行`;
- run `wails build -nsis` and install/uninstall once.

**Step 3: Record results**

Update `docs/gui-smoke-test.md` with pass/fail notes and any known limitations.

**Step 4: Commit**

```bash
git add docs/gui-smoke-test.md README.md
git commit -m "test: verify aether gui mvp release candidate"
```

## Notes for the Implementer

- Keep all new Windows-specific logic behind `*_windows.go` where possible.
- Do not rewrite `internal/tun` networking behavior unless a lifecycle seam is strictly required.
- Prefer adding hooks/callbacks over changing packet-processing logic.
- If tray integration blocks Wails event handling, keep the menu model isolated so the implementation can swap out `systray` without rewriting the GUI controller.
- If actual TUN startup cannot be covered by CI, create a fake runtime mode for deterministic GUI tests.

## Done Criteria

- A standard Windows user can install and launch Aether without using the terminal.
- The main window shows clear stopped/running/error states.
- Clicking “启动代理” triggers UAC and results in a running Core process.
- Closing the window minimizes to tray instead of exiting.
- Tray menu supports start/stop/open/logs/exit.
- README and smoke docs explain how to build, package, and verify the GUI MVP.
