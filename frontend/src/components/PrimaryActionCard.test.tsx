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

  const button = screen.getByRole("button", { name: "正在启动" });
  expect(button).toBeDisabled();
  expect(screen.getByText("需要 UAC 授权")).toBeInTheDocument();
  expect(screen.getByText(/启动后即可在日志页查看实时输出/)).toBeInTheDocument();
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
  expect(screen.getByText("支持托盘停止")).toBeInTheDocument();
  expect(screen.getByText(/如需切换代理地址/)).toBeInTheDocument();
});
