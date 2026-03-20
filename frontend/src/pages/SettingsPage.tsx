import type { BasicProxySettings, BasicProxyStatus, SaveBasicProxySettingsResult } from "../types";
import { BasicProxyConfigCard } from "../components/BasicProxyConfigCard";

type SettingsPageProps = {
    value: BasicProxySettings;
    dirty: boolean;
    saving: boolean;
    errors: Partial<Record<keyof BasicProxySettings, string>>;
    status: BasicProxyStatus;
    onChange: (next: BasicProxySettings) => void;
    onSave: () => void | Promise<void>;
    onResetDefaults: () => void | Promise<void>;
    onOpenConfigFile: () => void | Promise<void>;
};

export function SettingsPage(props: SettingsPageProps) {
    return (
        <div className="page">
            <div className="settings-layout">
                <BasicProxyConfigCard
                    value={props.value}
                    dirty={props.dirty}
                    saving={props.saving}
                    errors={props.errors}
                    status={props.status}
                    onChange={props.onChange}
                    onSave={props.onSave}
                    onResetDefaults={props.onResetDefaults}
                    onOpenConfigFile={props.onOpenConfigFile}
                />
            </div>
        </div>
    );
}
