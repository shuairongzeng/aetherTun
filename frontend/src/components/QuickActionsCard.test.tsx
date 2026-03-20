import { render, screen } from "@testing-library/react";
import { vi } from "vitest";
import { QuickActionsCard } from "./QuickActionsCard";

it("renders quick action items with auto-start disabled", () => {
  render(
    <QuickActionsCard
      autoStartEnabled={false}
      onOpenConfigFile={vi.fn().mockResolvedValue(undefined)}
      onOpenLogDirectory={vi.fn().mockResolvedValue(undefined)}
      onToggleAutoStart={vi.fn().mockResolvedValue(undefined)}
    />
  );

  expect(screen.getByText("快捷操作")).toBeInTheDocument();
  expect(screen.getByText("打开配置文件")).toBeInTheDocument();
  expect(screen.getByText("查看日志目录")).toBeInTheDocument();
  expect(screen.getByText("开机自启")).toBeInTheDocument();
  expect(screen.getByText("未启用，点击开启")).toBeInTheDocument();
});

it("renders auto-start enabled state", () => {
  render(
    <QuickActionsCard
      autoStartEnabled={true}
      onOpenConfigFile={vi.fn().mockResolvedValue(undefined)}
      onOpenLogDirectory={vi.fn().mockResolvedValue(undefined)}
      onToggleAutoStart={vi.fn().mockResolvedValue(undefined)}
    />
  );

  expect(screen.getByText("已启用，点击关闭")).toBeInTheDocument();
});
