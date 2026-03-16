# Aether Portable Package Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Produce a beginner-friendly Windows portable package that can be unzipped and run directly, while also shipping a clear example configuration file.

**Architecture:** Keep the runtime behavior unchanged: the app still stores live config under `%LOCALAPPDATA%\Aether`. Add a tracked portable packaging template directory, create a PowerShell packaging script that assembles the built binaries plus helper files into a staging folder, and emit a versionless `Aether-portable.zip` from `build/bin`.

**Tech Stack:** Go 1.25 tests, PowerShell 5+ packaging script, existing Wails build output under `build/bin`, JSON config assets, Windows `Compress-Archive`.

---

### Task 1: Lock the example config asset to the runtime defaults

**Files:**
- Create: `internal/config/example_config_test.go`
- Create: `packaging/portable/config.example.json`

**Step 1: Write the failing test**

Add a Go test that loads `packaging/portable/config.example.json`, unmarshals it into `config.Config`, and compares it with `config.DefaultConfig()`.

**Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/config -run TestPortableConfigExampleMatchesDefaultConfig -v
```

Expected: FAIL because `packaging/portable/config.example.json` does not exist yet.

**Step 3: Write minimal implementation**

Create `packaging/portable/config.example.json` with the exact JSON emitted by `DefaultConfig()`.

**Step 4: Run test to verify it passes**

Run:

```bash
go test ./internal/config -run TestPortableConfigExampleMatchesDefaultConfig -v
```

Expected: PASS.

### Task 2: Add portable package helper assets

**Files:**
- Create: `packaging/portable/README.txt`

**Step 1: Define the required contents**

Document, in plain Chinese, that:
- users launch `Aether.exe`;
- the real config file is `%LOCALAPPDATA%\Aether\config.json`;
- `config.example.json` is only a template;
- `aether-core.exe` and `wintun.dll` must stay beside `Aether.exe`.

**Step 2: Write minimal implementation**

Create `packaging/portable/README.txt` with unzip-and-run instructions suitable for novice users.

**Step 3: Manually verify content**

Open the file and confirm the wording matches the current product behavior and paths.

### Task 3: Add a reproducible portable packaging script

**Files:**
- Create: `scripts/package-portable.ps1`

**Step 1: Write the failing check**

Run:

```bash
powershell -ExecutionPolicy Bypass -File scripts/package-portable.ps1
```

Expected: FAIL because the script does not exist yet.

**Step 2: Write minimal implementation**

Implement a PowerShell script that:
- validates `build/bin/Aether.exe`, `build/bin/aether-core.exe`, and `build/bin/wintun.dll` exist;
- creates `build/bin/Aether-portable/`;
- copies the binaries, `packaging/portable/config.example.json`, and `packaging/portable/README.txt`;
- deletes any old `build/bin/Aether-portable.zip`;
- creates a fresh `build/bin/Aether-portable.zip`.

**Step 3: Run script to verify it passes**

Run:

```bash
powershell -ExecutionPolicy Bypass -File scripts/package-portable.ps1
```

Expected: PASS and both the staging folder plus zip file exist.

### Task 4: Document the release flow

**Files:**
- Modify: `README.md`

**Step 1: Update packaging docs**

Add a short section describing how to generate the portable zip and list its contents.

**Step 2: Run a quick verification**

Run:

```bash
Get-ChildItem build/bin/Aether-portable*
```

Expected: the staging folder and zip are visible.

### Task 5: Final verification

**Files:**
- No new files

**Step 1: Run focused config asset test**

Run:

```bash
go test ./internal/config -run TestPortableConfigExampleMatchesDefaultConfig -v
```

Expected: PASS.

**Step 2: Run broader automated verification**

Run:

```bash
go test ./...
cd frontend
npm run test
npm run build
cd ..
```

Expected: all checks pass.

**Step 3: Rebuild and package**

Run:

```bash
go run github.com/wailsapp/wails/v2/cmd/wails@v2.11.0 build -platform windows/amd64
powershell -ExecutionPolicy Bypass -File scripts/package-portable.ps1
```

Expected: fresh binaries and `build/bin/Aether-portable.zip` are produced.
