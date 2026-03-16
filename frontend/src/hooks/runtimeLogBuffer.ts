import type { LogEntry } from "../types";

export const LOG_BUFFER_LIMIT = 1000;
export const LOG_FETCH_LIMIT = 200;

function normalizeEntry(entry: LogEntry): LogEntry {
  return {
    time: entry.time || "",
    level: entry.level || "",
    source: entry.source || "",
    message: entry.message || ""
  };
}

function logEntryKey(entry: LogEntry): string {
  return [entry.time || "", entry.level || "", entry.source || "", entry.message].join("\u0000");
}

function compactEntries(entries: LogEntry[]): LogEntry[] {
  const result: LogEntry[] = [];
  const seen = new Set<string>();

  for (const rawEntry of entries) {
    const entry = normalizeEntry(rawEntry);
    if (!entry.message.trim()) {
      continue;
    }

    const key = logEntryKey(entry);
    if (seen.has(key)) {
      continue;
    }

    seen.add(key);
    result.push(entry);
  }

  return result;
}

export function mergeLogEntries(current: LogEntry[], incoming: LogEntry[], limit = LOG_BUFFER_LIMIT): LogEntry[] {
  const existingEntries = compactEntries(current);
  const incomingEntries = compactEntries(incoming);

  if (incomingEntries.length === 0) {
    return existingEntries;
  }

  const merged = [...existingEntries];
  const seen = new Set(merged.map((entry) => logEntryKey(entry)));

  for (const entry of incomingEntries) {
    const key = logEntryKey(entry);
    if (seen.has(key)) {
      continue;
    }

    seen.add(key);
    merged.push(entry);
  }

  if (merged.length > limit) {
    return incomingEntries.slice(-limit);
  }

  return merged;
}
