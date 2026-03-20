export type TabId = "overview" | "settings" | "logs";

type TabBarProps = {
    activeTab: TabId;
    onTabChange: (tab: TabId) => void;
    statusPhase: string;
};

const tabs: { id: TabId; label: string }[] = [
    { id: "overview", label: "概览" },
    { id: "settings", label: "设置" },
    { id: "logs", label: "日志" }
];

function phaseToClass(phase: string): string {
    switch (phase) {
        case "running":
            return "status-dot--running";
        case "starting":
            return "status-dot--starting";
        case "stopping":
            return "status-dot--stopping";
        case "error":
            return "status-dot--error";
        default:
            return "status-dot--stopped";
    }
}

export function TabBar({ activeTab, onTabChange, statusPhase }: TabBarProps) {
    return (
        <nav className="tab-bar" role="navigation" aria-label="主导航">
            <span className="tab-bar__brand">Aether</span>

            <div className="tab-bar__tabs" role="tablist">
                {tabs.map((tab) => (
                    <button
                        key={tab.id}
                        className={`tab-button${activeTab === tab.id ? " tab-button--active" : ""}`}
                        role="tab"
                        aria-selected={activeTab === tab.id}
                        onClick={() => onTabChange(tab.id)}
                        type="button"
                    >
                        {tab.label}
                    </button>
                ))}
            </div>

            <div className="tab-bar__status">
                <span className={`status-dot ${phaseToClass(statusPhase)}`} aria-label={`状态：${statusPhase}`} />
            </div>
        </nav>
    );
}
