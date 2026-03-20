import type { LogEntry } from "../types";
import { RecentLogsCard } from "../components/RecentLogsCard";

type LogsPageProps = {
  entries: LogEntry[];
  emptyText?: string;
  onOpenLogDirectory: () => Promise<void>;
};

export function LogsPage({ entries, emptyText, onOpenLogDirectory }: LogsPageProps) {
  return (
    <div className="page">
      <div className="logs-layout">
        <div className="logs-header">
          <div className="logs-header__left">
            <h2 className="logs-header__title">实时日志</h2>
            <span className="log-count">{entries.length} 条</span>
          </div>
          <button className="btn-secondary" type="button" onClick={() => void onOpenLogDirectory()}>
            打开日志目录
          </button>
        </div>
        <p className="logs-caption">核心输出会直接同步到这里，避免再弹出命令行窗口。</p>
        <RecentLogsCard entries={entries} emptyText={emptyText} />
      </div>
    </div>
  );
}
