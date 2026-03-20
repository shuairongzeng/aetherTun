import { fireEvent, render, screen } from "@testing-library/react";
import type { LogEntry } from "../types";
import { RecentLogsCard } from "./RecentLogsCard";

function buildEntry(index: number): LogEntry {
  return {
    time: new Date(Date.UTC(2026, 2, 7, 0, 0, index)).toISOString(),
    level: "info",
    source: "core",
    message: `log-${index}`
  };
}

function configureScroller(
  element: HTMLElement,
  measurements: { scrollHeight: number; clientHeight: number; scrollTop: number }
) {
  Object.defineProperty(element, "scrollHeight", {
    configurable: true,
    get: () => measurements.scrollHeight
  });
  Object.defineProperty(element, "clientHeight", {
    configurable: true,
    get: () => measurements.clientHeight
  });
  Object.defineProperty(element, "scrollTop", {
    configurable: true,
    get: () => measurements.scrollTop,
    set: (value: number) => {
      measurements.scrollTop = value;
    }
  });

  element.scrollTo = ({ top }: ScrollToOptions) => {
    measurements.scrollTop = top ?? measurements.scrollTop;
  };
}

it("sticks to the bottom when new logs arrive and the user is already at the bottom", () => {
  const initialEntries = [buildEntry(1), buildEntry(2)];
  const nextEntries = [...initialEntries, buildEntry(3)];
  const measurements = { scrollHeight: 240, clientHeight: 100, scrollTop: 140 };

  const { rerender } = render(<RecentLogsCard entries={initialEntries} />);
  const scroller = screen.getByRole("log", { name: "运行日志" });
  configureScroller(scroller, measurements);

  measurements.scrollHeight = 320;
  rerender(<RecentLogsCard entries={nextEntries} />);

  expect(scroller.scrollTop).toBe(220);
});

it("preserves manual scroll position when the user has scrolled up", () => {
  const initialEntries = [buildEntry(1), buildEntry(2)];
  const nextEntries = [...initialEntries, buildEntry(3)];
  const measurements = { scrollHeight: 240, clientHeight: 100, scrollTop: 60 };

  const { rerender } = render(<RecentLogsCard entries={initialEntries} />);
  const scroller = screen.getByRole("log", { name: "运行日志" });
  configureScroller(scroller, measurements);

  fireEvent.scroll(scroller);

  measurements.scrollHeight = 320;
  rerender(<RecentLogsCard entries={nextEntries} />);

  expect(scroller.scrollTop).toBe(60);
});

it("shows a jump button when the user scrolls away from the bottom", () => {
  const entries = [buildEntry(1), buildEntry(2), buildEntry(3)];
  const measurements = { scrollHeight: 320, clientHeight: 100, scrollTop: 60 };

  render(<RecentLogsCard entries={entries} />);
  const scroller = screen.getByRole("log", { name: "运行日志" });
  configureScroller(scroller, measurements);

  fireEvent.scroll(scroller);

  expect(screen.getByRole("button", { name: /回到最新/ })).toBeInTheDocument();
});

it("scrolls back to the bottom and hides the jump button after clicking it", () => {
  const entries = [buildEntry(1), buildEntry(2), buildEntry(3)];
  const measurements = { scrollHeight: 320, clientHeight: 100, scrollTop: 60 };

  render(<RecentLogsCard entries={entries} />);
  const scroller = screen.getByRole("log", { name: "运行日志" });
  configureScroller(scroller, measurements);

  fireEvent.scroll(scroller);
  fireEvent.click(screen.getByRole("button", { name: /回到最新/ }));

  expect(scroller.scrollTop).toBe(220);
  expect(screen.queryByRole("button", { name: /回到最新/ })).not.toBeInTheDocument();
});

it("renders a stronger empty state when there are no log entries", () => {
  render(<RecentLogsCard entries={[]} emptyText="后台核心未启动，启动后会在这里显示运行日志。" />);

  expect(screen.getByText("等待新的运行日志")).toBeInTheDocument();
  expect(screen.getByText("后台核心未启动，启动后会在这里显示运行日志。")).toBeInTheDocument();
});
