import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, vi } from "vitest";
import App from "./App";

afterEach(() => {
  delete (
    window as Window & {
      go?: unknown;
    }
  ).go;
  window.history.pushState({}, "", "/");
  vi.restoreAllMocks();
});

it("renders the control shell", () => {
  render(<App />);

  expect(screen.getAllByText(/Aether/i).length).toBeGreaterThan(0);
  expect(screen.getByRole("tab", { name: "概览" })).toBeInTheDocument();
  expect(screen.getByRole("tab", { name: "设置" })).toBeInTheDocument();
  expect(screen.getByRole("tab", { name: "日志" })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "启动代理" })).toBeInTheDocument();
});

it("renders preview data when the preview scenario is present", async () => {
  window.history.pushState({}, "", "/?preview=running");

  render(<App />);

  expect((await screen.findAllByText("代理运行中")).length).toBeGreaterThan(0);
  expect(screen.getAllByText("后台核心已连接，代理正在运行。").length).toBeGreaterThan(0);
});

it("shows a launch error when starting the core fails", async () => {
  const startError = "aether-core.exe not found";

  (
    window as Window & {
      go?: {
        main?: {
          App?: Record<string, unknown>;
        };
      };
    }
  ).go = {
    main: {
      App: {
        GetStatus: vi.fn().mockResolvedValue({ phase: "stopped" }),
        GetRecentLogs: vi.fn().mockResolvedValue([]),
        StartCore: vi.fn().mockRejectedValue(new Error(startError))
      }
    }
  };

  render(<App />);

  fireEvent.click(screen.getByRole("button", { name: "启动代理" }));

  expect(await screen.findByText(startError)).toBeInTheDocument();
});

it("keeps the start button disabled while launch is still pending", async () => {
  const getStatus = vi.fn().mockResolvedValue({ phase: "stopped" });
  const getRecentLogs = vi.fn().mockResolvedValue([]);
  const startCore = vi.fn().mockResolvedValue(undefined);

  (
    window as Window & {
      go?: {
        main?: {
          App?: Record<string, unknown>;
        };
      };
    }
  ).go = {
    main: {
      App: {
        GetStatus: getStatus,
        GetRecentLogs: getRecentLogs,
        StartCore: startCore
      }
    }
  };

  render(<App />);

  const button = screen.getByRole("button", { name: "启动代理" });
  fireEvent.click(button);

  await waitFor(() => expect(startCore).toHaveBeenCalledTimes(1));
  await waitFor(() => expect(getStatus.mock.calls.length).toBeGreaterThanOrEqual(2));

  expect(button).toBeDisabled();
});

it("shows a stopped-state hint when the core is not running and there are no logs", async () => {
  (
    window as Window & {
      go?: {
        main?: {
          App?: Record<string, unknown>;
        };
      };
    }
  ).go = {
    main: {
      App: {
        GetStatus: vi.fn().mockResolvedValue({ phase: "stopped" }),
        GetRecentLogs: vi.fn().mockResolvedValue([])
      }
    }
  };

  render(<App />);

  // Switch to logs tab to see log hint
  fireEvent.click(screen.getByRole("tab", { name: "日志" }));

  expect(await screen.findByText(/后台核心未启动/)).toBeInTheDocument();
});

it("shows a running-state hint when the proxy is active but no logs have arrived yet", async () => {
  (
    window as Window & {
      go?: {
        main?: {
          App?: Record<string, unknown>;
        };
      };
    }
  ).go = {
    main: {
      App: {
        GetStatus: vi.fn().mockResolvedValue({ phase: "running" }),
        GetRecentLogs: vi.fn().mockResolvedValue([])
      }
    }
  };

  render(<App />);

  expect((await screen.findAllByText(/代理运行中/)).length).toBeGreaterThan(0);
});

it("prompts to restart when saving while the core is running", async () => {
  const stopCore = vi.fn().mockResolvedValue(undefined);
  const startCore = vi.fn().mockResolvedValue(undefined);
  vi.spyOn(window, "confirm").mockReturnValue(true);

  (
    window as Window & {
      go?: {
        main?: {
          App?: Record<string, unknown>;
        };
      };
    }
  ).go = {
    main: {
      App: {
        GetStatus: vi.fn().mockResolvedValue({ phase: "running" }),
        GetRecentLogs: vi.fn().mockResolvedValue([]),
        GetBasicProxySettings: vi.fn().mockResolvedValue({
          host: "127.0.0.1",
          port: 10808,
          type: "socks5"
        }),
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

  // Switch to settings tab
  fireEvent.click(screen.getByRole("tab", { name: "设置" }));

  fireEvent.change(await screen.findByLabelText(/代理端口/), { target: { value: "7890" } });
  fireEvent.click(screen.getByRole("button", { name: "保存配置" }));

  await waitFor(() => expect(stopCore).toHaveBeenCalledTimes(1));
  await waitFor(() => expect(startCore).toHaveBeenCalledTimes(1));
});

it("shows onboarding overlay on first run", async () => {
  (
    window as Window & {
      go?: {
        main?: {
          App?: Record<string, unknown>;
        };
      };
    }
  ).go = {
    main: {
      App: {
        GetStatus: vi.fn().mockResolvedValue({ phase: "stopped" }),
        GetRecentLogs: vi.fn().mockResolvedValue([]),
        GetBasicProxySettings: vi.fn().mockResolvedValue({
          host: "127.0.0.1",
          port: 10808,
          type: "socks5"
        }),
        GetOnboardingState: vi.fn().mockResolvedValue({
          configExists: false,
          isDefaultProxyConfig: true,
          shouldShowOnboarding: true
        })
      }
    }
  };

  render(<App />);

  expect(await screen.findByRole("heading", { name: "欢迎使用 Aether" })).toBeInTheDocument();
});

it("shows reminder banner after skipping onboarding", async () => {
  (
    window as Window & {
      go?: {
        main?: {
          App?: Record<string, unknown>;
        };
      };
    }
  ).go = {
    main: {
      App: {
        GetStatus: vi.fn().mockResolvedValue({ phase: "stopped" }),
        GetRecentLogs: vi.fn().mockResolvedValue([]),
        GetBasicProxySettings: vi.fn().mockResolvedValue({
          host: "127.0.0.1",
          port: 10808,
          type: "socks5"
        }),
        GetOnboardingState: vi.fn().mockResolvedValue({
          configExists: false,
          isDefaultProxyConfig: true,
          shouldShowOnboarding: true
        })
      }
    }
  };

  render(<App />);

  fireEvent.click(await screen.findByRole("button", { name: "暂时跳过" }));

  await waitFor(() =>
    expect(screen.queryByRole("heading", { name: "欢迎使用 Aether" })).not.toBeInTheDocument()
  );
  expect(screen.getByText(/尚未完成首次代理配置/)).toBeInTheDocument();
});

it("reopens onboarding from the reminder banner", async () => {
  (
    window as Window & {
      go?: {
        main?: {
          App?: Record<string, unknown>;
        };
      };
    }
  ).go = {
    main: {
      App: {
        GetStatus: vi.fn().mockResolvedValue({ phase: "stopped" }),
        GetRecentLogs: vi.fn().mockResolvedValue([]),
        GetBasicProxySettings: vi.fn().mockResolvedValue({
          host: "127.0.0.1",
          port: 10808,
          type: "socks5"
        }),
        GetOnboardingState: vi.fn().mockResolvedValue({
          configExists: false,
          isDefaultProxyConfig: true,
          shouldShowOnboarding: true
        })
      }
    }
  };

  render(<App />);

  fireEvent.click(await screen.findByRole("button", { name: "暂时跳过" }));
  fireEvent.click(screen.getByRole("button", { name: "继续配置" }));

  expect(await screen.findByRole("heading", { name: "欢迎使用 Aether" })).toBeInTheDocument();
});

it("hides onboarding and reminder after onboarding save succeeds", async () => {
  (
    window as Window & {
      go?: {
        main?: {
          App?: Record<string, unknown>;
        };
      };
    }
  ).go = {
    main: {
      App: {
        GetStatus: vi.fn().mockResolvedValue({ phase: "stopped" }),
        GetRecentLogs: vi.fn().mockResolvedValue([]),
        GetBasicProxySettings: vi.fn().mockResolvedValue({
          host: "127.0.0.1",
          port: 10808,
          type: "socks5"
        }),
        GetOnboardingState: vi.fn().mockResolvedValue({
          configExists: false,
          isDefaultProxyConfig: true,
          shouldShowOnboarding: true
        }),
        SaveBasicProxySettings: vi.fn().mockResolvedValue({
          settings: { host: "10.0.0.2", port: 7890, type: "http" },
          requiresRestart: false
        })
      }
    }
  };

  render(<App />);

  fireEvent.click(await screen.findByRole("button", { name: "开始配置" }));
  const dialog = await screen.findByRole("dialog");
  fireEvent.change(within(dialog).getByLabelText(/代理地址/), { target: { value: "10.0.0.2" } });
  fireEvent.change(within(dialog).getByLabelText(/代理端口/), { target: { value: "7890" } });
  fireEvent.change(within(dialog).getByLabelText(/代理类型/), { target: { value: "http" } });
  fireEvent.click(within(dialog).getByRole("button", { name: "保存并进入主界面" }));

  await waitFor(() =>
    expect(screen.queryByRole("heading", { name: "欢迎使用 Aether" })).not.toBeInTheDocument()
  );
  expect(screen.queryByText(/尚未完成首次代理配置/)).not.toBeInTheDocument();
});
