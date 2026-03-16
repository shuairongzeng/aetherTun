import { render, screen } from "@testing-library/react";
import { vi } from "vitest";
import { PrimaryActionCard } from "./PrimaryActionCard";

it("marks the primary button as disabled while starting", () => {
  render(
    <PrimaryActionCard
      phase="starting"
      busy={false}
      onStart={vi.fn().mockResolvedValue(undefined)}
      onStop={vi.fn().mockResolvedValue(undefined)}
    />
  );

  expect(screen.getByText("主操作")).toBeInTheDocument();
  const button = screen.getByRole("button", { name: "正在启动" });
  expect(button).toBeDisabled();
  expect(button).toHaveClass("primary-button--disabled");
  expect(screen.getByText("正在等待后台核心响应，请稍候。")).toBeInTheDocument();
  expect(screen.getByText("启动后即可在右侧日志区查看实时输出。")).toBeInTheDocument();
  expect(screen.getByText("需要 UAC 授权")).toBeInTheDocument();
});

it("shows a connected message while running", () => {
  render(
    <PrimaryActionCard
      phase="running"
      busy={false}
      onStart={vi.fn().mockResolvedValue(undefined)}
      onStop={vi.fn().mockResolvedValue(undefined)}
    />
  );

  expect(screen.getByRole("button", { name: "停止代理" })).toBeEnabled();
  expect(screen.getByText("后台核心已连接，代理正在运行。")).toBeInTheDocument();
  expect(screen.getByText("如需切换代理地址，先保存配置再重启。")).toBeInTheDocument();
  expect(screen.getByText("支持托盘停止")).toBeInTheDocument();
});
