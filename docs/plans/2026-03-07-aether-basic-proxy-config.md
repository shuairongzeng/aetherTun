# Aether 基础代理配置 GUI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a beginner-friendly in-app editor for `proxy.host`, `proxy.port`, and `proxy.type`, with validation, save feedback, and an optional restart prompt when the core is already running.

**Architecture:** Reuse `internal/config` as the single source of truth, add minimal Wails bindings in `app.go` for reading and saving basic proxy settings, and introduce a dedicated React config card plus a small hook to manage dirty state, validation, and save/restart orchestration. Preserve all non-proxy config sections on save.

**Tech Stack:** Go 1.25, existing `internal/config` and `paths.DefaultPaths()`, Wails v2 bindings, React 18, TypeScript, Vitest, Testing Library.

---

### Task 1: Add config-domain helpers for basic proxy settings

**Files:**
- Create: `internal/config/basic_proxy_test.go`
- Modify: `internal/config/config.go`

**Step 1: Write the failing test**

Create `internal/config/basic_proxy_test.go` with two focused tests:

```go
func TestValidateBasicProxySettingsRejectsInvalidPort(t *testing.T) {
    err := ValidateBasicProxySettings(BasicProxySettings{
        Host: "127.0.0.1",
        Port: 70000,
        Type: "socks5",
    })

    if err == nil {
        t.Fatal("expected validation error")
    }
}

func TestSaveBasicProxySettingsPreservesAdvancedSections(t *testing.T) {
    path := filepath.Join(t.TempDir(), "config.json")
    cfg := DefaultConfig()
    cfg.Tun.AdapterName = "Custom-TUN"
    cfg.DNS.Upstream = "1.1.1.1:53"
    if err := Save(path, cfg); err != nil {
        t.Fatalf("seed config: %v", err)
    }

    saved, err := SaveBasicProxySettings(path, BasicProxySettings{
        Host: "10.0.0.2",
        Port: 7890,
        Type: "http",
    })
    if err != nil {
        t.Fatalf("save basic proxy settings: %v", err)
    }

    if saved.Proxy.Host != "10.0.0.2" || saved.Tun.AdapterName != "Custom-TUN" {
        t.Fatal("expected proxy updated and advanced sections preserved")
    }
}
```

**Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/config -run "TestValidateBasicProxySettings|TestSaveBasicProxySettingsPreservesAdvancedSections" -v
```

Expected: FAIL because `BasicProxySettings`, `ValidateBasicProxySettings`, and `SaveBasicProxySettings` do not exist yet.

**Step 3: Write minimal implementation**

In `internal/config/config.go`, add:

- `type BasicProxySettings struct { Host string; Port int; Type string }`
- `func LoadBasicProxySettings(path string) (BasicProxySettings, error)`
- `func ValidateBasicProxySettings(input BasicProxySettings) error`
- `func SaveBasicProxySettings(path string, input BasicProxySettings) (*Config, error)`

Rules:

- Trim host whitespace before validating
- Allow only `socks5` and `http`
- Keep `Tun`, `DNS`, `Routing`, and `LogLevel` untouched
- Reuse `LoadOrCreate()` and `Save()`

**Step 4: Run test to verify it passes**

Run:

```bash
go test ./internal/config -run "TestValidateBasicProxySettings|TestSaveBasicProxySettingsPreservesAdvancedSections" -v
```

Expected: PASS.

**Step 5: Commit**

```bash
git add internal/config/config.go internal/config/basic_proxy_test.go
git commit -m "feat: add basic proxy config helpers"
```

### Task 2: Expose Wails bindings for reading and saving proxy settings

**Files:**
- Modify: `app.go`
- Modify: `app_test.go`

**Step 1: Write the failing test**

Add tests to `app_test.go`:

```go
func TestGetBasicProxySettingsLoadsCurrentConfig(t *testing.T) {
    root := t.TempDir()
    t.Setenv("LOCALAPPDATA", root)

    cfg := config.DefaultConfig()
    cfg.Proxy.Host = "192.168.1.2"
    cfg.Proxy.Port = 8899
    cfg.Proxy.Type = "http"
    if err := config.Save(paths.DefaultPaths().ConfigFile, cfg); err != nil {
        t.Fatalf("seed config: %v", err)
    }

    app := &App{controller: gui.NewController(guiTestLauncher{}, unavailableClient{})}
    got, err := app.GetBasicProxySettings()
    if err != nil {
        t.Fatalf("GetBasicProxySettings error: %v", err)
    }

    if got.Host != "192.168.1.2" || got.Port != 8899 || got.Type != "http" {
        t.Fatalf("unexpected settings: %#v", got)
    }
}

func TestSaveBasicProxySettingsMarksRunningConfigForRestart(t *testing.T) {
    t.Setenv("LOCALAPPDATA", t.TempDir())
    app := &App{controller: gui.NewController(guiTestLauncher{}, runningClient{})}

    result, err := app.SaveBasicProxySettings(config.BasicProxySettings{
        Host: "127.0.0.1",
        Port: 7890,
        Type: "socks5",
    })
    if err != nil {
        t.Fatalf("SaveBasicProxySettings error: %v", err)
    }

    if !result.RequiresRestart {
        t.Fatal("expected requires restart when runtime is running")
    }
}
```

**Step 2: Run test to verify it fails**

Run:

```bash
go test . -run "TestGetBasicProxySettingsLoadsCurrentConfig|TestSaveBasicProxySettingsMarksRunningConfigForRestart" -v
```

Expected: FAIL because the new App methods and result type do not exist yet.

**Step 3: Write minimal implementation**

In `app.go`, add:

- `type SaveBasicProxySettingsResult struct`
- `func (a *App) GetBasicProxySettings() (config.BasicProxySettings, error)`
- `func (a *App) SaveBasicProxySettings(input config.BasicProxySettings) (SaveBasicProxySettingsResult, error)`

Behavior:

- Read/write via `paths.DefaultPaths().ConfigFile`
- Return normalized saved values
- Set `RequiresRestart` when `GetStatus().Phase == "running"`
- Do not stop/start automatically in the backend

**Step 4: Run test to verify it passes**

Run:

```bash
go test . -run "TestGetBasicProxySettingsLoadsCurrentConfig|TestSaveBasicProxySettingsMarksRunningConfigForRestart" -v
```

Expected: PASS.

**Step 5: Commit**

```bash
git add app.go app_test.go
git commit -m "feat: expose basic proxy config bindings"
```

### Task 3: Build the basic proxy config card UI

**Files:**
- Create: `frontend/src/components/BasicProxyConfigCard.tsx`
- Create: `frontend/src/components/BasicProxyConfigCard.test.tsx`
- Modify: `frontend/src/styles.css`

**Step 1: Write the failing test**

Create `frontend/src/components/BasicProxyConfigCard.test.tsx`:

```tsx
it("disables save until the form becomes dirty and valid", () => {
  render(
    <BasicProxyConfigCard
      value={{ host: "127.0.0.1", port: 10808, type: "socks5" }}
      dirty={false}
      saving={false}
      errors={{}}
      statusMessage=""
      onChange={() => {}}
      onSave={() => {}}
      onResetDefaults={() => {}}
      onOpenConfigFile={() => Promise.resolve()}
    />
  );

  expect(screen.getByRole("button", { name: "保存配置" })).toBeDisabled();
});

it("shows inline validation text", () => {
  render(
    <BasicProxyConfigCard
      value={{ host: "", port: 10808, type: "socks5" }}
      dirty
      saving={false}
      errors={{ host: "代理地址不能为空" }}
      statusMessage=""
      onChange={() => {}}
      onSave={() => {}}
      onResetDefaults={() => {}}
      onOpenConfigFile={() => Promise.resolve()}
    />
  );

  expect(screen.getByText("代理地址不能为空")).toBeInTheDocument();
});
```

**Step 2: Run test to verify it fails**

Run:

```bash
cd frontend
npm run test -- src/components/BasicProxyConfigCard.test.tsx
```

Expected: FAIL because the new component does not exist yet.

**Step 3: Write minimal implementation**

Implement `BasicProxyConfigCard.tsx` as a presentational component with:

- host text input
- port number input
- type select (`socks5` / `http`)
- buttons: `保存配置`, `恢复默认值`, `打开配置文件`
- inline error text
- success/info message area

Add matching styles to `frontend/src/styles.css` using the existing card visual language.

**Step 4: Run test to verify it passes**

Run:

```bash
cd frontend
npm run test -- src/components/BasicProxyConfigCard.test.tsx
```

Expected: PASS.

**Step 5: Commit**

```bash
git add frontend/src/components/BasicProxyConfigCard.tsx frontend/src/components/BasicProxyConfigCard.test.tsx frontend/src/styles.css
git commit -m "feat: add basic proxy config card"
```

### Task 4: Add frontend state management and restart prompt flow

**Files:**
- Create: `frontend/src/hooks/useBasicProxySettings.ts`
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/App.test.tsx`

**Step 1: Write the failing test**

Extend `frontend/src/App.test.tsx` with:

```tsx
it("prompts to restart when saving while the core is running", async () => {
  const stopCore = vi.fn().mockResolvedValue(undefined);
  const startCore = vi.fn().mockResolvedValue(undefined);
  vi.spyOn(window, "confirm").mockReturnValue(true);

  (window as any).go = {
    main: {
      App: {
        GetStatus: vi.fn().mockResolvedValue({ phase: "running" }),
        GetRecentLogs: vi.fn().mockResolvedValue([]),
        GetBasicProxySettings: vi.fn().mockResolvedValue({ host: "127.0.0.1", port: 10808, type: "socks5" }),
        SaveBasicProxySettings: vi.fn().mockResolvedValue({
          settings: { host: "127.0.0.1", port: 7890, type: "socks5" },
          requiresRestart: true
        }),
        StopCore: stopCore,
        StartCore: startCore
      }
    }
  };

  render(<App />);
  fireEvent.change(await screen.findByLabelText("代理端口"), { target: { value: "7890" } });
  fireEvent.click(screen.getByRole("button", { name: "保存配置" }));

  await waitFor(() => expect(stopCore).toHaveBeenCalledTimes(1));
  await waitFor(() => expect(startCore).toHaveBeenCalledTimes(1));
});
```

**Step 2: Run test to verify it fails**

Run:

```bash
cd frontend
npm run test -- src/App.test.tsx
```

Expected: FAIL because the App does not yet fetch/save proxy settings or handle restart confirmation.

**Step 3: Write minimal implementation**

Implement `useBasicProxySettings.ts` to manage:

- initial load from `GetBasicProxySettings`
- form state
- dirty detection
- field validation
- save lifecycle
- success/error message

Update `frontend/src/types.ts` with:

- `BasicProxySettings`
- `SaveBasicProxySettingsResult`
- backend bindings for `GetBasicProxySettings` and `SaveBasicProxySettings`

Update `frontend/src/App.tsx` to:

- render `BasicProxyConfigCard`
- pass `openConfigFile`
- on successful save with `requiresRestart === true`, use `window.confirm(...)`
- if confirmed, call `stopCore()` then `startCore()`

Keep restart orchestration in the frontend so the backend stays a simple config API.

**Step 4: Run test to verify it passes**

Run:

```bash
cd frontend
npm run test -- src/App.test.tsx
```

Expected: PASS.

**Step 5: Commit**

```bash
git add frontend/src/hooks/useBasicProxySettings.ts frontend/src/types.ts frontend/src/App.tsx frontend/src/App.test.tsx
git commit -m "feat: wire basic proxy config flow into app shell"
```

### Task 5: Document the new workflow and verify end-to-end

**Files:**
- Modify: `README.md`
- Modify: `docs/gui-smoke-test.md`

**Step 1: Write the documentation update**

Add:

- where the GUI-edited config lives
- which fields are editable in the GUI
- that advanced fields still require manual file edits
- that saving while running may require restart to apply

**Step 2: Run targeted automated verification**

Run:

```bash
go test ./internal/config -run "TestValidateBasicProxySettings|TestSaveBasicProxySettingsPreservesAdvancedSections" -v
go test . -run "TestGetBasicProxySettingsLoadsCurrentConfig|TestSaveBasicProxySettingsMarksRunningConfigForRestart" -v
cd frontend
npm run test -- src/components/BasicProxyConfigCard.test.tsx
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

Expected: all tests pass and the Windows GUI binary rebuilds successfully.

**Step 4: Smoke the feature manually**

Verify:

- GUI loads current `proxy.host`, `proxy.port`, `proxy.type`
- invalid host/port blocks save and shows inline errors
- saving while stopped updates `%LOCALAPPDATA%\Aether\config.json`
- saving while running prompts for restart
- confirming restart leads to stop/start success and updated status text
- cancelling restart leaves a clear “重启后生效” hint

**Step 5: Commit**

```bash
git add README.md docs/gui-smoke-test.md
git commit -m "docs: document basic proxy config gui flow"
```

## Notes for the Implementer

- Do not let the GUI overwrite `tun`, `dns`, `routing`, or `log_level`.
- Keep the first version limited to `proxy.host`, `proxy.port`, `proxy.type`.
- Prefer inline field errors over top-level generic errors.
- Use `window.confirm()` for the restart prompt in v1 instead of introducing modal infrastructure.
- Reuse existing button and card styles so the feature fits the current MVP shell.

## Done Criteria

- Users can edit `proxy.host`, `proxy.port`, and `proxy.type` directly in the GUI.
- Invalid input blocks save with inline Chinese validation messages.
- Saving updates the real config file under `%LOCALAPPDATA%\Aether\config.json`.
- Running-state saves clearly offer a restart path to apply the new config.
- The full Go, frontend, and Wails build verification suite passes.
