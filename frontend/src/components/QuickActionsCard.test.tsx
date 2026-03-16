import { render, screen } from "@testing-library/react";
import { vi } from "vitest";
import { QuickActionsCard } from "./QuickActionsCard";

it("renders descriptive quick action items", () => {
  render(
    <QuickActionsCard
      onOpenConfigFile={vi.fn().mockResolvedValue(undefined)}
      onOpenLogDirectory={vi.fn().mockResolvedValue(undefined)}
      onToggleAutoStart={vi.fn().mockResolvedValue(undefined)}
    />
  );

  expect(screen.getByText("常用入口")).toBeInTheDocument();
  expect(screen.getByText("快速打开真实配置文件进行高级编辑。")).toBeInTheDocument();
  expect(screen.getByText("直接跳到日志目录，便于排查问题。")).toBeInTheDocument();
  expect(screen.getByText("预留开机自启入口，后续会接入系统注册。")).toBeInTheDocument();
});
