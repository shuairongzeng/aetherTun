import { describe, expect, it } from "vitest";
import type { LogEntry } from "../types";
import { LOG_BUFFER_LIMIT, mergeLogEntries } from "./runtimeLogBuffer";

function buildLog(index: number): LogEntry {
  return {
    time: new Date(Date.UTC(2026, 2, 7, 0, 0, index)).toISOString(),
    level: "info",
    source: "core",
    message: `log-${index}`
  };
}

describe("mergeLogEntries", () => {
  it("appends only new log entries across refreshes", () => {
    const firstBatch = [buildLog(1), buildLog(2)];
    const secondBatch = [buildLog(2), buildLog(3)];

    expect(mergeLogEntries(firstBatch, secondBatch)).toEqual([
      buildLog(1),
      buildLog(2),
      buildLog(3)
    ]);
  });

  it("clears the displayed logs when the buffer grows beyond 1000 entries", () => {
    const current = Array.from({ length: LOG_BUFFER_LIMIT }, (_, index) => buildLog(index));
    const incoming = [buildLog(1001), buildLog(1002)];

    expect(mergeLogEntries(current, incoming)).toEqual(incoming);
  });
});
