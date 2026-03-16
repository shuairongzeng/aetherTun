import { useEffect, useRef, useState } from "react";
import type { LogEntry } from "../types";

type RecentLogsCardProps = {
  entries: LogEntry[];
  emptyText?: string;
};

const stickToBottomThreshold = 24;

const timeFormatter = new Intl.DateTimeFormat("zh-CN", {
  hour: "2-digit",
  minute: "2-digit",
  second: "2-digit"
});

function formatTime(value?: string): string {
  if (!value) {
    return "实时";
  }

  const parsed = new Date(value);
  if (Number.isNaN(parsed.valueOf())) {
    return value;
  }

  return timeFormatter.format(parsed);
}

function formatLevel(level?: string): string {
  if (level === "error") {
    return "错误";
  }

  return "信息";
}

function isNearBottom(element: HTMLElement): boolean {
  const distanceToBottom = element.scrollHeight - element.clientHeight - element.scrollTop;
  return distanceToBottom <= stickToBottomThreshold;
}

function scrollToBottom(element: HTMLElement) {
  const top = Math.max(0, element.scrollHeight - element.clientHeight);

  if (typeof element.scrollTo === "function") {
    element.scrollTo({ top, behavior: "smooth" });
    return;
  }

  element.scrollTop = top;
}

export function RecentLogsCard({ entries, emptyText }: RecentLogsCardProps) {
  const scrollerRef = useRef<HTMLDivElement>(null);
  const shouldStickToBottomRef = useRef(true);
  const previousCountRef = useRef(0);
  const [showJumpButton, setShowJumpButton] = useState(false);

  useEffect(() => {
    const element = scrollerRef.current;
    if (!element) {
      return;
    }

    if (entries.length === 0) {
      previousCountRef.current = 0;
      setShowJumpButton(false);
      return;
    }

    const hasNewEntries = entries.length > previousCountRef.current;
    previousCountRef.current = entries.length;

    if (hasNewEntries && shouldStickToBottomRef.current) {
      scrollToBottom(element);
      setShowJumpButton(false);
      return;
    }

    setShowJumpButton(!shouldStickToBottomRef.current);
  }, [entries]);

  const handleScroll = () => {
    const element = scrollerRef.current;
    if (!element) {
      return;
    }

    const nearBottom = isNearBottom(element);
    shouldStickToBottomRef.current = nearBottom;
    setShowJumpButton(entries.length > 0 && !nearBottom);
  };

  const handleJumpToBottom = () => {
    const element = scrollerRef.current;
    if (!element) {
      return;
    }

    shouldStickToBottomRef.current = true;
    scrollToBottom(element);
    setShowJumpButton(false);
  };

  return (
    <article className="panel panel--logs">
      <div className="panel-header">
        <div>
          <p className="eyebrow eyebrow--subtle">运行日志</p>
          <h2>实时输出</h2>
          <p className="panel-caption">核心输出会直接同步到这里，避免再弹出命令行窗口。</p>
          <div className="log-legend-row">
            <span className="log-legend">界面内嵌日志</span>
            <span className="log-legend">滚轮可滚动</span>
          </div>
        </div>
        <span className="log-count">{entries.length} 条</span>
      </div>

      <div className="log-scroller-wrap">
        <div
          ref={scrollerRef}
          className="log-scroller"
          role="log"
          aria-live="polite"
          aria-label="运行日志"
          onScroll={handleScroll}
        >
          {entries.length === 0 ? (
            <div className="log-empty-state">
              <strong className="log-empty-state__title">等待新的运行日志</strong>
              <p className="log-empty">{emptyText ?? "暂无日志"}</p>
            </div>
          ) : (
            <ul className="log-list">
              {entries.map((entry, index) => (
                <li
                  key={`${entry.time ?? "log"}-${entry.source ?? "core"}-${entry.message}-${index}`}
                  className="log-entry"
                >
                  <div className="log-entry__meta">
                    <span className="log-time">{formatTime(entry.time)}</span>
                    <span className="log-source">{entry.source || "core"}</span>
                    <span className={`log-level log-level--${entry.level || "info"}`}>
                      {formatLevel(entry.level)}
                    </span>
                  </div>
                  <div className="log-entry__message">{entry.message}</div>
                </li>
              ))}
            </ul>
          )}
        </div>

        {showJumpButton ? (
          <button className="log-jump-button" type="button" onClick={handleJumpToBottom}>
            回到底部
          </button>
        ) : null}
      </div>
    </article>
  );
}
